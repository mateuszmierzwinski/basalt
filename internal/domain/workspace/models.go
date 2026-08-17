package workspace

// Workspace is the validated monorepo checkout basalt operates on. Every path
// basalt reads, every command it runs and every cache entry it writes is rooted
// here, so the value is produced once at startup and injected thereafter.
type Workspace struct {
	Root       string
	SourceRoot string
	CacheDir   string
}
