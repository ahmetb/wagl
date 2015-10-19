package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/ahmetalpbalkan/wagl/clusterdns"
	"github.com/ahmetalpbalkan/wagl/rrstore"
	"github.com/ahmetalpbalkan/wagl/server"
	"github.com/ahmetalpbalkan/wagl/swarm"

	"github.com/codegangsta/cli"
)

const (
	defaultDnsDomain       = "swarm."
	defaultAddr            = ":53"
	defaultSwarm           = "127.0.0.1:2376"
	defaultRefreshInterval = time.Second * 15
	defaultRefreshTimeout  = time.Second * 10
	defaultStalenessPeriod = time.Second * 60
)

type Options struct {
	// User input
	domain          string
	bindAddr        string
	swarmAddr       string
	tlsDir          string
	tlsVerify       bool
	recurse         bool
	nameservers     []string
	refreshInterval time.Duration
	refreshTimeout  time.Duration
	stalenessPeriod time.Duration
}

func (o *Options) String() string {
	return fmt.Sprintf(`Configuration:
 - Domain:    "%s"
 - Listen:    "%s"
 - Swarm:     %s
   - TLS:     %s (verify: %v)
 - External:  %v (ns: [%s])
 - Refresh:   Every %v (timeout: %v) (staleness: %v)
-------------------`,
		o.domain,
		o.bindAddr,
		o.swarmAddr,
		o.tlsDir,
		o.tlsVerify,
		o.recurse, strings.Join(o.nameservers, ","),
		o.refreshInterval, o.refreshTimeout, o.stalenessPeriod)
}

func main() {
	cmd := cli.NewApp()
	cmd.Name = "wagl"
	cmd.Version = "0.1"
	cmd.Usage = "DNS service discovery for Docker Swarm clusters"
	cli.AppHelpTemplate = usageTemplate
	cmd.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "bind",
			Value: defaultAddr,
			Usage: "IP:port on which the server shoud listen",
		},
		cli.StringFlag{
			Name:  "swarm",
			Value: defaultSwarm,
			Usage: "address of the Swarm manager", // TODO accept multiple
		},
		cli.StringFlag{
			Name:   "swarm-cert-path",
			Value:  "",
			Usage:  "directory TLS certs for Swarm manager is stored",
			EnvVar: "DOCKER_CERT_PATH",
		},
		cli.BoolFlag{
			Name:   "swarm-tlsverify",
			Usage:  "verify remote Swarm's identity using TLS",
			EnvVar: "DOCKER_TLS_VERIFY",
		},
		cli.StringFlag{
			Name:  "domain",
			Value: defaultDnsDomain,
			Usage: "DNS domain (FQDN suffix) for which this server is authoritative",
		},
		cli.BoolTFlag{
			Name:  "external",
			Usage: "use external nameservers to resolve DNS requests outside the domain (true by default)",
		},
		cli.StringSliceFlag{
			Name:  "ns",
			Usage: "external nameserver(s) to forward requests (default: nameservers in /etc/resolv.conf)",
		},
		cli.DurationFlag{
			Name:  "refresh",
			Value: defaultRefreshInterval,
			Usage: "how frequently refresh DNS table from cluster records",
		},
		cli.DurationFlag{
			Name:  "refresh-timeout",
			Value: defaultRefreshTimeout,
			Usage: "time alotted for Swarm to list containers in the cluster",
		},
		cli.DurationFlag{
			Name:  "staleness",
			Value: defaultStalenessPeriod,
			Usage: "how long to serve stale DNS records before exiting",
		},
	}
	cmd.Action = func(c *cli.Context) {
		opts := &Options{
			domain:          c.String("domain"),
			bindAddr:        c.String("bind"),
			swarmAddr:       c.String("swarm"),
			tlsDir:          c.String("swarm-cert-path"),
			tlsVerify:       c.Bool("swarm-tlsverify"),
			recurse:         c.BoolT("external"),
			nameservers:     c.StringSlice("ns"),
			refreshInterval: c.Duration("refresh"),
			refreshTimeout:  c.Duration("refresh-timeout"),
			stalenessPeriod: c.Duration("staleness"),
		}
		if err := validate(opts); err != nil {
			log.Fatalf("Error: %v", err)
		}
		log.Printf("%s", opts)
		serve(opts)
	}
	cmd.Run(os.Args)
}

