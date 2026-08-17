package runlog

import (
	"errors"
	"testing"
	"time"
)

const (
	testRoot  = "/repo"
	testCache = "/repo/basalt_target_cache"
)

func sampleHistory(t *testing.T) History {
	t.Helper()
	label := "//src/go/tools/basalt/cmd/basalt:basalt"
	return History{
		SchemaVersion: SchemaVersion,
		Entries: map[string]Entry{
			label: {
				Label:   label,
				Action:  "bazel.build",
				Status:  StatusPassed,
				RanAt:   time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
				LogFile: LogFileName(label),
			},
		},
	}
}

func encode(t *testing.T, history History) []byte {
	t.Helper()
	data, err := Encode(history)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	return data
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := sampleHistory(t)
	got, err := Decode(encode(t, original), testCache)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("Decode() entries = %+v", got.Entries)
	}
	for label, entry := range got.Entries {
		if entry.Status != StatusPassed || entry.Label != label {
			t.Fatalf("entry = %+v", entry)
		}
	}
}

func TestDecodeRejectsCorruptJSON(t *testing.T) {
	for _, data := range []string{"", "{", "not json", `{"schema_version":`} {
		if _, err := Decode([]byte(data), testCache); !errors.Is(err, ErrCorruptHistory) {
			t.Fatalf("Decode(%q) error = %v, want ErrCorruptHistory", data, err)
		}
	}
}

func TestDecodeRejectsSchemaMismatch(t *testing.T) {
	history := sampleHistory(t)
	history.SchemaVersion = SchemaVersion + 1
	if _, err := Decode(encode(t, history), testCache); !errors.Is(err, ErrSchemaMismatch) {
		t.Fatalf("Decode() error = %v, want ErrSchemaMismatch", err)
	}
}

// The same checkout sits at a different absolute path on each machine that
// syncs it, so a history recording one would be discarded on the other. Nothing
// in it is absolute — entries are labels and log names — so it loads, and the
// names are still resolved and re-checked against the running cache directory.
func TestDecodeAcceptsAHistoryFromAnotherCheckoutPath(t *testing.T) {
	got, err := Decode(encode(t, sampleHistory(t)), "/elsewhere/Monorepo/basalt_target_cache")
	if err != nil {
		t.Fatalf("Decode() error = %v, want a history staged from another path to load", err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("entries = %+v, want the entry carried across", got.Entries)
	}
}

// The history is editable JSON in the repository, and the file it names is
// opened in a pager — so a forged path is an arbitrary-file-read if it is not
// caught here.
func TestDecodeRejectsALogOutsideTheCache(t *testing.T) {
	for _, forged := range []string{
		"/etc/shadow",
		"/repo/basalt_target_cache/logs/../../../etc/passwd",
		"/repo/basalt_target_cache/history.json",
		"../../../../etc/passwd",
		"../history.json",
		"nested/name.log",
		"..",
		".",
	} {
		history := sampleHistory(t)
		for label, entry := range history.Entries {
			entry.LogFile = forged
			history.Entries[label] = entry
		}
		if _, err := Decode(encode(t, history), testCache); !errors.Is(err, ErrLogEscapesDir) {
			t.Fatalf("Decode() accepted log path %q (error = %v)", forged, err)
		}
	}
}

func TestDecodeRejectsAMisfiledEntry(t *testing.T) {
	history := sampleHistory(t)
	history.Entries["//src/go/a:b"] = Entry{Label: "//src/go/somewhere/else:x"}
	if _, err := Decode(encode(t, history), testCache); !errors.Is(err, ErrPoisonedEntry) {
		t.Fatalf("Decode() error = %v, want ErrPoisonedEntry", err)
	}
}

func TestDecodeRejectsAMalformedLabel(t *testing.T) {
	history := sampleHistory(t)
	history.Entries["not-a-label"] = Entry{Label: "not-a-label"}
	if _, err := Decode(encode(t, history), testCache); !errors.Is(err, ErrPoisonedEntry) {
		t.Fatalf("Decode() error = %v, want ErrPoisonedEntry", err)
	}
}

// An entry with no log yet is ordinary: the target ran before logs were kept,
// or the capture failed.
func TestDecodeAcceptsAnEntryWithNoLog(t *testing.T) {
	history := sampleHistory(t)
	for label, entry := range history.Entries {
		entry.LogFile = ""
		history.Entries[label] = entry
	}
	if _, err := Decode(encode(t, history), testCache); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

func TestDecodeAcceptsAnEmptyHistory(t *testing.T) {
	got, err := Decode(encode(t, History{SchemaVersion: SchemaVersion}), testCache)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Entries) != 0 {
		t.Fatalf("entries = %+v", got.Entries)
	}
}

func BenchmarkEncode(b *testing.B) {
	history := History{SchemaVersion: SchemaVersion}
	for b.Loop() {
		Encode(history)
	}
}

func BenchmarkDecode(b *testing.B) {
	data, _ := Encode(History{SchemaVersion: SchemaVersion})
	for b.Loop() {
		Decode(data, testCache)
	}
}
