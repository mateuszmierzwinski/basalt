package index

import (
	"testing"
	"time"
)

func TestFilePath(t *testing.T) {
	if got := FilePath("/repo/basalt_target_cache"); got != "/repo/basalt_target_cache/targets.json" {
		t.Fatalf("FilePath() = %q", got)
	}
}

func TestIsStale(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name      string
		generated time.Time
		want      bool
	}{
		{"just generated", now, false},
		{"an hour old", now.Add(-time.Hour), false},
		{"just under the TTL", now.Add(-TTL + time.Minute), false},
		{"exactly at the TTL", now.Add(-TTL), true},
		{"well past the TTL", now.Add(-72 * time.Hour), true},
		{"generated in the future", now.Add(time.Hour), true},
		{"zero value", time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsStale(Index{GeneratedAt: tc.generated}, now); got != tc.want {
				t.Fatalf("IsStale() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestAge(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	if got := Age(Index{GeneratedAt: now.Add(-90 * time.Minute)}, now); got != 90*time.Minute {
		t.Fatalf("Age() = %v", got)
	}
}

func BenchmarkIsStale(b *testing.B) {
	idx := Index{GeneratedAt: time.Now()}
	now := time.Now()
	for b.Loop() {
		IsStale(idx, now)
	}
}

func BenchmarkFilePath(b *testing.B) {
	for b.Loop() {
		FilePath("/repo/basalt_target_cache")
	}
}

func BenchmarkAge(b *testing.B) {
	idx := Index{GeneratedAt: time.Now()}
	now := time.Now()
	for b.Loop() {
		Age(idx, now)
	}
}
