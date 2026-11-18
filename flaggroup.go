// Copyright 2026 fishfinal <contact.fishfinal@gmail.com>;
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package flaggroup renders command-line flags in groups on top of
// github.com/spf13/pflag.
//
// On its own pflag lists every flag flat in alphabetical order, so once a single
// command has more than a dozen flags, --help degrades into a wall of text that
// is hard to read. This package makes use of the Flag.Annotations field that
// pflag has long reserved but never consumes itself, and renders flags grouped
// by function, so that even a single-command CLI can have help output with clear
// sections and tidy alignment.
//
// Minimal usage:
//
//	fs := flag.NewFlagSet("myapp", flag.ExitOnError)
//	fs.String("input", "", "Input file path")
//	fs.String("output", "out.txt", "Output file")
//	flaggroup.New(fs).
//		Group("input", "Input Options", "input").
//		Group("output", "Output Options", "output").
//		Ungrouped("Other Options").
//		Apply()
package flaggroup

import (
	"fmt"
	"io"
	"strings"

	flag "github.com/spf13/pflag"
)

// AnnotationKey is the annotation key under which group names are stored.
// A flag may belong to several groups at once, so the value is a []string.
const AnnotationKey = "group"

// MinColumnWidth is the minimum alignment width of the flag-name column,
// which keeps short flag names from crowding together.
const MinColumnWidth = 24

// Group represents one flag section in the help output.
// It also forwards the grouping and rendering methods of its Renderer, so an
// entire chained call can be written in one go:
//
//	flaggroup.New(fs).
//		Group("yara", "YARA Options").Add("yara-dir", "yara-name").
//		Group("output", "Output Options").Add("output", "verbose").
//		Ungrouped("Other Options").
//		Apply()
type Group struct {
	r       *Renderer
	name    string
	title   string
	members []string
}

// Name returns the group identifier, that is, the value written to the annotation.
func (g *Group) Name() string { return g.name }

// Title returns the heading this group is displayed with in the help output.
func (g *Group) Title() string { return g.title }

// Add appends flag names belonging to this group; the order they are passed in
// is the order they are displayed in.
// It returns the receiver to support chained calls.
func (g *Group) Add(names ...string) *Group {
	g.members = append(g.members, names...)
	return g
}

// Group declares the next section from a Group, forwarding to its Renderer.
func (g *Group) Group(name, title string, members ...string) *Group {
	return g.r.Group(name, title, members...)
}

// Ungrouped declares the fallback section, forwarding to the owning Renderer.
func (g *Group) Ungrouped(title string) *Renderer {
	return g.r.Ungrouped(title)
}

// Apply binds the rendered output to FlagSet.Usage, forwarding to the owning Renderer.
func (g *Group) Apply() *Renderer {
	return g.r.Apply()
}

// Render renders the help output, forwarding to the owning Renderer.
func (g *Group) Render() error {
	return g.r.Render()
}

// Renderer renders a *flag.FlagSet as help text, grouped into sections.
// The zero value is not usable; create one with New.
type Renderer struct {
	fs           *flag.FlagSet
	w            io.Writer
	columnWidth  int
	heading      string
	footer       string
	groups       []*Group
	ungrouped    string
	hasUngrouped bool
	strict       bool
}

// Group declares a flag section and optionally specifies its members directly.
//
// name is the group identifier written to the annotation; title is the heading
// shown in the help output. If no members are given through the variadic
// argument or through Add, flags whose annotations carry this name are
// collected automatically at render time, which supports the decoupled style of
// declaring groups at the definition site and only listing group names at the
// render site.
func (r *Renderer) Group(name, title string, members ...string) *Group {
	g := &Group{r: r, name: name, title: title, members: members}
	r.groups = append(r.groups, g)
	return g
}

// Ungrouped declares the fallback section: every flag not claimed by any group
// is rendered here in alphabetical order.
// This is the key safeguard against the trap of adding a flag, forgetting to
// group it, and having it silently vanish from the help output.
func (r *Renderer) Ungrouped(title string) *Renderer {
	r.ungrouped = title
	r.hasUngrouped = true
	return r
}

