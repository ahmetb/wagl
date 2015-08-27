package swarm

import (
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"wagl/task"
)

func TestGetTasks(t *testing.T) {
	srv := testServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// implement partial /info endpoint
		b := `[
			{
				"Id": "nginx",
				"Labels": {
					"dns.domain":  "bilLING",
					"dns.service": "API"
				},
				"Ports": [
					{
						"IP":          "192.168.99.103",
						"PrivatePort": 80,
						"PublicPort":  8000,
						"Type":        "tcp"
					},
					{
						"IP":          "",
						"PrivatePort": 443,
						"PublicPort":  0,
						"Type":        "tcp"
					}
				]
			},
			{
				"Id": "no-ports-but-has-labels",
				"Labels": {
					"dns.domain":  "billing",
					"dns.service": "db"
				}
			}
		]`
		w.Write([]byte(b))
	}))
	defer srv.Close()

	sw, err := New(srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	out, err := sw.Tasks()
	if err != nil {
		t.Fatal(err)
	}

	expected := task.ClusterState([]task.Task{
		{
			Id:      "nginx",
			Service: "api",
			Domain:  "billing",
			Ports: []task.Port{{
				HostIP:   net.IPv4(192, 168, 99, 103),
				HostPort: 8000,
				Proto:    "tcp",
			}},
		},
		{
			Id:      "no-ports-but-has-labels",
			Service: "db",
			Domain:  "billing",
			Ports:   []task.Port{},
		},
	})
	if !reflect.DeepEqual(out, expected) {
		t.Fatalf("got wrong value.\nexpected: %#v\ngot:%#v", expected, out)
	}
}

func Test_isMappedPort(t *testing.T) {
	cases := []struct {
		in  containerPort
		out bool
	}{
		{containerPort{
			IP:          "192.168.99.103",
			PrivatePort: 80,
			PublicPort:  80,
			Type:        "tcp",
		}, true},
		{containerPort{
			IP:          "",
			PrivatePort: 443,
			PublicPort:  0,
			Type:        "tcp",
		}, false},
	}

	for i, c := range cases {
		if o := isMappedPort(c.in); o != c.out {
			t.Fatal("wrong value for case %d", i)
		}
	}
}

func Test_mappedPorts(t *testing.T) {
	in := []containerPort{
		{
			IP:          "192.168.99.103",
			PrivatePort: 80,
			PublicPort:  8000,
			Type:        "tcp",
		},
		{
			IP:          "",
			PrivatePort: 443,
			PublicPort:  0,
			Type:        "tcp",
		},
	}

	o, err := mappedPorts(in)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o, []task.Port{{
		HostIP:   net.IPv4(192, 168, 99, 103),
		HostPort: 8000,
		Proto:    "tcp",
	}}) {
		t.Fatal("got wrong mappings: %#v", o)
	}
}

func Test_toPort(t *testing.T) {
	p, err := toPort(containerPort{
		IP:          "192.168.99.103",
		PrivatePort: 80,
		PublicPort:  8001,
		Type:        "tcp",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := task.Port{
		HostIP:   net.IPv4(192, 168, 99, 103),
		HostPort: 8001,
		Proto:    "tcp",
	}

	if !reflect.DeepEqual(p, expected) { // deep equal required: net.IP is []byte
		t.Fatal("got wrong value: %#v", p)
	}
}

func Test_toPortFails(t *testing.T) {
	badIP := "not-an-ip"
	_, err := toPort(containerPort{
		IP:          badIP,
		PrivatePort: 80,
		PublicPort:  8000,
		Type:        "tcp",
	})
	if err == nil {
		t.Fatal("IP parsing did not fail")
	} else if !strings.Contains(err.Error(), badIP) {
		t.Fatalf("error message does not contain faulty IP value: %v", err)
	}
}

func Test_containerSrvNames(t *testing.T) {
	cases := []struct {
		labels map[string]string
		srv    string
		prj    string
	}{
		{map[string]string{}, "", ""}, // empty
		{map[string]string{ // no service name
			"dns.domain": "billing"}, "", ""},
		{map[string]string{ // no framework
			"dns.service": "API"}, "api", ""},
		{map[string]string{ // both values, case-insensitivity test
			"dns.service": "API",
			"dns.domain":  "Billing"}, "api", "billing"},
	}

	for _, c := range cases {
		srv, prj := dnsPartsFromLabels(c.labels)
		if srv != c.srv {
			t.Fatalf("wrong service name. expected: '%s', got: '%s'", c.srv, srv)
		}
		if prj != c.prj {
			t.Fatalf("wrong service name. expected: '%s', got: '%s'", c.prj, prj)
		}
	}

}

func testServer(handler http.Handler) *httptest.Server {
	s := httptest.NewServer(handler)
	return s
}
