package rrtype

import (
	"testing"

	"github.com/miekg/dns"
)

func TestIsSupported(t *testing.T) {
	cases := []struct {
		rrType    uint16
		supported bool
	}{
		// supported
		{dns.TypeA, true},
		{dns.TypeSRV, true},

		// some others
		{dns.TypeCNAME, false},
		{dns.TypeNS, false},
		{dns.TypeMX, false},
	}
	for _, c := range cases {
		out := IsSupported(c.rrType)
		if out != c.supported {
			t.Fatal("wrong value for %s", dns.TypeToString[c.rrType])
		}
	}
}
