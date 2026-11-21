# Changelog

Notable changes to `go-bannerlord`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.3] - 2026-11-21

### Changed

- Load order picks the next ready module from a heap instead of re-sorting each time.

### Fixed

- BOM-marked UTF-16 documents are read.
- Mixed-case metadata attributes no longer drop load order.
- `Sort` never checks `Metadata.Version`.
- `ParseVersion` refuses a `-0` part.
- `DLLName` also refuses `COM0`, `LPT0`, superscript ports and `CONIN$`/`CONOUT$`; module ids that are Windows reserved device names are refused too.
- Booleans parse like `bool.Parse`, not only lowercase `true`.

## [0.1.2] - 2026-11-07

### Fixed

- The same code as 0.1.1, released again: 0.1.1's tag moved after the Go module proxy had cached it, so `go get` from the proxy and from GitHub disagreed on its checksum. Use 0.1.2.

## [0.1.1] - 2026-11-07

### Added

- Fuzz targets for `ParseModule` and `ParseLauncher`, seeded from `testdata`.

### Fixed

- `.` and `..` are no longer valid module ids.
- `DLLName` must be a plain file name: no drive or stream colon, Windows device name, wildcard, control character, or trailing dot or space.
- Text split by comments in a large document parses in linear time.
- `Launcher.SetMods` keeps every entry of a module listed twice in the launcher file, with its unknown fields.

## [0.1.0] - 2026-11-02

### Added

- `ParseModule` and `Version`: read SubModule.xml (ids, versions, dependencies, load-before/after, DLLs) with bounded, DOCTYPE-free XML parsing.
- `Sort`: dependency and load-order sort with official modules first; cycles, missing, outdated, incompatible and duplicate modules are typed issues.
- `Launcher`: read and rewrite LauncherData.xml's enabled mod set and order, keeping unknown elements.

[Unreleased]: https://github.com/Rethunk-Tech/go-bannerlord/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/Rethunk-Tech/go-bannerlord/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/Rethunk-Tech/go-bannerlord/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/Rethunk-Tech/go-bannerlord/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/Rethunk-Tech/go-bannerlord/releases/tag/v0.1.0
