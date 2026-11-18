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

package flaggroup

import (
	"io"
	"os"

	flag "github.com/spf13/pflag"
)

// Option configures a Renderer.
type Option func(*Renderer)

// New creates a grouped renderer for the given FlagSet.
// By default it writes to os.Stderr, sizes the flag-name column automatically,
// and prints no footer.
func New(fs *flag.FlagSet, opts ...Option) *Renderer {
	r := &Renderer{fs: fs, w: os.Stderr}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithWriter sets the output destination, which defaults to os.Stderr.
// Use it when the help output should go to stdout or to a buffer, as in unit tests.
func WithWriter(w io.Writer) Option {
	return func(r *Renderer) { r.w = w }
}

// WithColumnWidth fixes the width of the flag-name column and turns off
// automatic sizing. A non-positive value keeps the automatic behavior.
func WithColumnWidth(n int) Option {
	return func(r *Renderer) {
		if n > 0 {
			r.columnWidth = n
		}
	}
}

// WithHeading sets the usage line at the top of the help output, for example
// "Usage: taggler [options] <target>".
// When it is not set, "Usage: <flagset name> [options]" is printed instead.
func WithHeading(text string) Option {
	return func(r *Renderer) {
		r.heading = text
	}
}

// WithFooter sets a note printed at the end of the help output, such as a
// feedback channel or a documentation link.
func WithFooter(text string) Option {
	return func(r *Renderer) {
		r.footer = text
	}
}

// WithStrictValidation enables strict validation: before rendering, it checks
// that the flag names referenced by groups exist and are not declared by more
// than one group, and returns an error as soon as a problem is found.
// Enabling it during development is recommended, since it prevents a misspelled
// flag name from making the flag silently vanish from the help output.
func WithStrictValidation() Option {
	return func(r *Renderer) {
		r.strict = true
	}
}
