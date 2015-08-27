package rrstore

import (
	"reflect"
	"sync"
	"testing"
)

func TestRRStore(t *testing.T) {
	s := New()
	s.Get("foo", 0)

	in := map[uint16]map[string][]string{
		0: {"1": []string{"2"}},
		3: {"4": []string{"5"}},
	}
	s.Set(in)
	if v, _ := s.Get("1", 0); !reflect.DeepEqual(v, in[0]["1"]) {
		t.Fatal("wrong value")
	}
	if v, _ := s.Get("4", 3); !reflect.DeepEqual(v, in[3]["4"]) {
		t.Fatal("wrong value")
	}
	if _, ok := s.Get("1", 1); ok {
		t.Fatal("wrong value")
	}
}

func TestRRStore_RaceCond(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(2)
		go func() {
			s.Get("1", 0)
			wg.Done()
		}()
		go func() {
			s.Set(make(map[uint16]map[string][]string))
			wg.Done()
		}()
	}
	wg.Wait()
}
