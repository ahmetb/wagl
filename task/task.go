// Package task describes a task resource in the cluster
package task

import (
	"fmt"
	"net"
)

// ClusterState describes the current state of the cluster.
type ClusterState []Task

// Task describes a running (active) container in the cluster.
type Task struct {
	Id      string // Identifies container in the cluster
	Ports   []Port // List of container ports mapped to host <IP:port>
	Service string // Name of the service that groups tasks under the same DNS record
	Domain  string // Optional, a domain name describing the project name the task belongs to, or the launcher framework/orchestrator.
}

// Port describes network port of a service on the host machine.
type Port struct {
	HostIP   net.IP
	HostPort int
	Proto    string
}

func (p Port) String() string {
	return fmt.Sprintf("%s:%d/%s", p.HostIP, p.HostPort, p.Proto)
}
