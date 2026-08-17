# basalt

Interactive picker for a Bazel monorepo's targets.

A large Bazel tree grows past the point where you can't remember where anything
lives, and typing `bazel run //src/go/apps/falcon:linux` from memory is tedious
and error-prone. `basalt` opens a three-panel view over every target in the
tree, filters it as you type, and runs the thing you picked.

```
┌─[BAZEL TARGETS 51/1359]────────┬────────────────────────┬─[TARGET ACTIONS]───────────────┐
│ 🔧 gcc/g++ 16.1.1              │ 🐹 go 1.25.8 → 1.26.5  │ Bazel > Run                    │
├────────────────────────────────┴────────────────────────┤ Bazel > Build                  │
│▦ //src/go/tools/basalt/cmd/basalt                     🐹│ Bazel > Test package           │
│  ✓ basalt · go_binary                                   │ Less > Last run log            │
│  → basalt_lib · go_library                              │ Nano > README.md               │
│  → basalt_rpm · pkg_rpm                               📦│ Claude > Run here (DSP)        │
│▦ //src/go/tools/basalt/internal/adapter/ansi          🐹│ Claude > Run here              │
│  ☠ ansi · go_library                                    │                                │
├─[SEARCH]────────────────────────────────────────────────┼────────────────────────────────┤
│ literal > tools/basalt█                                 │ 1359 targets · cache           │
└─────────────────────────────────────────────────────────┴────────────────────────────────┘
```

## About this repository

**basalt is exported from a private monorepo. No development happens on
GitHub.** This repository is a read-only mirror of the source plus the release
artifacts; the code is written, built, tested and released inside the monorepo
and pushed here as a snapshot, so it carries no development history.

Consequently:

- **Issues and feature requests are welcome** — they are read, and they do
  influence what gets built.
- **Pull requests cannot be accepted.** Patches sent through GitHub have nowhere
  to land: the branch here is regenerated wholesale on each export, and any
  commit made against it would be overwritten. If you want a change, open an
  issue describing it.

## Install

