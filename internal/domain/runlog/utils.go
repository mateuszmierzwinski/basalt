package runlog

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

const (
	// SchemaVersion is bumped whenever History or Entry changes shape. An older
	// or newer file is discarded rather than misparsed.
	//
	// 2 dropped the recorded root, which made the history checkout-portable.
	SchemaVersion = 2

	historyFileName = "history.json"
	logsDirName     = "logs"
	logExtension    = ".log"

	// digestLength keeps the label's fingerprint short while leaving collisions
	// out of reach — the alternative, trusting the flattened name to be unique,
	// would show one target's log under another's name.
	digestLength = 12
)

// FilePath returns the history file inside the cache directory.
func FilePath(cacheDir string) string {
	return filepath.Join(cacheDir, historyFileName)
}

// LogsDirectory returns the directory holding captured command output.
func LogsDirectory(cacheDir string) string {
	return filepath.Join(cacheDir, logsDirName)
}

// LogFileName derives a log file name from a target label.
//
// The label is flattened to a safe character set and a digest of the original
// is appended. The flattening is what keeps a hostile label from steering the
// path; the digest is what keeps two labels that flatten alike from sharing a
// file. One file per target, overwritten each run, so the logs directory is
// bounded by the number of targets ever run.
func LogFileName(label string) string {
	digest := sha256.Sum256([]byte(label))
	return flatten(label) + "-" + hex.EncodeToString(digest[:])[:digestLength] + logExtension
}

// LogPath is where a target's captured output belongs.
func LogPath(cacheDir, label string) string {
	return filepath.Join(LogsDirectory(cacheDir), LogFileName(label))
}

// LogNameOf reduces a log path to the name the history records.
//
// The name is what is persisted, never the path: the same checkout sits at a
// different absolute path on each machine that syncs it, and a recorded path
// would point at the other machine's directory.
func LogNameOf(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

// ResolveLog turns a recorded name back into a path under the running cache
// directory, reporting false when the result would not stay inside it.
//
// It refuses anything that is not a bare file name, and then re-checks the
// joined result: a recorded value such as "/etc/shadow" or "../../etc/passwd"
// is rejected outright rather than coerced into something harmless-looking,
// because the file it names is handed to a pager.
func ResolveLog(cacheDir, name string) (string, bool) {
	if name == "" || name != filepath.Base(name) {
		return "", false
	}
	logs := LogsDirectory(cacheDir)
	path := filepath.Join(logs, name)
	if !IsInside(logs, path) {
		return "", false
	}
	return path, true
}

// flatten reduces a label to characters that cannot mean anything to a path.
func flatten(label string) string {
	var safe strings.Builder
	safe.Grow(len(label))
	for _, r := range label {
		if isSafe(r) {
			safe.WriteRune(r)
			continue
		}
		safe.WriteByte('_')
	}
	return strings.Trim(safe.String(), "_")
}

// isSafe keeps the flattened name to characters that cannot mean anything to a
// path resolver.
//
// The dot is deliberately excluded: a label containing ".." would otherwise
// carry it into the file name. It resolves harmlessly once joined — there is no
// separator to go with it — but a log file literally named ".._.._etc_passwd"
// invites the next reader to assume traversal is possible here. The digest
// already guarantees uniqueness, so nothing is lost by dropping it.
func isSafe(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-':
		return true
	default:
		return false
	}
}

// IsInside reports whether path lies within dir.
//
// It is the guard on every log path that reaches a pager: the history file is
// editable JSON in the repository, so an entry could otherwise name
// /etc/shadow and have basalt open it.
func IsInside(dir, path string) bool {
	cleanDir := filepath.Clean(dir)
	cleanPath := filepath.Clean(path)
	if cleanPath == cleanDir {
		return false
	}
	return strings.HasPrefix(cleanPath, cleanDir+string(filepath.Separator))
}
