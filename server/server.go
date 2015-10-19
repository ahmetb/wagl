package server

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/ahmetalpbalkan/wagl/rrstore"
	"github.com/ahmetalpbalkan/wagl/rrtype"
	"github.com/miekg/dns"
)

var (
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
)

type DnsServer struct {
	*dns.Server
	rr rrstore.RRReader

	recurse     bool
	nameservers []string
}

// New creates a DnsServer ready to serve queries for the specified domain on
// the given host:port using the specified DNS Resource Record table as the
// source of truth.
func New(domain, addr string, rr rrstore.RRReader, recurse bool, nameservers []string) *DnsServer {
	d := &DnsServer{rr: rr,
		recurse:     recurse,
		nameservers: nameservers}

	mux := dns.NewServeMux()
	mux.HandleFunc(".", d.handleExternal)
	mux.HandleFunc(dns.Fqdn(domain), d.handleDomain)
	d.Server = &dns.Server{
		Addr:    addr,
		Net:     "udp",
		Handler: mux,
	}
	d.Server.NotifyStartedFunc = func() {
		log.Printf("DNS server started listening at %s", d.Server.Addr)
	}
	return d
}

// handleExternal handles DNS queries that are outside the cluster's domain such
// as the Public Internet.
func (d *DnsServer) handleExternal(w dns.ResponseWriter, r *dns.Msg) {
	dom, qType := parseQuestion(r)
	q := dns.TypeToString[qType] + " " + dom
	log.Printf("--> External: %s", q)

	if !d.recurse {
		log.Printf("<-x %s: SERVFAIL: recursion disabled", q)
		m := new(dns.Msg)
		m.SetReply(r)
		m.SetRcode(r, dns.RcodeServerFailure)
		m.Authoritative = false
		m.RecursionAvailable = false
		w.WriteMsg(m)
	} else {
		in, ns, err := d.queryExternal(r)
		if err != nil {
			log.Printf("<-x %s (@%s): SERVFAIL: %v", q, ns, err)
			m := new(dns.Msg)
			m.SetReply(r)
			m.SetRcode(r, dns.RcodeServerFailure)
			w.WriteMsg(m)
		} else {
			log.Printf("<-- %s (@%s): %d answers, %d extra, %d ns", q, ns, len(in.Answer), len(in.Extra), len(in.Ns))
			in.Compress = true
			w.WriteMsg(in)
		}
	}
}

// handleDomain handles DNS queries that come to the cluster
func (d *DnsServer) handleDomain(w dns.ResponseWriter, r *dns.Msg) {
	dom, qType := parseQuestion(r)
	q := dns.TypeToString[qType] + " " + dom
	log.Printf("--> Internal: %s", q)

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	supported, found, recs := d.queryRR(qType, dom)
	if !supported {
		log.Printf("<-x %s: NOTIMP", q)
		m.SetRcode(r, dns.RcodeNotImplemented) // NOTIMP
	} else if !found {
		log.Printf("<-x %s: NXDOMAIN", q)
		m.SetRcode(r, dns.RcodeNameError) // NXDOMAIN
	} else {
		for _, rec := range recs {
			rr, err := rrtype.ToRR(qType, dom, rec)
			if err != nil {
				log.Printf("<-x %s SERVFAIL: record conv err: %v", q, err)
				m.SetRcode(r, dns.RcodeServerFailure)
				break
			} else {
				log.Printf("<-- %s: %s", q, rr.String())
				m.Answer = append(m.Answer, rr)
			}
		}
	}
	w.WriteMsg(m)
}

// queryExternal makes an external DNS query to a randomly picked external
// nameserver.
func (d *DnsServer) queryExternal(req *dns.Msg) (*dns.Msg, string, error) {
	// TODO use other nameservers in case of failure?
	ns := d.nameservers[rnd.Intn(len(d.nameservers))]
	c := new(dns.Client)
	in, _, err := c.Exchange(req, ns)
	return in, ns, err
}

// queryRR queries the DNS Resource Records for given record type. If the record
// type is not supported or record is not found, false is returned from return
// values, respectively. If records are found, they are returned in a shuffled
// manner.
func (d *DnsServer) queryRR(qType uint16, domain string) (supported bool, found bool, records []string) {
	if !rrtype.IsSupported(qType) {
		return false, false, nil
	}
	recs, ok := d.rr.Get(domain, qType)
	if !ok {
		return true, false, nil
	}
	shuffle(recs)
	return true, true, recs
}

// parseQuestion parses the first question in the DNS message into domain name
// and DNS RR Type.
func parseQuestion(r *dns.Msg) (domain string, qType uint16) {
	q := r.Question[0]
	return strings.TrimSpace(strings.ToLower(q.Name)), q.Qtype
}

// shuffle is an implementation of Modern Fisher–Yates shuffle algortihm.
func shuffle(a []string) {
	for i := len(a) - 1; i > 0; i-- {
		r := rnd.Intn(i)
		a[i], a[r] = a[r], a[i]
	}
}
