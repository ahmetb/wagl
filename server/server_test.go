package server

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/miekg/dns"
	"wagl/rrstore"
)

func TestHandleExternalOff(t *testing.T) {
	srv, ready := testServer(t, rrstore.New())
	<-ready
	defer srv.Shutdown()

	if r, err := query(srv.Addr, "example.com", dns.TypeA); err != nil {
		t.Fatalf("exchange failed: %v", err)
	} else if r.Rcode != dns.RcodeServerFailure {
		t.Fatalf("unexpected rcode. expected=%s got=%s",
			dns.TypeToString[dns.RcodeServerFailure],
			dns.TypeToString[uint16(r.Rcode)])
	}
}

func TestHandleExternalOn(t *testing.T) {
	srv, ready := testServerExternal(t)
	<-ready
	defer srv.Shutdown()

	cases := []struct {
		fqdn          string
		qType         uint16
		expectedRCode int
	}{
		// Uppercase vs lowercase
		{"example.com", dns.TypeA, dns.RcodeSuccess},
		{"eXaMpLe.com", dns.TypeA, dns.RcodeSuccess},
		{"EXAMPLE.COM", dns.TypeA, dns.RcodeSuccess},

		// Various DNS RR classes
		{"google.com", dns.TypeA, dns.RcodeSuccess},
		{"google.com", dns.TypeNS, dns.RcodeSuccess},
		{"google.com", dns.TypeSOA, dns.RcodeSuccess},
		{"google.com", dns.TypeMX, dns.RcodeSuccess},

		// Tribute
		{"bilkent.edu.tr", dns.TypeA, dns.RcodeSuccess},

		// Non-existing domains
		{"booyakashahearmenowrepresentkeepitreal.com", dns.TypeA, dns.RcodeNameError},
	}

	for _, c := range cases {
		q := fmt.Sprintf("%s %s", dns.TypeToString[c.qType], c.fqdn)

		if r, err := query(srv.Addr, c.fqdn, c.qType); err != nil {
			t.Fatalf("exchange failed (%s): %v", q, err)
		} else if r.Rcode != c.expectedRCode {
			t.Fatalf("unexpected rcode (%s). expected=%s got=%s", q,
				dns.RcodeToString[c.expectedRCode], dns.RcodeToString[r.Rcode])
			if len(r.Answer) == 0 {
				t.Fatalf("No answers for %q", q)
			}
		}
	}
}

func TestHandleDomain(t *testing.T) {
	rr := rrstore.New()
	rr.Set(map[uint16]map[string][]string{
		dns.TypeA: {
			"api.domain.":  []string{"10.0.0.1", "10.0.0.2"},
			"blog.domain.": []string{"10.0.1.1", "10.0.1.2", "10.0.1.3"},
		},
		dns.TypeSRV: {
			"_web._tcp.domain.": []string{"10.0.0.1:80"},
			"_web._udp.domain.": []string{"10.0.0.1:5001",
				"10.0.0.2:5002",
				"10.0.0.3:5003"},
		},
	})

	srv, ready := testServer(t, rr)
	<-ready
	defer srv.Shutdown()

	cases := []struct {
		fqdn            string
		qType           uint16
		expectedRCode   int
		expectedAnswers int
	}{
		// List all test cases for all possible DNS questions here within the
		// domain.
		{"nonexistent.domain.", dns.TypeA, dns.RcodeNameError, 0},
		{"nonexistent.domain.", dns.TypeSRV, dns.RcodeNameError, 0},
		{"api.domain", dns.TypeA, dns.RcodeSuccess, 2},
		{"_web._tcp.domain", dns.TypeSRV, dns.RcodeSuccess, 1},
		{"_WEB._UDP.domain", dns.TypeSRV, dns.RcodeSuccess, 3},
	}

	for _, c := range cases {
		q := fmt.Sprintf("%s %s", dns.TypeToString[c.qType], c.fqdn)

		if r, err := query(srv.Addr, c.fqdn, c.qType); err != nil {
			t.Fatalf("exchange failed (%s): %v", q, err)
		} else if r.Rcode != c.expectedRCode {
			t.Fatalf("unexpected rcode (%s). expected=%s got=%s", q,
				dns.RcodeToString[c.expectedRCode], dns.RcodeToString[r.Rcode])
		} else if len(r.Answer) != c.expectedAnswers {
			t.Fatalf("unexpected answers count. expected=%d got=%d", q,
				len(r.Answer), c.expectedAnswers)
		}
	}
}

func TestRRShuffling(t *testing.T) {
	rr := rrstore.New()
	recs := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	rr.Set(map[uint16]map[string][]string{
		dns.TypeA: {"a.domain.": []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}}})

	srv, ready := testServer(t, rr)
	<-ready
	defer srv.Shutdown()

	n := 50
	same := true
	for i := 0; i < n; i++ {
		r, err := query(srv.Addr, "a.domain.", dns.TypeA)
		if err != nil {
			t.Fatal(err)
		}
		as := make([]string, len(r.Answer))
		for i, _ := range r.Answer {
			as[i] = r.Answer[i].(*dns.A).A.String()
			if len(as) != len(recs) {
				t.Fatalf("wrong answer count: %d", len(as))
			}
		}
		if !reflect.DeepEqual(as, recs) {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("same RR ordering occurred even after %d requests", n)
	}
}

func query(addr string, domain string, qType uint16) (*dns.Msg, error) {
	c, m := new(dns.Client), new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), qType)
	r, _, err := c.Exchange(m, addr)
	return r, err
}

// testServer gives a test server capable of serving only internal requests.
func testServer(t *testing.T, rr rrstore.RRReader) (*DnsServer, <-chan struct{}) {
	srv := New("domain", ":8053", rr, false, []string{})

	ready := make(chan struct{}, 1)
	srv.NotifyStartedFunc = func() {
		close(ready)
	}
	go srv.ListenAndServe()
	return srv, ready
}

// testServerExternal gives a test server capable of serving only external
// requests.
func testServerExternal(t *testing.T) (*DnsServer, <-chan struct{}) {
	ns := []string{"8.8.8.8:53", "8.8.4.4:53"}
	srv := New("dontcare", ":8053", rrstore.New(), true, ns)
	ready := make(chan struct{}, 1)
	srv.NotifyStartedFunc = func() {
		close(ready)
	}
	go srv.ListenAndServe()
	return srv, ready
}
