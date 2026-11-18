<div align="center">

# flaggroup

**Grouped `--help` sections for `spf13/pflag`** — stop shipping alphabetically sorted walls of flags.

[![Go Reference](https://pkg.go.dev/badge/github.com/fishfinal/flaggroup.svg)](https://pkg.go.dev/github.com/fishfinal/flaggroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/fishfinal/flaggroup)](https://goreportcard.com/report/github.com/fishfinal/flaggroup)
[![License](https://img.shields.io/github/license/go-defender/flaggroup)](https://github.com/fishfinal/flaggroup/blob/main/LICENSE)
[![CI](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml/badge.svg)](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml)

**English** | [简体中文](README_zh.md) | [日本語](README_ja.md) | [Español](README_es.md) | [Deutsch](README_de.md)

</div>

Adds **grouped flags** to `github.com/spf13/pflag`: a single command's `--help` is rendered in functional sections instead of an alphabetically sorted wall of flags.

pflag reserves an `Annotations map[string][]string` field on the `Flag` struct but never consumes it itself. `flaggroup` is that consumer — it stores grouping information in annotations and hands `FlagSet.Usage` over to a configurable renderer.

## Installation

```bash
go get github.com/fishfinal/flaggroup
```

## Quick start

```go
package main

import (
	flag "github.com/spf13/pflag"
	"github.com/fishfinal/flaggroup"
)

func main() {
	fs := flag.NewFlagSet("myapp", flag.ExitOnError)
	input := fs.String("input", "", "Input file path")
	output := fs.String("output", "out.txt", "Output file")
	verbose := fs.BoolP("verbose", "v", false, "Enable verbose mode")

	flaggroup.New(fs).
		Group("input", "Input Options", "input").
		Group("output", "Output Options").Add("output", "verbose").
		Ungrouped("Other Options").
		Apply()

	fs.Parse(nil)
	_, _, _ = input, output, verbose
}
```

`Apply()` binds the render function to `fs.Usage`, so from then on both `--help` and the output printed on parse errors are grouped automatically:

```
Usage: myapp [options]

Input Options:
  --input string       Input file path

Output Options:
  --output string      Output file (default: "out.txt")
  -v, --verbose        Enable verbose mode

Other Options:
  --help               help for myapp
```

## Two grouping styles

### Style one: declare members explicitly at render time

This is the most straightforward option when all flags live in a single file. The order passed to `Add` is the display order, and it overrides pflag's default lexicographical order:

```go
r := flaggroup.New(fs)
r.Group("yara", "YARA Options").Add("yara-dir", "yara-name")
r.Group("output", "Output Options").Add("output", "workers", "verbose")
```

### Style two: annotate flags at the definition site

When flags are spread across several packages and each module registers its own, let every module declare where its flags belong and only list the groups at render time:

```go
// Inside each module's flag registration function
flaggroup.MustSetGroup(fs, "yara", "yara-dir", "yara-name")

// At render time
r.Group("yara", "YARA Options")   // no members: collected automatically from annotations
```

> Do not call `fs.SetAnnotation` directly: pflag's implementation returns silently when a flag name is misspelled, swallowing the error entirely. `SetGroup` validates that each flag exists first and returns an error.

## Cobra integration

Cobra's `Group` groups **subcommands**, not flags. To group a subcommand's `--help` as well, attach the renderer through `SetUsageFunc`:

```go
rootCmd.SetUsageFunc(func(c *cobra.Command) error {
	return flaggroup.New(c.Flags(),
		flaggroup.WithHeading(fmt.Sprintf("Usage: %s [flags] [args]", c.CommandPath())),
	).
		Group("main", "Main Options", "yara-dir", "output").
		Ungrouped("Other Options").
		Render()
})
```

If you need to keep Cobra's native command list, Example section and other blocks, take over only the flags part instead:

```go
rootCmd.SetUsageTemplate(rootCmd.UsageTemplate() + "")  // keep the template
rootCmd.SetFlagErrorFunc(...)                            // handle the error path separately
```

The safer approach is to leave Cobra's templates untouched and override `SetUsageFunc` only on the leaf commands that need grouping.

## API overview

| Function / method | Description |
| --- | --- |
| `New(fs, ...Option)` | Creates a renderer; writes to `os.Stderr` with automatic column sizing by default |
| `(*Renderer) Group(name, title, members...)` | Declares a section and returns a `*Group` so `Add` can be chained |
| `(*Group) Add(names...)` | Appends group members; the order is the display order |
| `(*Renderer) Ungrouped(title)` | Declares the fallback section that catches ungrouped flags |
| `(*Renderer) Render() error` | Renders and returns any write error |
| `(*Renderer) Apply()` | Binds the output to `fs.Usage` |
| `(*Renderer) UsageFunc() func()` | Returns a function that can be assigned to `fs.Usage` directly |
| `SetGroup / MustSetGroup` | Writes annotations safely and reports misspelled flag names |
| `GroupsOf(f)` | Returns all groups a flag belongs to |

| Option | Description |
| --- | --- |
| `WithWriter(w)` | Sets the output destination (defaults to `os.Stderr`) |
| `WithColumnWidth(n)` | Fixes the flag-name column width and turns off automatic sizing |
| `WithHeading(text)` | Customizes the usage line at the top |
| `WithFooter(text)` | Appends a footer note |
| `WithStrictValidation()` | Validates before rendering that flag names exist and are not claimed twice |

## Rendering rules

- **Shorthand**: when `Shorthand` is set, the flag renders as `-v, --verbose`
- **Type hint**: types other than `bool` get their type appended to the name, as in `--output string`
- **Defaults**: strings are quoted with `%q`; a `false` bool, an empty string and an empty slice are not displayed
- **Hidden**: flags passed to `MarkHidden` appear in no group
- **Deprecated**: `(DEPRECATED: <message>)` is appended at the end of the line
- **Empty groups**: a group with no visible flags is skipped along with its title
- **Column width**: by default the maximum length of all visible flag names, floored at 24, aligned across groups

## Testing

```bash
go test ./...
go run ./example --help
```
