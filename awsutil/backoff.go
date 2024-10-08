package awsutil

import (
	"math/rand"
	"time"
)

// JitteredExponentialBackoff sleeps and returns true to make it easy to
// implement jittered exponential backoff in a for-loop. Here's how you
// should use it:
//
// for range awsutil.JitteredExponentialBackoff(time.Second, 10*time.Second) {
// }
func JitteredExponentialBackoff(init, max time.Duration) <-chan time.Duration {
	ch := make(chan time.Duration)
	go func(ch chan<- time.Duration) {
		ch <- 0 // send immediately to be fast on the happy path
		//log.Print("sent without sleeping")
		d := init
		for {

			// Compute jitter by first getting a random jitter up to 50% of the
			// base sleep and then shifting it down so its range is -25% to +25%.
			jitter := time.Duration(rand.Int63n(int64(d/2))) - d/4

			// Sleep and then send another value across the channel. If this
			// would block then we know the caller has finished their loop.
			// We can safely break to let this goroutine exit and the channel
			// be garbage-collected.
			time.Sleep(d + jitter)
			/*
				var closed bool
				select {
				case ch <- d + jitter:
					//log.Printf("slept %v+%v and sent", d, jitter)
				default:
					closed = true // indirection because break applies to selects, too
					//log.Printf("slept %v+%v and broke the loop", d, jitter)
				}
				if closed {
					break
				}
			*/ // XXX fine, let the channel and the goroutine leak
			ch <- d + jitter

			d *= 2
			if d > max {
				d = max
			}
		}
		//log.Print("goroutine exit")
	}(ch)
	return ch
}

// StandardJitteredExponentialBackoff returns a JitteredExponentialBackoff
// configured to start at 1 second and backoff to 10 seconds between attempts.
// Here's how you should use it:
//
// for range awsutil.StandardJitteredExponentialBackoff() {
// }
func StandardJitteredExponentialBackoff() <-chan time.Duration {
	return JitteredExponentialBackoff(time.Second, 10*time.Second)
}
