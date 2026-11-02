# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- `ParseModule` and `ParseLauncher` read XML from mod archives and the user's game folder. A crafted document
  must not cause an unbounded allocation (input is capped at `MaxXMLSize`, nesting at 32), entity expansion
  (DOCTYPE and any directive are refused; `encoding/xml` supports no external entities), or a panic.
- Module ids from either file (and ids passed to `SetMods`) must not carry path separators, `..`, control characters
  or anything outside letters, digits and `.-_`, since callers use them as names and keys.
- `Module.DLLs` entries never contain a path separator or NUL.
- `Sort` must terminate and not panic on any module set, including cycles, duplicates and missing dependencies.

## What is out of scope

The caller is trusted to bound the archive it extracts and to decide which module files to open. Declared
dependencies and versions are claims by the mod author, not verified. A cycle or missing dependency in a mod set is
reported by design, not a vulnerability.
