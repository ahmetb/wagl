// Package rrgen finds out tasks eligible for having DNS records and generates
// th DNS Resource Records for these.
package rrgen

import (
	"fmt"
	"log"

	"github.com/ahmetalpbalkan/wagl/rrstore"
	"github.com/ahmetalpbalkan/wagl/task"
	"github.com/miekg/dns"
)

type rrEntry struct {
	rrType uint16
	domain string
	record string
}

func (r *rrEntry) String() string {
	return fmt.Sprintf("%s %s %s", dns.TypeToString[r.rrType], r.domain, r.record)
}

// RRs determines the tasks which can have DNS Resource Records and returns the
// RRs based on the given cluster state.
func RRs(domain string, state task.ClusterState) rrstore.RRs {
	goodTasks, badTasks := DnsFilters.FilterTasks(state)
	if len(badTasks) > 0 {
		log.Printf("Found %d tasks are not eligible for DNS records:", len(badTasks))
		for _, v := range badTasks {
			log.Printf("\t- %s: %s", v.Id, v.Reason)
		}
	}
	log.Printf("Tasks with DNS records: %d", len(goodTasks))
	return getRRs(domain, goodTasks)
}

// getRRs generates all DNS Resource Record table for the given tasks by
// generating records for each task individually and then grouping them by their
// service[.domain] name.
func getRRs(domain string, ll []task.Task) rrstore.RRs {
	rr := make(rrstore.RRs)
	for _, t := range ll {
		for _, r := range getTaskRRs(domain, t) {
			log.Printf("\t+RR: %s", r.String())
			insertRR(rr, r)
		}
	}
	return rr
}

// getTaskRRs returns all DNS RRs of a Task as a list
func getTaskRRs(domain string, t task.Task) []rrEntry {
	l := make([]rrEntry, 0)

	// Prepend task domain to DNS domain
	tail := dns.Fqdn(domain)
	if t.Domain != "" {
		tail = dns.Fqdn(t.Domain) + tail
	}

	// A record ("A service.domain. IP")
	ip := t.Ports[0].HostIP.String() // use first port mapping's IP addr
	l = append(l, rrEntry{dns.TypeA, fmt.Sprintf("%s.%s", t.Service, tail), ip})

	// SRV records for each port mapping ("SRV _service._tcp.domain. IP PORT")
	for _, p := range t.Ports {
		val := fmt.Sprintf("%s:%d", p.HostIP, p.HostPort)
		l = append(l, rrEntry{dns.TypeSRV, fmt.Sprintf("_%s._%s.%s", t.Service, p.Proto, tail), val})
	}
	return l
}

// insertRR adds the specified RR entry into the RR table.
func insertRR(rr rrstore.RRs, entry rrEntry) {
	if rr[entry.rrType] == nil {
		rr[entry.rrType] = make(map[string][]string)
	}
	rr[entry.rrType][entry.domain] = append(rr[entry.rrType][entry.domain], entry.record)
}
