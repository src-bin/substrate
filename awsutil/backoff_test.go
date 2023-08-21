package awsutil

import (
	"testing"
	"time"
)

func TestJitteredExponentialBackoff(t *testing.T) {

	// Typially we'd see `for range JitteredExponentialBackoff(...)` but here
	// we want to interrogate the channel and goroutine after the loop breaks.
	ch := JitteredExponentialBackoff(10*time.Millisecond, 100*time.Millisecond)

	// Observe 7 iterations through a loop that's pacing itself with this
	// JitteredExponentialBackoff channel. That's enough to see it top out at
	// 100ms +- 25%.
	var i int
	expected := [][2]time.Duration{
		{0, 0},
		{7500 * time.Microsecond, 12500 * time.Microsecond},
		{15 * time.Millisecond, 25 * time.Millisecond},
		{30 * time.Millisecond, 50 * time.Millisecond},
		{60 * time.Millisecond, 100 * time.Millisecond},
		{75 * time.Millisecond, 125 * time.Millisecond},
		{75 * time.Millisecond, 125 * time.Millisecond},
	}
	for actual := range ch {
		//t.Log(time.Now().Format(time.StampMilli))
		if actual < expected[i][0] || expected[i][1] < actual {
			t.Fatalf("expected iteration %d to sleep between %v and %v but slept %v", i, expected[i][0], expected[i][1], actual)
		}
		i++
		if i >= 7 {
			break
		}
	}

	// Sleep longer than the backoff could possibly sleep so that it notices
	// the loop has broken, then verify the goroutine has exited by observing
	// that no more values are sent through the channel.
	time.Sleep(126 * time.Millisecond)
	select {
	case d := <-ch:
		t.Fatalf("expected the JitteredExponentialBackoff goroutine to have exited but it sent %v", d)
	case <-time.After(126 * time.Millisecond):
	}

	//t.Log("test exit")
}
