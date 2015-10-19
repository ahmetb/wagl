package rrgen

import (
	"testing"

	"github.com/ahmetalpbalkan/wagl/task"
)

func TestFilterTasks(t *testing.T) {
	var (
		allGood = func(t task.Task) (bool, string) { return true, "" }
		allBad  = func(t task.Task) (bool, string) { return false, "nope" }
		hasPort = func(t task.Task) (bool, string) {
			if len(t.Ports) > 0 {
				return true, ""
			}
			return false, "task has no ports"
		}
	)
	ll := make([]task.Task, 5)

	// 0th task has a port
	ll[0] = task.Task{Ports: []task.Port{
		{HostPort: 80}}}

	cases := []struct {
		fs   Filters
		good int
		bad  int
	}{
		{[]FilterFunc{allGood}, len(ll), 0},
		{[]FilterFunc{allGood, allBad}, 0, len(ll)},
		{[]FilterFunc{allGood, hasPort}, 1, len(ll) - 1},
	}
	for i, c := range cases {
		if o, b := c.fs.FilterTasks(ll); len(o) != c.good {
			t.Fatalf("case %d: wrong good task count", i)
		} else if len(b) != c.bad {
			t.Fatalf("case %d: wrong bad task count", i)
		}
	}
}
