package refresh

import (
	"errors"
	"sync"
	"testing"
	"time"
)

var (
	dummyError = errors.New("dummy error")
)

func TestRefresh_simple(t *testing.T) {
	i := 0
	f := func(cancel <-chan struct{}) error {
		i++
		return nil
	}
	tick := make(chan time.Time)
	done := make(chan struct{})
	errCh, okCh := New(f, tick, time.Millisecond*15, done)

	go func() {
		for err := range errCh {
			t.Fatalf("err received: %v", err)
		}
	}()

	// (tick + sleep 10 ms) * 5
	n := 5
	go func() {
		for j := 0; j < n; j++ {
			tick <- time.Now()
			time.Sleep(time.Millisecond * 10)
		}
	}()

	// validate ok signals
	for j := 0; j < n; j++ {
		<-okCh
	}
	select {
	case <-okCh:
		t.Fatal("okCh still has values")
	default:
	}

	// validate calls to f are made
	if expected := n; i != expected {
		t.Fatalf("wrong # of calls to f, expected: %d got: %d", expected, i)
	}
}

func TestRefresh_timeout(t *testing.T) {
	// orchestrate predefined sleeps on each call to f
	sleeps := []int{1, 5, 15, 20, 25, 2, 3, 2}
	var m sync.Mutex
	cur := 0

	var ww sync.WaitGroup

	f := func(cancel <-chan struct{}) error {
		defer ww.Done()
		m.Lock()
		d := time.Duration(sleeps[cur]) * time.Millisecond
		cur++
		num := cur
		m.Unlock()

		t.Logf("f%d sleep for: %v -- %v", num, d, time.Now())
		select {
		case <-cancel:
			t.Logf("f%d is cancelled", num)
		case <-time.After(d):
			t.Logf("f%d done: %v", num, time.Now())
		}
		return nil
	}

	tick := make(chan time.Time)
	done := make(chan struct{})
	errCh, okCh := New(f, tick, time.Millisecond*10, done) //TODO use okch

	// send all ticks at once (won't block b/c buffered ch)
	ww.Add(len(sleeps))
	for _ = range sleeps {
		tick <- time.Now()
	}
	ww.Wait()

	expectedErrs := 3
	for i := 0; i < expectedErrs; i++ {
		<-errCh
	}
	select {
	case <-errCh:
		t.Fatal("there are more errors")
	default:
	}

	expectedOks := len(sleeps) - expectedErrs
	for i := 0; i < expectedOks; i++ {
		<-okCh
	}
	select {
	case <-okCh:
		t.Fatal("there are more OKs")
	default:
	}
	close(done)
}

func TestRefresh_errGoesToErrCh(t *testing.T) {
	f := func(cancel <-chan struct{}) error {
		return dummyError
	}
	tick := make(chan time.Time)
	done := make(chan struct{})
	defer close(done)
	errCh, _ := New(f, tick, time.Millisecond*15, done) //TODO use okch

	n := 5
	go func() {
		for i := 0; i < n; i++ {
			tick <- time.Now()
		}
	}()

	for i := 0; i < n; i++ {
		err := <-errCh
		if err != dummyError {
			t.Fatalf("got wrong error: %v", err)
		}
	}
}

func TestRefresh_cancelsTaskOnTimeout(t *testing.T) {
	timeout := time.Millisecond * 20

	f := func(cancel <-chan struct{}) error {
		select {
		case <-cancel:
			t.Log("successfully cancelled")
		case <-time.After(timeout * 2):
			t.Fatal("did not cancel the task on timeout")
		}
		return nil
	}
	tick := make(chan time.Time)
	done := make(chan struct{})
	defer close(done)

	_, _ = New(f, tick, timeout, done) //TODO use okch

	// send a tick
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case tick <- time.Now():
		case <-done:
			t.Fatal("could not send tick")
		}
	}()

	wg.Wait()
}

func TestRefresh_cancellation(t *testing.T) {
	timeout := time.Second

	fCancelled := make(chan struct{}, 1)
	f := func(cancel <-chan struct{}) error {
		select {
		case fCancelled <- <-cancel:
			t.Log("got cancellation")
		case <-time.After(time.Second):
			t.Fatalf("f not cancelled")
		}
		return nil
	}
	tick := make(chan time.Time, 1)
	done := make(chan struct{})

	_, _ = New(f, tick, timeout, done)
	tick <- time.Now() // won't block

	<-time.After(time.Millisecond * 10)
	t.Log("closing cancel ch")
	close(done) // cancel!

	time.Sleep(time.Millisecond * 20) // give f some time to cancel

	select {
	case <-fCancelled:
		t.Log("cancellation happened")
	default:
		t.Fatal("f did not do cancellation")
	}
}

func TestRefresh_fTakesMoreTimeThanTicks_soThatTasksInterleave(t *testing.T) {
	started := 0
	n := 10 // tick count
	finished := make(chan struct{}, n)

	sleep := 200 * time.Millisecond

	var m sync.Mutex
	f := func(cancel <-chan struct{}) error {
		m.Lock()
		started++
		m.Unlock()
		t.Log("task started")

		select {
		case <-time.After(sleep):
			t.Log("task done")
			finished <- struct{}{}
		case <-cancel:
		}
		return nil
	}

	done := make(chan struct{})
	defer close(done)
	tick := make(chan time.Time)
	_, _ = New(f, tick, time.Second, done)

	var wg sync.WaitGroup
	wg.Add(n)

	start := time.Now()

	// send ticks
	for i := 0; i < n; i++ {
		t.Logf("--> scheduling (%d): %v", i, time.Now())
		go func(num int) {
			defer wg.Done()

			select {
			case tick <- time.Now():
				t.Logf("--- scheduled (%d)...: %v", num, time.Now())
			case <-done:
				t.Fatalf("retryloop is canceled before sending tick")
			}
		}(i)
	}
	wg.Wait()

	// prove calls to f interleave and not serialized
	d := time.Since(start)
	if expMax := sleep; d > expMax {
		t.Fatalf("sending ticks took longer than expected (%v > %v). are they blocked on f to return? ", d, expMax)
	}

	var wg2 sync.WaitGroup
	wg2.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg2.Done()
			<-finished
		}()
	}
	wg2.Wait()
}

func TestRefresh_fSometimesReturnsErr(t *testing.T) {
	vals := []bool{false, true, false, false, true, false}
	cur := 0
	var m sync.Mutex
	f := func(c <-chan struct{}) error {
		m.Lock()
		v := vals[cur]
		cur++
		m.Unlock()

		if !v {
			return dummyError
		}
		return nil
	}

	tick := make(chan time.Time)
	done := make(chan struct{})
	errCh, okCh := New(f, tick, time.Second, done)
	defer close(done)

	// send ticks
	go func() {
		for _ = range vals {
			tick <- time.Now()
		}
	}()

	expectedErrs := 4
	expectedOKs := len(vals) - expectedErrs

	oks := 0
	errs := 0

	for {
		if oks+errs == len(vals) {
			break
		}
		select {
		case <-okCh:
			oks++
		case <-errCh:
			errs++
		}
	}

	if errs != expectedErrs {
		t.Fatalf("wrong errs count: %d, expected: %d", errs, expectedErrs)
	}
	if oks != expectedOKs {
		t.Fatalf("wrong OKs count: %d, expected: %d", oks, expectedOKs)
	}

	t.Log("alright... see if chans still have value")
	select {
	case <-errCh:
		t.Fatal("errCh still has value")
	default:
	}
	select {
	case <-okCh:
		t.Fatal("okCh still has value")
	default:
	}
}
