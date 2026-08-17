package index

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/mateuszmierzwinski/basalt/internal/domain/target"
)

func sampleTarget(t *testing.T) target.Target {
	t.Helper()
	tgt, err := target.New(target.KindGoBinary, "//src/go/tools/basalt/cmd/basalt:basalt")
	if err != nil {
		t.Fatalf("target.New() error = %v", err)
	}
	return tgt
}

func sampleIndex(t *testing.T) Index {
	t.Helper()
	return Index{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
		Targets:       []target.Target{sampleTarget(t)},
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := sampleIndex(t)
	data, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	got, err := Decode(data, "/repo")
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Targets) != 1 || got.Targets[0] != original.Targets[0] {
		t.Fatalf("Decode() targets = %+v", got.Targets)
	}
	if !got.GeneratedAt.Equal(original.GeneratedAt) {
		t.Fatalf("Decode() GeneratedAt = %v", got.GeneratedAt)
	}
}

func TestDecodeRejectsCorruptJSON(t *testing.T) {
	if _, err := Decode([]byte("{not json"), "/repo"); !errors.Is(err, ErrCorruptCache) {
		t.Fatalf("Decode() error = %v, want ErrCorruptCache", err)
	}
}

func TestDecodeRejectsTruncatedJSON(t *testing.T) {
	data, err := Encode(sampleIndex(t))
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if _, err := Decode(data[:len(data)/2], "/repo"); !errors.Is(err, ErrCorruptCache) {
		t.Fatalf("Decode() error = %v, want ErrCorruptCache", err)
	}
}

func TestDecodeRejectsSchemaMismatch(t *testing.T) {
	idx := sampleIndex(t)
	idx.SchemaVersion = SchemaVersion + 1
	data, _ := Encode(idx)
	if _, err := Decode(data, "/repo"); !errors.Is(err, ErrSchemaMismatch) {
		t.Fatalf("Decode() error = %v, want ErrSchemaMismatch", err)
	}
}

// The same tree synced to two machines sits at a different absolute path on
// each, so a cache that recorded one would be rejected on the other. Nothing in
// it is absolute, and every label is still re-resolved against the running
// root, so it is accepted and revalidated rather than thrown away.
func TestDecodeAcceptsACacheFromAnotherCheckoutPath(t *testing.T) {
	data, _ := Encode(sampleIndex(t))

	got, err := Decode(data, "/somewhere/else/Monorepo")
	if err != nil {
		t.Fatalf("Decode() error = %v, want a cache staged from another path to load", err)
	}
	if len(got.Targets) != 1 {
		t.Fatalf("Decode() targets = %+v, want the entry carried across", got.Targets)
	}
}

// Portability does not weaken validation: a label that escapes the running root
// is still refused, whichever root that is.
func TestDecodeStillValidatesLabelsAgainstTheRunningRoot(t *testing.T) {
	data := []byte(`{"schema_version":` + strconv.Itoa(SchemaVersion) + `,"targets":[{"kind":"go_binary","package":"//../../etc","name":"passwd","label":"//../../etc:passwd","display":"[etc:passwd]"}]}`)

	if _, err := Decode(data, "/somewhere/else/Monorepo"); !errors.Is(err, ErrPoisonedEntry) {
		t.Fatalf("Decode() error = %v, want ErrPoisonedEntry", err)
	}
}

// The cache is a writable JSON file in the repository root, so these are the
// realistic attacks against it (test/cve covers them end to end too).
func TestDecodeRejectsPoisonedEntries(t *testing.T) {
	cases := []struct {
		name  string
		entry target.Target
	}{
		{
			name:  "label escaping the monorepo",
			entry: target.Target{Kind: "go_binary", Package: "//../../etc", Name: "passwd", Label: "//../../etc:passwd", Display: "[etc:passwd]"},
		},
		{
			name:  "display string not derived from the label",
			entry: target.Target{Kind: "go_binary", Package: "//src/go", Name: "x", Label: "//src/go:x", Display: "[totally:different]"},
		},
		{
			name:  "package disagreeing with the label",
			entry: target.Target{Kind: "go_binary", Package: "//src/evil", Name: "x", Label: "//src/go:x", Display: "[go:x]"},
		},
		{
			name:  "malformed label",
			entry: target.Target{Kind: "go_binary", Package: "src/go", Name: "x", Label: "src/go:x", Display: "[go:x]"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idx := sampleIndex(t)
			idx.Targets = []target.Target{tc.entry}
			data, _ := Encode(idx)
			if _, err := Decode(data, "/repo"); !errors.Is(err, ErrPoisonedEntry) {
				t.Fatalf("Decode() error = %v, want ErrPoisonedEntry", err)
			}
		})
	}
}

func TestDecodeAcceptsAnEmptyIndex(t *testing.T) {
	idx := sampleIndex(t)
	idx.Targets = nil
	data, _ := Encode(idx)
	got, err := Decode(data, "/repo")
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Targets) != 0 {
		t.Fatalf("Decode() targets = %+v", got.Targets)
	}
}

func BenchmarkEncode(b *testing.B) {
	idx := Index{SchemaVersion: SchemaVersion}
	for b.Loop() {
		Encode(idx)
	}
}

func BenchmarkDecode(b *testing.B) {
	idx := Index{SchemaVersion: SchemaVersion}
	data, _ := Encode(idx)
	for b.Loop() {
		Decode(data, "/repo")
	}
}
