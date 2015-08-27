// Package rrstore provides concurrency-safe storage for DNS Resource Records
// (RRs).
package rrstore

import (
	"sync"
)

// RRs stores FQDN RR answer for various RR Types.
// Example:
//     {
//       dns.TypeA: {"a.b." : ["10.0.0.3"]},
//       dns.TypeSRV: {"a.b." : ["10.0.0.3:23481","10.0.0.7:11215"]}
//     }
type RRs map[uint16]map[string][]string

type RRReader interface {
	Get(fqdn string, rrType uint16) (rrs []string, ok bool)
}

type RRWriter interface {
	Set(rl RRs)
}

type RRStore interface {
	RRReader
	RRWriter
}

type rrStore struct {
	rrs RRs
	m   sync.RWMutex
}

// New creates a new record table to store DNS Resource Records.
func New() RRStore {
	return &rrStore{}
}

func (r *rrStore) Get(fqdn string, rrType uint16) (rrs []string, ok bool) {
	r.m.RLock()
	defer r.m.RUnlock()
	rrs, ok = r.rrs[rrType][fqdn]
	return
}

func (r *rrStore) Set(rl RRs) {
	r.m.Lock()
	defer r.m.Unlock()
	r.rrs = rl
}
