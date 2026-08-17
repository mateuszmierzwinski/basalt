package systime

import (
	"testing"
	"time"
)

func TestNowIsCurrent(t *testing.T) {
	before := time.Now()
	got := New().Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Fatalf("Now() = %v, outside [%v, %v]", got, before, after)
	}
}

func TestNowAdvances(t *testing.T) {
	clock := New()
	first := clock.Now()
	time.Sleep(time.Millisecond)
	if !clock.Now().After(first) {
		t.Fatal("Now() did not advance")
	}
}
