package rrtype

import (
	"fmt"
	"net"
	"strconv"

	"github.com/miekg/dns"
)

type formatterFunc func(name, rr string) (dns.RR, error)

var rrFormatters = map[uint16]formatterFunc{
	dns.TypeA:   formatA,
	dns.TypeSRV: formatSRV,
}

// IsSupported returns if the system supports answering to questions
// for specified DNS RR Type.
func IsSupported(rrType uint16) bool {
	_, ok := rrFormatters[rrType]
	return ok
}

// ToRR converts stored RR info to an appropriate DNS RR based on rrType
// specified (e.g. A, SRV).
func ToRR(rrType uint16, name, rec string) (dns.RR, error) {
	f, ok := rrFormatters[rrType]
	if !ok {
		return nil, fmt.Errorf("Formatting RR to %s(%d) REC not implemented", dns.TypeToString[rrType], rrType)
	}
	return f(name, rec)
}

// formatA formats an IP address record for a into A record.
func formatA(name, rec string) (dns.RR, error) {
	return &dns.A{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeA,
			Class:  dns.ClassINET,
			Ttl:    0},
		A: net.ParseIP(rec),
	}, nil
}

// formatSRV formats an IP:port record into a SRV record.

func formatSRV(name, rec string) (dns.RR, error) {
	host, port, err := net.SplitHostPort(rec)
	if err != nil {
		return nil, fmt.Errorf("cannot format addr %s to SRV record: %v", rec, err)
	}
	host = dns.Fqdn(host) // have . suffix per SRV RFC

	portNum, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("cannot parse port number in %s: %v", rec, err)
	}

	return &dns.SRV{
		Hdr: dns.RR_Header{
			Name:   name,
			Rrtype: dns.TypeSRV,
			Class:  dns.ClassINET,
			Ttl:    0,
		},
		Target:   host,
		Port:     uint16(portNum),
		Priority: 1, // keep all records equal
		Weight:   1, // keep all records equal
	}, nil
}
