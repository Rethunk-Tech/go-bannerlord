# Changelog

Notable changes to `go-bannerlord`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-11-02

### Added

- `ParseModule` and `Version`: read SubModule.xml (ids, versions, dependencies, load-before/after, DLLs) with bounded, DOCTYPE-free XML parsing.
- `Sort`: dependency and load-order sort with official modules first; cycles, missing, outdated, incompatible and duplicate modules are typed issues.
- `Launcher`: read and rewrite LauncherData.xml's enabled mod set and order, keeping unknown elements.

[Unreleased]: https://github.com/Rethunk-Tech/go-bannerlord/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Rethunk-Tech/go-bannerlord/releases/tag/v0.1.0
