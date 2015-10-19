// Package swarm provides the cluster state and tasks that are going to be
// load balanced in the cluster.
package swarm

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ahmetalpbalkan/wagl/task"
)

const (
	dnsLabel  = "dns.service"
	dnsDomain = "dns.domain"
)

var (
	defaultTimeout = time.Second * 30
)

type Swarm struct {
	client *http.Client
	url    *url.URL
}

// container represents a container item in /containers/json Endpoint of Docker
// Remote API
type container struct {
	Id     string            `json:"Id"`
	Ports  []containerPort   `json:"Ports"`
	Names  []string          `json:"Names"`
	Labels map[string]string `json:"Labels"`
}

// containerPort represents a port declaration item as it appears in Docker
// Remote API /containers/json.
type containerPort struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

// New constructs a client to access a Docker Swarm cluster state. If the cluster
// does not use TLS, tlsConfig must be nil.
func New(swarmUrl string, tlsConfig *tls.Config) (*Swarm, error) {
	u, err := url.Parse(swarmUrl)
	if err != nil {
		return nil, err
	}

	// Convert unix:// to http(s)://
	if u.Scheme == "" || u.Scheme == "tcp" {
		if tlsConfig == nil {
			u.Scheme = "http"
		} else {
			u.Scheme = "https"
		}
	}

	cl, err := httpClient(u, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("error initializing HTTP client for Docker: %v", err)
	}

	return &Swarm{
		client: cl,
		url:    u,
	}, nil
}

// httpClient provides an HTTP client to make requests to the Docker API.
// The code is mostly copied from https://github.com/samalba/dockerclient/
// instead of copying the entire package for one method. Please check the
// project's license at: https://github.com/samalba/dockerclient/blob/master/LICENSE
func httpClient(u *url.URL, tlsConfig *tls.Config) (*http.Client, error) {
	httpTransport := &http.Transport{TLSClientConfig: tlsConfig}

	// Choose between Unix and TCP clients
	switch u.Scheme {
	default:
		httpTransport.Dial = func(proto, addr string) (net.Conn, error) {
			return net.DialTimeout(proto, addr, defaultTimeout)
		}
	case "unix":
		socketPath := u.Path
		unixDial := func(proto, addr string) (net.Conn, error) {
			return net.DialTimeout("unix", socketPath, defaultTimeout)
		}
		httpTransport.Dial = unixDial
		// Override the main URL object so the HTTP lib won't complain
		u.Scheme = "http"
		u.Host = "unix.sock"
		u.Path = ""
	}
	return &http.Client{Transport: httpTransport}, nil
}

// Tasks provides running containers in a Swarm cluster.
func (s *Swarm) Tasks() (task.ClusterState, error) {
	ll, err := s.listContainers()
	if err != nil {
		return nil, err
	}

	out, err := containersToTasks(ll)
	if err != nil {
		return nil, err
	}
	return out, err
}

// listContainers returns list of running containers from Docker API
func (s *Swarm) listContainers() ([]container, error) {
	url := strings.TrimSuffix(s.url.String(), "/")
	req, err := http.NewRequest("GET", url+"/containers/json?all=false", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating the HTTP request: %v", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Docker API error (Status: %d) Body: %q", resp.Status, data)
	}

	var ll []container
	if err := json.Unmarshal(data, &ll); err != nil {
		return nil, fmt.Errorf("Error unmarshaling response: %v", err)
	}
	return ll, nil
}

// containersToTasks strips out unnecessary info from Container type and
// makes task.Task instances out of given list.
func containersToTasks(ll []container) ([]task.Task, error) {
	out := make([]task.Task, len(ll))
	for i, c := range ll {
		ports, err := mappedPorts(c.Ports)
		if err != nil {
			return nil, fmt.Errorf("error parsing ports for container %s (%v): %v", c.Id, c.Names, err)
		}
		srv, dom := dnsPartsFromLabels(c.Labels)
		out[i] = task.Task{
			Id:      c.Id,
			Ports:   ports,
			Service: srv,
			Domain:  dom,
		}
	}
	return out, nil
}

// dnsPartsFromLabels gives service name and domain name (if
// applicable) based which are going to be used in the DNS Resource Records as
// part of the FQDN. If the container is not configured or does not have enough
// info to resolve the service name, both return values will be empty string.
//
// Domain name can be used as project name to categorize many services that
// belong to a project in a FQDN like service.domain.swarm.
//
// In Docker, container labels dns.service and dns.domain are used to come up
// with DNS records for FQDNs like api.swarm., _api._tcp.swarm (no dns.domain
// specified), api.billing.swarm. and _api._tcp.billing.swarm. where dns.domain
// is specified as "billing" and dns.service is specified as "api".
//
// These labels are case insensitive and invalid characters (per DNS spec)
// would cause no DNS records to be generated for these services.
func dnsPartsFromLabels(labels map[string]string) (string, string) {
	var (
		service = strings.ToLower(labels[dnsLabel])
		project = strings.ToLower(labels[dnsDomain])
	)
	if service == "" { // does not make sense to have a project name w/o service
		project = ""
	}
	return service, project
}

// mappedPorts returns only list of ports mapped to the host from a list of
// port mappings.
func mappedPorts(l []containerPort) ([]task.Port, error) {
	out := make([]task.Port, 0)
	for _, v := range l {
		if isMappedPort(v) {
			p, err := toPort(v)
			if err != nil {
				return nil, err
			}
			out = append(out, p)
		}
	}
	return out, nil
}

// isMappedPort determines if a port listing is actually mapped to the host.
func isMappedPort(l containerPort) bool {
	return l.IP != "" && l.PublicPort != 0
}

// toPort converts Docker port mapping to task.Port. p must be mapped to host.
func toPort(p containerPort) (task.Port, error) {
	ip := net.ParseIP(p.IP)
	if ip == nil {
		return task.Port{}, fmt.Errorf("cannot parse IP '%s'", p.IP)
	}
	return task.Port{
		HostIP:   ip,
		HostPort: p.PublicPort,
		Proto:    p.Type,
	}, nil
}
