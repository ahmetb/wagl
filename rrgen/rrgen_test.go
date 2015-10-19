package rrgen

import (
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/ahmetalpbalkan/wagl/rrstore"
	"github.com/ahmetalpbalkan/wagl/task"
	"github.com/miekg/dns"
)

func Test_insertRR(t *testing.T) {
	rr := make(rrstore.RRs)
	insertRR(rr, rrEntry{dns.TypeA, "foo.domain.", "10.0.0.1"})
	insertRR(rr, rrEntry{dns.TypeA, "foo.domain.", "10.0.0.2"})
	insertRR(rr, rrEntry{dns.TypeSRV, "_foo._tcp.domain.", "10.0.0.3:3000"})

	expected := rrstore.RRs(map[uint16]map[string][]string{
		dns.TypeA:   {"foo.domain.": []string{"10.0.0.1", "10.0.0.2"}},
		dns.TypeSRV: {"_foo._tcp.domain.": []string{"10.0.0.3:3000"}}})

	if !reflect.DeepEqual(expected, rr) {
		t.Fatalf("wrong value.\nexpected=%#v\ngot=%#v", expected, rr)
	}
}

func Test_getTaskRRs(t *testing.T) {
	cases := []struct {
		t  task.Task
		rs []string
	}{
		// Task with no domain
		{task.Task{
			Service: "foo",
			Ports: []task.Port{
				{
					HostIP:   net.IPv4(10, 0, 0, 1),
					HostPort: 8000,
					Proto:    "tcp",
				},
			}},
			[]string{
				"A foo.domain. 10.0.0.1",
				"SRV _foo._tcp.domain. 10.0.0.1:8000",
			}},

		// Task with project domain and multiple ports
		{task.Task{
			Service: "api",
			Domain:  "billing",
			Ports: []task.Port{
				{
					HostIP:   net.IPv4(10, 0, 0, 2),
					HostPort: 8001,
					Proto:    "tcp",
				},
				{
					HostIP:   net.IPv4(10, 0, 0, 2),
					HostPort: 8002,
					Proto:    "udp",
				},
			}},
			[]string{
				"A api.billing.domain. 10.0.0.2",
				"SRV _api._tcp.billing.domain. 10.0.0.2:8001",
				"SRV _api._udp.billing.domain. 10.0.0.2:8002",
			}},
	}

	for _, c := range cases {
		rrs := getTaskRRs("domain", c.t)
		ll := make([]string, len(rrs))
		for i := range rrs {
			ll[i] = rrs[i].String()
		}

		in := strings.Join(c.rs, "\n")
		out := strings.Join(ll, "\n")

		if in != out {
			t.Fatalf("wrong RRs.\nexpected: '%s'\ngot: '%s'", in, out)
		}
	}
}

func Test_RRs_empty(t *testing.T) {
	rr := getRRs("domain", nil)
	if len(rr) > 0 {
		t.Fatal("output has records")
	}

	rr = RRs("domain", task.ClusterState([]task.Task{
		{
			Id:      "no-ports",
			Service: "api",
			Ports:   []task.Port{},
		},
		{
			Id:    "no-service-name",
			Ports: []task.Port{{net.IPv4(10, 0, 0, 2), 8001, "tcp"}},
		},
	}))
	if len(rr) > 0 {
		t.Fatal("output has records")
	}
}

func Test_RRs_actualWorkload(t *testing.T) {
	rr := RRs("domain", task.ClusterState([]task.Task{
		{
			Id:      "bind",
			Service: "dns",
			Domain:  "infra",
			Ports:   []task.Port{{net.IPv4(192, 168, 0, 3), 53, "udp"}},
		},
		{
			Id:      "web1",
			Service: "api",
			Ports:   []task.Port{{net.IPv4(192, 168, 0, 1), 8000, "tcp"}},
		},
		{
			Id:      "web2",
			Service: "api",
			Ports: []task.Port{
				{net.IPv4(192, 168, 0, 2), 8000, "tcp"},
				{net.IPv4(192, 168, 0, 2), 5000, "udp"},
			},
		},
		{
			Id:      "nginx",
			Service: "frontend",
			Domain:  "blog",
			Ports: []task.Port{
				{net.IPv4(192, 168, 0, 3), 8000, "tcp"},
			},
		},
		{ // no proto on port
			Id:      "debian",
			Service: "test",
			Ports:   []task.Port{{net.IPv4(192, 168, 0, 3), 500, ""}},
		},
		{ // no service name
			Id:    "debian",
			Ports: []task.Port{{net.IPv4(192, 168, 0, 3), 500, "udp"}},
		},
	}))

	expected := rrstore.RRs(map[uint16]map[string][]string{
		dns.TypeA: {
			"dns.infra.domain.":     []string{"192.168.0.3"},
			"api.domain.":           []string{"192.168.0.1", "192.168.0.2"},
			"frontend.blog.domain.": []string{"192.168.0.3"},
		},
		dns.TypeSRV: {
			"_dns._udp.infra.domain.":     []string{"192.168.0.3:53"},
			"_api._tcp.domain.":           []string{"192.168.0.1:8000", "192.168.0.2:8000"},
			"_api._udp.domain.":           []string{"192.168.0.2:5000"},
			"_frontend._tcp.blog.domain.": []string{"192.168.0.3:8000"},
		}})

	if !reflect.DeepEqual(rr, expected) {
		t.Fatalf("wrong value.\nexp: %#v\ngot: %#v", expected, rr)
	}
}
