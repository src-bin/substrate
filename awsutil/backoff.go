package awsutil

import (
	"log"
	"math/rand"
	"time"
)

// JitteredExponentialBackoff sleeps and returns true to make it easy to
// implement jittered exponential backoff in a for-loop. Here's how you
// should use it:
//
// for range JitteredExponentialBackoff(time.Second, 10*time.Second) {
// }
func JitteredExponentialBackoff(init, max time.Duration) <-chan bool {
	ch := make(chan bool)
	go func(ch chan<- bool) {
		ch <- true // send immediately to be fast on the happy path
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
			select {
			case ch <- true:
				log.Printf("slept %v+%v and sent", d, jitter)
			default:
				log.Printf("slept %v+%v and broke", d, jitter)
				break
			}

			d *= 2
			if d > max {
				d = max
			}
		}
	}(ch)
	return ch
}
