package clusterdns

import (
	"fmt"
	"log"
	"time"

	"wagl/clusterdns/refresh"
	"wagl/rrgen"
	"wagl/rrstore"
	"wagl/task"
)

// ClusterDriver describes a distributed task execution environment.
type ClusterDriver interface {
	// Tasks gives the active tasks in the cluster which may or may not be
	// eligible for load balancing due to various reasons such as  having no
	// ports exposed or invalid characters in service/domain names.
	Tasks() (task.ClusterState, error)
}

// ClusterDNS keeps the DNS records in sync with Cluster state.
type ClusterDNS struct {
	domain string
	rr     rrstore.RRWriter
	cl     ClusterDriver
}

func New(domain string, rr rrstore.RRWriter, cl ClusterDriver) *ClusterDNS {
	return &ClusterDNS{domain, rr, cl}
}

// SyncRecords syncs the DNS records in the RR table with the cluster by
// querying the cluster and updating the RR table.
func (c *ClusterDNS) SyncRecords() error {
	state, err := c.cl.Tasks()
	if err != nil {
		return fmt.Errorf("error fetching cluster state: %v", err)
	}
	c.rr.Set(rrgen.RRs(c.domain, state))
	return nil
}

func (c *ClusterDNS) StartRefreshing(interval, timeout time.Duration, cancel <-chan struct{}) (<-chan error, <-chan struct{}) {
	t := time.NewTicker(interval)
	go func() { // garbage collect the ticker
		<-cancel
		t.Stop()
	}()

	log.Printf("Starting to refresh DNS records every %v...", interval)
	return refresh.New(func(cancel <-chan struct{}) error {
		// TODO see if we can plumb the cancellation to SyncRecords
		log.Println("Refreshing DNS records...")
		return c.SyncRecords()
	}, t.C, timeout, cancel)
}
