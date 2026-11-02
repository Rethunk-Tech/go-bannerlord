# AGENTS.md

Package `bannerlord` parses Bannerlord `SubModule.xml` files, orders modules, and reads and writes the launcher's
`LauncherData.xml`. A mod manager imports it; it imports no consumer package and does no file or network I/O of its
own: callers pass an `io.Reader` and write the bytes themselves.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH. The `windows-latest` test job is the only place
Windows-only behaviour runs.
No code is OS-specific; the Windows job runs the same tests.

## Shape

- `parseTree` (xmltree.go) is the only XML entry point. It owns the size cap, the DOCTYPE refusal and the depth cap,
  so no parser can skip them. The tree stays unexported to keep those limits the only way in.
- `validID` runs on every module id read or written. Ids become file names and launcher keys in callers, so
  separators and control characters never get through.
- `Sort` is pure and deterministic (official rank, then id). A caller's UI order must not depend on map iteration.
- `Sort` reports problems as `Issue`s and never fails: a manager wants to show every problem at once, and a missing
  dependency must not hide the rest of the list. Only cycle members are dropped from the order, since no valid
  position exists for them.
- `Version.Compare` ignores the channel prefix (`v`, `e`, ...): the game orders by the numbers only.
- `Launcher` keeps unmodelled elements as the generic tree and edits only `UserModData` entries, so a launcher
  update that adds fields does not lose them. Comments and namespace prefixes are not preserved; LauncherData.xml
  uses neither.
- Attribute lookups are case-insensitive: SubModule.xml in the wild mixes `Id`/`id` and `Order`/`order`.

## Files

| File | Holds |
| --- | --- |
| `doc.go` | Package doc and the cited upstream sources |
| `xmltree.go` | Bounded XML reader and writer, id validation, shared errors |
| `version.go` | `Version` parse, compare, render |
| `submodule.go` | `Module` and `ParseModule` |
| `order.go` | `Sort`, `Issue` |
| `launcher.go` | `Launcher`: read, edit and write LauncherData.xml |
