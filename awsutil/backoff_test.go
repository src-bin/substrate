package awsutil

import (
	"testing"
	"time"
)

func TestJitteredExponentialBackoff(t *testing.T) {
	var i int
	for range JitteredExponentialBackoff(10*time.Millisecond, 100*time.Millisecond) { // for fast tests that still prove results
		t.Log(time.Now().Format(time.StampMilli))
		if i >= 5 {
			break
		}
		i++
	}
}
