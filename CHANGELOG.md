# Changelog

All notable changes to basalt are recorded here, newest release first
([Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format).

This file and the `%changelog` in `basalt.spec` must say the same thing — the
spec's version is what `rpm -q --changelog` and `dnf` show (CLAUDE.md Rule 27.6).

## [0.6.0] — 2026-08-16

### Added

- **A portable tarball**, `//src/go/tools/basalt:basalt_tgz`, alongside the RPM —
  for distributions where an RPM is no use. There is deliberately no AppImage:
  AppImage exists to carry heavy runtime dependencies, and a pure-static
  terminal binary has none to carry.
- **basalt is published at github.com/mateuszmierzwinski/basalt**, as a source
  mirror plus the release artifacts. `./export_to_github.sh` regenerates it and
  `EXPORT_TO_GITHUB.md` is the procedure. Development stays in the monorepo; the
  mirror carries no history and accepts no patches.

### Changed

- **Each toolchain gets its own frame in the header.** They previously ran
  together on one line, which read as a single sentence — `🔧 gcc/g++ 16.1.1
  🐹 go 1.25.8 → 1.26.5` invites you to parse it as one statement about one
  thing. They are now separate cells divided by a rule, closed off from the
  target list beneath, with the panel border above carrying the connector. A
  package written in one language gets one full-width frame.
- A toolchain the host cannot satisfy now reddens **only its own cell**. The
  whole header used to turn red, which said "something here is wrong" without
  saying which compiler — and reddened the C frame for a Go requirement it has
  nothing to do with.
- The header belongs to the target column, so the action panel keeps filling
  from the top of the frame rather than losing rows to it.
- Test fixtures name invented projects — `falcon`, `mixer`, `probe` — instead of
  the monorepo's real ones. A test has no business depending on a sibling
  project's name, and it is what lets the public mirror ship its tests unedited.

## [0.5.0] — 2026-08-16

### Added

- **`Bazel > Test package`.** A library or a binary now offers to run the tests
  that cover it. Tests almost never live in the same Bazel rule as the code they
  test, so the old rule — offer `Bazel > Test` only when the selected target is
  itself a `*_test` — meant a project with a full suite looked untested from
  every target except the test rules themselves. The new action runs
  `bazel test //<package>/...`, so it reaches tests in subpackages beneath the
  selection. It is offered only when there is something under the package to
  run, and a test rule keeps its own precise action instead of being offered
  both.

### Changed

- **The target cache and the run history are portable between checkouts.** Both
  recorded the absolute path of the monorepo they were written in and refused to
  load anywhere else, so the same tree synced to two machines — where `$MONOREPO`
  points at a different directory on each — invalidated its cache on every
  switch and lost its run history entirely. Neither file holds an absolute path
  now: entries are Bazel labels, which are workspace-relative by construction,
  and a log is recorded by name rather than by path. Both schema versions are
  bumped to 2, so the existing files are discarded once and rebuilt.

### Security

- Dropping the recorded root removes a check, so the guarantee it was standing
  in for is now made where it belongs: every label is re-resolved against the
  running monorepo root on load, and every log name is resolved against the
  running cache directory and refused unless it is a bare file name landing
  inside it. A cache or history staged from anywhere else still cannot name a
  path outside this tree, and the CVE suite asserts exactly that rather than
  asserting that foreign files are rejected.

## [0.4.0] — 2026-08-15

### Added

- **Markers on every row.** A package heading is led by `▦`, and each target by
  its last outcome: `✓` passed, `☠` failed, `→` never run. The state of the tree
  is now readable without relying on colour at all, which matters on a terminal
  with an unusual palette as much as it does for colour-blind readers.
- The unrun marker is grey but its target's name is not: most of the list has
  never been run, and greying all of it would make the rows carrying no news
  harder to read than the few that do.
- **Languages on package headings.** A heading is suffixed with what its targets
  are written in — `🐹` for Go, `🔧` for C and C++ — in alphabetical order, so a
  package holding both reads `🔧 🐹`.
- **A toolchain header.** Above the list, basalt reports what the selected
  package needs against what this machine has: `🐹 go 1.25.8 → 1.26.5` for Go —
  the `go` directive of the nearest `go.mod` above the target, not a repo-wide
  guess — and `🔧 gcc/g++ 16.1.1` for C. A version basalt cannot determine shows
  as `?`, which is deliberately different from showing nothing. When the host Go
  is older than required the whole line turns red.
- **Packaging targets are marked** with `📦` — an RPM, TGZ, AppImage or tarball
  target. Building one is a different act from building a binary, and the list
  now says which is which.

### Changed

- Target rows no longer carry their own indentation — the marker does. Match
  highlighting is therefore measured against the target name rather than
  against decoration.

### Fixed

- The end-to-end suite inherited the developer's full `PATH`, so it saw whatever
  was installed on the machine. Symlinking GoLand and CLion into `~/.local/bin`
  duly broke the case asserting that absent tools stay hidden — basalt was
  right, the test was not isolated. Fixtures now expose only their own bin
  directory plus `/usr/bin:/bin`.

## [0.3.0] — 2026-08-15

### Added

- **Run history.** basalt remembers how each target's last bazel command went.
  A target that passed is drawn green, one that failed red, and one that has
  never been run is left plain — "no news" and "passed" must not look alike. The
  colour survives the selection highlight, because the target you just ran is
  the one under the cursor.
- **`Less > Last run log`.** A bazel command's output is captured to a file as
  well as shown, so it can be read again later — coming back to a failure once
  you have looked at the code, or checking what a build actually said an hour
  ago. The action appears only when there is a log to read.
- The history and the logs live beside the target cache under
  `$MONOREPO/basalt_target_cache`, and survive restarts. One log per target,
  overwritten on each run, so the directory stays bounded.

### Changed

- Capturing output means bazel's stdout is a pipe rather than a terminal, so it
  prints plain lines instead of its curses progress display. The trade is
  deliberate: everything is still visible as it happens, and the saved log is
  readable afterwards instead of a screenful of cursor movements.

### Security

- The run history is JSON in the repository, and the log path it names is opened
  in a pager — so it is treated as untrusted input, exactly like the target
  cache. A log path outside the cache directory is refused, and the check is
  re-asserted at the point of use rather than trusted from load time. A history
  from another checkout, another schema version, or with an entry filed under
  the wrong label is discarded whole.
- Log file names are derived by flattening the label to `[A-Za-z0-9-]` and
  appending a digest of the original, so a hostile target name cannot steer the
  path and two labels that flatten alike still get separate files.
- Captured logs and the history are written 0644 — never group- or
  world-writable, since build output can carry paths and environment detail.

## [0.2.0] — 2026-08-15

### Added

- **The target list is a tree.** Each Bazel package now heads a group, drawn in
  its own colour, with that package's targets indented beneath it — named
  without repeating the path and tagged with their rule kind. Filtering keeps
  the shape rather than flattening it, so it is obvious *where* in the monorepo
  the matches live. A query matching a path highlights the heading; one matching
  a target name highlights the target.
- **Bazel actions wait before returning.** After a build, test or run, basalt
  reports the outcome and holds the screen with `Press Enter to return to
  basalt…`, so the command's output can be read instead of being wiped by the
  next frame. Escape and Ctrl+C dismiss it too. Editor and IDE actions do not
  pause — they are quit deliberately and leave nothing to review.

### Changed

- Arrow keys, Home/End and paging move between targets and step over the
  package headings, which are labels rather than choices. The heading a
  selection belongs to is kept on screen while scrolling.

## [0.1.0] — 2026-08-15

First release.

### Added

- Three-panel picker over every Bazel target in the monorepo: the target list,
  the actions available for the selected target, and a search field. Typing
  filters the list on every keystroke, whichever panel has focus.
- Literal case-insensitive search by default; `Ctrl+R` switches to regular
  expressions. A pattern that does not compile leaves the previous results on
  screen and reports the error, rather than blanking the list while you type.
- An action panel built from the target's rule kind and the tools actually
  installed: `Bazel > Run` only for binaries, `Bazel > Test` only for tests,
  `Bazel > Build` for every rule, a `Nano >` entry per Markdown file present in
  the target's directory, Claude Code with and without its permission bypass,
  and GoLand or CLion for Go and C/C++ targets.
- Actions suspend the UI and hand over the terminal, then restore it when the
  child exits — so `nano` and `claude` are fully interactive and one basalt
  session covers a whole build-edit-test loop.
- A JSON target index at `$MONOREPO/basalt_target_cache`, refreshed after a day,
  on `F5`, with `-nocache`, or when a `BUILD` file changes under `$MONOREPO/src`.
  The watcher is debounced, so a Dropbox sync of the tree does not cause a storm
  of re-indexing.
- `-index` prints what was indexed and exits, for checking the setup without
  entering the UI.

### Security

- Target names, labels and file names are stripped of control characters before
  they are rendered. A hostile name cannot repaint the screen, retitle the
  window, or send a device-status-report request that the terminal would answer
  into basalt's own input.
- Commands are executed as an argv, never through a shell, so `;`, `$(…)`,
  backticks and pipes in a target or file name are inert data.
- Labels that resolve outside `$MONOREPO` are refused, and a target carrying one
  is offered no actions.
- The cache is treated as untrusted input — it is a JSON file in the repository
  root that anyone with the checkout can edit. An entry whose display text
  disagrees with its label, or that points outside the checkout, is rejected and
  the index rebuilt. A cache from another checkout or another schema version is
  discarded rather than misread.
- The terminal is restored on every exit path, including signals and a failed
  child process.

### Known limitations

- Linux only: the terminal handling talks to `TCGETS`/`TCSETS` directly.
- The index covers `//src/...`, which is the tree the tool exists to navigate.

[0.1.0]: https://mattmierzwinski.com