// Render renders the help output and returns any error encountered while writing.
func (r *Renderer) Render() error {
	if r == nil || r.fs == nil {
		return fmt.Errorf("flaggroup: Render called without a FlagSet")
	}
	if err := r.validate(); err != nil {
		return err
	}

	claimed := make(map[string]bool)
	type block struct {
		title string
		flags []*flag.Flag
	}
	var blocks []block

	width := MinColumnWidth
	auto := r.columnWidth == 0
	if !auto {
		width = r.columnWidth
	}

	measure := func(flags []*flag.Flag) {
		if !auto {
			return
		}
		for _, f := range flags {
			if n := len(flagLabel(f)); n > width {
				width = n
			}
		}
	}

	for _, g := range r.groups {
		flags := r.resolve(g, claimed)
		measure(flags)
		blocks = append(blocks, block{title: g.title, flags: flags})
	}
	if r.hasUngrouped {
		flags := r.rest(claimed)
		measure(flags)
		blocks = append(blocks, block{title: r.ungrouped, flags: flags})
	}

	e := &errWriter{w: r.w}
	heading := r.heading
	if heading == "" {
		heading = fmt.Sprintf("Usage: %s [options]", r.fs.Name())
	}
	e.printf("%s\n", heading)

	for _, b := range blocks {
		if len(b.flags) == 0 {
			// Empty groups print no heading, so the help output never contains
			// hollow sections.
			continue
		}
		e.printf("\n%s:\n", b.title)
		for _, f := range b.flags {
			e.printf("%s\n", renderLine(f, width))
		}
	}
	if r.footer != "" {
		e.printf("\n%s\n", r.footer)
	}
	return e.err
}

// UsageFunc returns a function that can be assigned to FlagSet.Usage directly.
func (r *Renderer) UsageFunc() func() {
	return func() { _ = r.Render() }
}

// Apply binds the rendered output to FlagSet.Usage and returns the receiver to
// support chained calls.
func (r *Renderer) Apply() *Renderer {
	r.fs.Usage = r.UsageFunc()
	return r
}

// resolve determines the flags in a group. Explicit members win and are
// returned in the order they were passed; when no members are given, flags are
// collected from annotations in pflag's lexicographical order.
func (r *Renderer) resolve(g *Group, claimed map[string]bool) []*flag.Flag {
	var out []*flag.Flag
	if len(g.members) > 0 {
		for _, name := range g.members {
			f := r.fs.Lookup(name)
			if f == nil || f.Hidden {
				continue
			}
			claimed[name] = true
			out = append(out, f)
		}
		return out
	}
	r.fs.VisitAll(func(f *flag.Flag) {
		if f.Hidden || claimed[f.Name] {
			return
		}
		for _, name := range f.Annotations[AnnotationKey] {
			if name == g.name {
				claimed[f.Name] = true
				out = append(out, f)
				return
			}
		}
	})
	return out
}

// rest returns the flags not yet claimed by any group, in pflag's
// lexicographical order.
func (r *Renderer) rest(claimed map[string]bool) []*flag.Flag {
	var out []*flag.Flag
	r.fs.VisitAll(func(f *flag.Flag) {
		if f.Hidden || claimed[f.Name] {
			return
		}
		out = append(out, f)
	})
	return out
}

// validate checks, in strict mode, that the flag names referenced by groups
// exist and are not claimed by more than one group.
func (r *Renderer) validate() error {
	if !r.strict {
		return nil
	}
	seen := make(map[string]string)
	for _, g := range r.groups {
		for _, name := range g.members {
			if r.fs.Lookup(name) == nil {
				return fmt.Errorf("flaggroup: group %q refers to unknown flag %q", g.name, name)
			}
			if prev, ok := seen[name]; ok {
				return fmt.Errorf("flaggroup: flag %q is claimed by both group %q and %q", name, prev, g.name)
			}
			seen[name] = g.name
		}
	}
	return nil
}

// flagLabel builds the display name of a flag in the help output,
// such as "-v, --verbose" or "--output string".
func flagLabel(f *flag.Flag) string {
	var b strings.Builder
	if f.Shorthand != "" {
		b.WriteString("-")
		b.WriteString(f.Shorthand)
		b.WriteString(", ")
	}
	b.WriteString("--")
	b.WriteString(f.Name)
	if t := f.Value.Type(); t != "bool" {
		b.WriteString(" ")
		b.WriteString(t)
	}
	return b.String()
}

// defaultSuffix builds the default-value suffix. A false bool, an empty string
// and an empty slice are omitted to keep the output free of noise.
func defaultSuffix(f *flag.Flag) string {
	dv := f.DefValue
	if dv == "" {
		return ""
	}
	typ := f.Value.Type()
	switch typ {
	case "bool":
		if dv == "false" {
			return ""
		}
	case "stringSlice", "stringArray":
		if dv == "[]" {
			return ""
		}
	}
	if typ == "string" {
		return fmt.Sprintf(" (default: %q)", dv)
	}
	return fmt.Sprintf(" (default: %v)", dv)
}

// renderLine renders a single flag description line.
func renderLine(f *flag.Flag, width int) string {
	line := fmt.Sprintf("  %-*s  %s%s", width, flagLabel(f), f.Usage, defaultSuffix(f))
	if f.Deprecated != "" {
		line += fmt.Sprintf(" (DEPRECATED: %s)", f.Deprecated)
	}
	return line
}

// errWriter records the first write error; later writes short-circuit.
type errWriter struct {
	w   io.Writer
	err error
}

func (e *errWriter) printf(format string, args ...interface{}) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, args...)
}