// validate looks for logical correctness and consistency of the input arguments
// to the program.
func validate(opt *Options) error {
	// No NS must be specified if recursion is off
	if !opt.recurse && len(opt.nameservers) > 0 {
		return errors.New("External querying disabled, but external nameservers specified")
	}

	// TLS verify can be used only if certs are specified
	if opt.tlsVerify && opt.tlsDir == "" {
		return errors.New("TLS verify specified; but not TLS cert path")
	}

	// No nameservers speficied, check resolv.conf, add it.
	if opt.recurse && len(opt.nameservers) == 0 {
		if ns, err := localNameservers(); err != nil {
			return fmt.Errorf("Failed to load nameservers list: %v", err)
		} else if len(ns) == 0 {
			return fmt.Errorf("No nameservers found in /etc/resolv.conf")
		} else {
			opt.nameservers = ns
		}
	}

	// Nameserver validations:
	// - make sure nameservers are IP[:port]
	// - add default DNS port to nameservers if missing
	for i, v := range opt.nameservers {
		host := v
		if h, _, err := net.SplitHostPort(v); err != nil { // Missing port
			opt.nameservers[i] = v + ":53"
		} else {
			host = h
		}
		// Make sure hostname is IP (do not support domain names as nameservers)
		if ip := net.ParseIP(host); ip == nil {
			return fmt.Errorf("Nameserver is not an IP address: '%s'", host)
		}
	}

	// Refresh timeout < refresh interval
	if opt.refreshTimeout >= opt.refreshInterval {
		return fmt.Errorf("Refresh timeout (%v) should be less than refresh interval (%v)", opt.refreshTimeout, opt.refreshInterval)
	}

	return nil
}

// serve starts the DNS server and blocks.
func serve(opt *Options) {
	dockerTLS, err := tlsConfig(opt.tlsDir, opt.tlsVerify)
	if err != nil {
		log.Fatalf("Error establishing TLS config: %v", err)
	}

	rrs := rrstore.New()
	cluster, err := swarm.New(opt.swarmAddr, dockerTLS)
	if err != nil {
		log.Fatalf("Error initializing Swarm: %v", err)
	}
	dns := clusterdns.New(opt.domain, rrs, cluster)

	cancel := make(chan struct{})
	defer close(cancel)
	errCh, okCh := dns.StartRefreshing(opt.refreshInterval, opt.refreshTimeout,
		cancel)

	go func() {
		var lastSuccess time.Time
		var start = time.Now()
		var errs = 0
		var ok = 0

		for {
			// Exit if records are stale. Here we prefer consistency over
			// liveliness/availability.
			if (ok > 0 && time.Since(lastSuccess) > opt.stalenessPeriod) ||
				(ok == 0 && time.Since(start) > opt.stalenessPeriod) {
				close(cancel)

				var last string
				if lastSuccess.IsZero() {
					last = "never"
				} else {
					last = lastSuccess.String()
				}
				log.Fatalf("Fatal: exiting rather than serving stale records. Staleness period: %v, last success: %s", opt.stalenessPeriod, last)
			}

			select {
			case err := <-errCh:
				errs++
				log.Printf("Refresh error (#%d): %v ", errs, err)
			case <-okCh:
				errs = 0 // reset errs
				ok++
				lastSuccess = time.Now()
				log.Printf("Successfully refreshed records.")
			case <-cancel:
				log.Fatal("Fatal: Refreshing records cancelled.")
			}
		}
	}()

	srv := server.New(opt.domain, opt.bindAddr, rrs, opt.recurse, opt.nameservers)
	log.Fatal(srv.ListenAndServe())
}
