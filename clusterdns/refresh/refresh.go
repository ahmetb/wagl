package refresh

import (
	"errors"
	"time"
)

type RefreshFunc func(cancel <-chan struct{}) error

// New starts a new loop to call f and returns two channels, one for errors that
// come out of the calls to f or the refresh loop, and one for signaling every
// time f returns successfully.
func New(f RefreshFunc, tickCh <-chan time.Time, timeout time.Duration, cancelCh <-chan struct{}) (<-chan error, <-chan struct{}) {
	errCh := make(chan error, 1) // buffered so that we don't block on error
	okCh := make(chan struct{}, 1)

	go func() { // in the background
		for _ = range tickCh {
			// handle starting, timeout, cancellation or completion of call
			// to f in the background
			go func() {
				cancelF := make(chan struct{})
				doneF := make(chan struct{}, 1)
				errF := make(chan error, 1)
				go func() { // Kick "f" off in the background
					err := f(cancelF)
					if err != nil {
						errF <- err
					} else {
						doneF <- struct{}{} // never blocks sending b/c buffered
					}
				}()
				select {
				case <-time.After(timeout): // task timed out
					errCh <- errors.New("refreshing timed out")
					close(cancelF)
				case <-cancelCh: // cancellation from upstream
					close(cancelF)
				case err := <-errF: // f errored
					errCh <- err
				case <-doneF: // task done
					okCh <- struct{}{}
				}

			}()
		}
	}()
	return errCh, okCh
}