Download an artifact from [Releases](https://github.com/mateuszmierzwinski/basalt/releases).

```console
# Fedora / RHEL
$ sudo dnf install ./basalt-<version>.x86_64.rpm

# Any distribution — tarball
$ tar xzf basalt-<version>-linux-x86_64.tar.gz
$ install -Dm755 basalt-<version>/basalt ~/.local/bin/basalt
```

Linux x86-64 only. The binary is pure-static Go: it links nothing at runtime,
which is also why there is no AppImage — there is no runtime to bundle.

## Build from source

The only build dependency is a Go toolchain. There is no cgo, no code
generation step, and no Node or Python anywhere in the build.

```console
$ git clone https://github.com/mateuszmierzwinski/basalt.git
$ cd basalt
$ go build -o basalt ./cmd/basalt
$ ./basalt -index
```

Or straight from the module proxy:

```console
$ go install github.com/mateuszmierzwinski/basalt/cmd/basalt@latest
```

Run the tests with:

```console
$ go test ./...
```

A pure-static binary, matching what the releases ship:

```console
$ CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o basalt ./cmd/basalt
```

Note that the canonical build is Bazel, inside the monorepo — that is what
produces the release artifacts. The `go build` above is the supported path for
building this mirror on its own, and produces the same program.

## Requirements

- **`$MONOREPO`** set to the absolute path of the Bazel workspace you want to
  browse. basalt refuses to start without it — every path it reads and every
  command it runs is rooted there.
- **`bazel`** on `$PATH`. It is the source of truth for what targets exist.
- Optionally `nano`, `less`, `claude`, `goland` and `clion`. Each one that is
  missing simply has its actions hidden.

basalt indexes `//src/...`, so it expects a workspace whose code lives under a
top-level `src/` directory.

## The interface

The list is a tree: each Bazel package heads a group in its own colour, with
that package's targets indented beneath it, named without repeating the path and
tagged with their rule kind. Filtering keeps the shape — a package disappears
only when none of its targets match — so you can see at a glance *where* in the
tree your matches live, not just how many there are.

Every row is led by a marker — `▦` for a package, `✓` `☠` `→` for a target that
passed, failed or has never been run — so the state of the tree reads without
depending on colour at all. A heading is trailed by the languages of its
targets, `🐹` for Go and `🔧` for C and C++, alphabetically; a target that
produces an installable artifact is trailed by `📦`.

Above the list, one framed cell per compiler the selection needs answers "can
this machine build what I have selected". Go reads `🐹 go 1.25.8 → 1.26.5` —
required against installed, where the requirement is the `go` directive of the
nearest `go.mod` above the target — and C reads the installed `gcc`/`g++`. A
package written in one language gets one full-width frame; a package written in
both gets a cell each, C first. A version basalt cannot determine reads `?`, and
a cell whose requirement the host cannot meet turns red on its own, so the alarm
names which compiler is behind.

The cursor moves between targets and steps over the headings, and the heading a
selection belongs to is kept on screen as you scroll.

## Keys

| Key | Does |
|---|---|
| *any character* | filter the target list — works whichever panel has focus |
| `Tab` / `Shift+Tab` | move between the target list and the action panel |
| `↑` `↓` | move the selection |
| `PgUp` `PgDn` | move a screenful |
| `Home` `End` | jump to the first or last entry |
| `Enter` | from the targets, focus the actions; from the actions, run one |
| `Esc` | back out one step: actions → targets → clear the query → quit |
| `Ctrl+R` | switch between literal and regular-expression search |
| `Ctrl+U` / `Ctrl+W` | clear the query / delete the last word |
| `Ctrl+L` | repaint |
| `F5` | re-index from bazel |
| `Ctrl+C` | quit |

## Search

Literal, case-insensitive substring matching by default. `Ctrl+R` switches to
regular expressions, so `(.*)falcon(.*)` works, as do anchors like `engine$`. A
pattern that does not compile leaves the previous results on screen and shows
the error in the status line — you are usually mid-way through typing it.

## Actions

The action panel is rebuilt for whatever is selected, from the target's Bazel
rule kind and the tools actually installed:

| Action | Offered for |
|---|---|
| `Bazel > Run` | `*_binary` rules |
| `Bazel > Test` | `*_test` rules — runs that test alone |
| `Bazel > Test package` | any other rule with tests under its package — runs `//<package>/...` |
| `Bazel > Build` | every rule |
| `Nano > <file>.md` | each Markdown file in the target's directory |
| `Less > Last run log` | targets whose last bazel command left a log |
| `Claude > Run here` | any target, when `claude` is installed |
| `Claude > Run here (DSP)` | the same, with `--dangerously-skip-permissions` |
| `Goland > Open as bazel project` | `go_*` rules, when GoLand is installed |
| `Clion > Open as bazel project` | `cc_*` rules, when CLion is installed |

Choosing an action restores the terminal, hands it to the child process, and
takes it back when the child exits — so `nano` and `claude` are fully
interactive, and one basalt session covers a whole build-edit-test loop.

After a **Bazel** action basalt stops and waits:

```
── command finished successfully
Press Enter to return to basalt…
```

The build or test output is still on screen behind it, so you can read what
happened before the UI paints over it. Escape and Ctrl+C dismiss the prompt too.
Editor and IDE actions do not pause — you quit those deliberately and they leave
nothing to review, so an extra keystroke would only be in the way.

Bazel actions run from the workspace root, because a label resolves relative to
the workspace. Everything else runs in the target's own package directory.

## Run history

basalt remembers how each target's last bazel command went:

- a target that **passed** is drawn **green**,
- one that **failed** is drawn **red**,
- one that has **never been run** is left plain — "no news" and "passed" must
  not look alike.

The colour survives the selection highlight, since the target you just ran is
the one under the cursor. It also survives restarts.

The command's output is captured alongside, so `Less > Last run log` can show it
again later. One log per target, overwritten each run, so the directory stays
bounded by the number of targets you have ever run.

Everything basalt persists lives in one directory:

```
$MONOREPO/basalt_target_cache/
├── targets.json          the target index
├── history.json          how each target last ran
└── logs/                 one captured log per target
```

Deleting `basalt_target_cache` clears the history and the logs; it is all
rebuilt on demand.

## Indexing and the cache

The target list comes from:

```
bazel query 'kind(rule, //src/...:*)' --output=label_kind
```

which takes well under a second warm, for around 1350 targets. The result is
cached as readable JSON and reused for a day. It is refreshed when:

- the cache is older than 24 hours,
- you press `F5`,
- you pass `-nocache`,
- or a `BUILD` file changes anywhere under `$MONOREPO/src` — basalt watches the
  tree and re-indexes, debounced so a bulk filesystem sync does not trigger a
  storm.

Neither the cache nor the history holds an absolute path: entries are Bazel
labels, which are relative to the workspace, and a log is recorded by name. The
same checkout synced to two machines under different `$MONOREPO` paths therefore
shares one cache instead of invalidating it on every switch.

Both files are validated on load, and treated as untrusted input — they are
plain JSON that anything with write access to the checkout could edit, and a log
path ends up in a pager. A file that is corrupt, was written by a different
version, or names a label or a log outside the workspace is discarded and
rebuilt rather than trusted.

## Options

```
-index      print what was indexed and exit, without starting the UI
-nocache    ignore the cache and re-index from bazel
```

`-index` is the quickest way to check that basalt sees your workspace.

## License

MIT — see [LICENSE](LICENSE).
