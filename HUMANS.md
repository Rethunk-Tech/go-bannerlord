# HUMANS.md

Run and use `go-bannerlord`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-bannerlord
```

```go
package main

import (
	"fmt"
	"os"

	bannerlord "github.com/Rethunk-Tech/go-bannerlord"
)

func main() {
	var mods []bannerlord.Module
	for _, path := range os.Args[1:] {
		f, err := os.Open(path)
		if err != nil {
			fmt.Println(err)
			return
		}
		m, err := bannerlord.ParseModule(f)
		_ = f.Close()
		if err != nil {
			fmt.Println(path, err)
			return
		}
		mods = append(mods, m)
	}
	order, issues := bannerlord.Sort(mods)
	for _, is := range issues {
		fmt.Println("issue:", is)
	}
	fmt.Println(order)
}
```

To write the result into the launcher, `ParseLauncher` the existing file, `SetMods(bannerlord.Singleplayer, order)`,
then `Marshal` and save the bytes.

## Contracts

| Call | Contract |
| --- | --- |
| `ParseModule` | Refuses documents over `MaxXMLSize`, any DOCTYPE, an invalid id, a DLL name with a path separator, an unknown metadata order. Returns `ErrTooLarge`, `ErrDoctype` or `ErrInvalid`-wrapped errors. A UTF-8 BOM is accepted. |
| `ParseVersion` | A prefix letter (a, b, e, v, d) and three or four numbers; a `*` part and all after it are zero. `Compare` orders by prefix (a < b < e < v < d), then numbers. Refuses oversized numbers. `ParseModule` treats an unparsable version as unspecified, as BUTR does. |
| `Sort` | Never panics or errors; returns `(order, issues)`. Dependencies and `LoadBeforeThis` precede, `LoadAfterThis` follows. Modules with `Official` set first, then by id. Cycle members are omitted from `order` and named in one `CycleDetected` issue. Missing dependencies are issues; the dependent stays in the order. |
| `ParseLauncher` | Same input limits as `ParseModule`; every module id must be valid. |
| `Launcher.SetMods` | Selected ids come first in the given order; other existing entries follow, unselected. Unknown elements and fields survive. Invalid or repeated ids are refused. |
| `Launcher.Marshal` | Indented XML with a declaration; comments and namespace prefixes are not kept. |

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
