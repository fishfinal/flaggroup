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

// Command example demonstrates the two grouping styles supported by flaggroup:
// annotating flags at their definition site (a good fit for projects whose
// options are spread across several files), and declaring group members
// explicitly at render time (a good fit for tools whose options all live in a
// single file).
package main

import (
	"fmt"
	"os"

	"github.com/fishfinal/flaggroup"
	flag "github.com/spf13/pflag"
)

const (
	groupYARA   = "yara"
	groupOutput = "output"
)

func main() {
	fs := flag.NewFlagSet("taggler", flag.ExitOnError)

	yaraDir := fs.String("yara-dir", "yara", "Specify the YARA file in directory")
	yaraName := fs.String("yara-name", "global.yar", "Specify the YARA filename")
	output := fs.String("output", "result.txt", "Output file")
	workers := fs.IntP("workers", "w", 4, "Number of concurrent workers")
	verbose := fs.BoolP("verbose", "v", false, "Enable verbose mode")
	stdout := fs.Bool("stdout", false, "Print result to stdout instead of a file")

	// Style one: annotate flags where they are defined; the annotated members
	// are collected automatically when the help is rendered.
	flaggroup.MustSetGroup(fs, groupYARA, "yara-dir", "yara-name")

	// Internal debugging flags stay out of the help output.
	fs.Bool("profile", false, "Collect runtime profile (internal)")
	_ = fs.MarkHidden("profile")

	// Style two: declare members explicitly at render time; declaration order
	// is the display order.
	// stdout is intentionally left ungrouped to show the fallback section that
	// catches ungrouped flags.
	flaggroup.New(fs,
		flaggroup.WithHeading("Usage: taggler [options] <target>"),
		flaggroup.WithFooter("Report issues: https://github.com/fishfinal/taggler/issues"),
	).
		Group(groupYARA, "YARA Options").
		Group(groupOutput, "Output Options").Add("output", "workers", "verbose").
		Ungrouped("Other Options").
		Apply()

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	fmt.Printf("yara-dir=%s yara-name=%s output=%s workers=%d verbose=%v stdout=%v\n",
		*yaraDir, *yaraName, *output, *workers, *verbose, *stdout)
}
