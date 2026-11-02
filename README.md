<h1 align="center">go-bannerlord</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

Read Mount & Blade II: Bannerlord module metadata (SubModule.xml), compute a valid module load order, and read and write the launcher's LauncherData.xml. Package `bannerlord`, module `github.com/Rethunk-Tech/go-bannerlord`.

It reads a module's `SubModule.xml` (id, version, dependencies, load-before/after hints, DLLs), sorts a set of modules
into a load order the game accepts, and edits the launcher's enabled list without losing fields it does not know.
Input is treated as untrusted. No file or network access.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-bannerlord
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- `ParseModule`: SubModule.xml with typed `Version`, dependencies and metadata.
- `Version.Compare`: Bannerlord versions such as `v1.2.3.0`, `e1.0.0` and `v1.2.*`.
- `Sort`: deterministic topological order, official modules first, problems returned as typed `Issue`s.
- `Launcher`: read and rewrite `LauncherData.xml`'s enabled set and order, keeping unknown elements.
- Bounded input: size and depth limits, DOCTYPE refused, ids validated.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
