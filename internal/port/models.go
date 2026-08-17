package port

// CommandSpec is a program invocation. Path is an already-resolved executable
// path rather than a bare name, so a $PATH change mid-session cannot redirect
// an action, and Args is argv — never a shell string.
type CommandSpec struct {
	Path string
	Args []string
	Dir  string
	// LogPath, when set, is a file the child's output is copied into as well as
	// shown. It is how a build's result survives long enough to be reviewed
	// after the fact; an empty value means the output is only displayed.
	LogPath string
}

// CommandResult is the captured outcome of a CommandSpec. Stderr is retained
// even on success because bazel writes its progress there.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}
