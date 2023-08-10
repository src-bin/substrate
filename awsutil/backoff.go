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
// for jeb := JitteredExponentialBackoff(time.Second, 10*time.Second); jeb(); {
// }
func JitteredExponentialBackoff(init, max time.Duration) func() bool {
	d := init
	return func() bool {
		log.Print(d)

		// Compute jitter by first getting a random jitter up to 50% of the
		// base sleep and then shifting it down so its range is -25% to +25%.
		jitter := time.Duration(rand.Int63n(int64(d/2))) - d/4

		time.Sleep(d + jitter)

		d *= 2
		if d > max {
			d = max
		}
		return true // backoff is never the thing that breaks a loop
	}
}
