package rrgen

import (
	"fmt"

	"wagl/task"
)

// FilterFunc determines if a Task can be used, if not provides a reason.
type FilterFunc func(t task.Task) (bool, string)

type Filters []FilterFunc

// BadTask describes a Task that is not eligible.
type BadTask struct {
	task.Task
	Reason string
}

// DnsFilters are list of filters applied in order to determine DNS eligibility
// of tasks. The end result of the filters are Tasks (containers) that can have
// DNS RRs.
var DnsFilters = Filters([]FilterFunc{
	HasDnsName,
	HasPorts,
	PortsHaveProtos,
})

// filterTasks filters tasks based on their eligibility for having DNS records
// and returns the list of good tasks and bad ones along with their reasons.
func (f Filters) FilterTasks(ll []task.Task) ([]task.Task, []BadTask) {
	badTasks := make([]BadTask, 0)
	goodTasks := make([]task.Task, 0)

	for _, t := range ll {
		bad := false
		for _, ff := range f {
			if ok, reason := ff(t); !ok {
				badTasks = append(badTasks, BadTask{t, reason})
				bad = true
				break
			}
		}
		if !bad {
			goodTasks = append(goodTasks, t)
		}
	}
	return goodTasks, badTasks
}

// Filters

func HasPorts(t task.Task) (bool, string) {
	return len(t.Ports) > 0, "has no port mappings"
}

func HasDnsName(t task.Task) (bool, string) {
	return t.Service != "", "has no DNS name specified (or not configured for DNS)"
}

func PortsHaveProtos(t task.Task) (bool, string) {
	for _, p := range t.Ports {
		if p.Proto == "" {
			return false, fmt.Sprintf("no network protocol specified for port mapping '%s'", p)
		}
	}
	return true, ""
}

// TODO implement DNS name checks (length, valid characters and such)
