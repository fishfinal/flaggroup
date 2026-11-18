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
	"bytes"
	"strings"
	"testing"

	flag "github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
)

type RendererSuite struct {
	suite.Suite
	fs *flag.FlagSet
}

func TestRendererSuite(t *testing.T) {
	suite.Run(t, new(RendererSuite))
}

func (s *RendererSuite) SetupTest() {
	s.fs = s.newTestFlagSet()
}

// newTestFlagSet builds a fresh FlagSet for each test case, so that group
// registration and annotation state never leaks between cases.
func (s *RendererSuite) newTestFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("taggler", flag.ContinueOnError)
	fs.String("yara-dir", "yara", "Specify the YARA file in directory")
	fs.String("yara-name", "global.yar", "Specify the YARA filename")
	fs.String("output", "result.txt", "Output file")
	fs.IntP("workers", "w", 4, "Number of concurrent workers")
	fs.BoolP("verbose", "v", false, "Enable verbose mode")
	fs.Bool("version", false, "Show version info")
	fs.Bool("stdout", false, "Print result to stdout")
	s.Require().NoError(fs.MarkHidden("version"))
	return fs
}

// renderToString renders the help output inside a test.
// Options passed through opts take precedence over the ones set here.
func (s *RendererSuite) renderToString(configure func(*Renderer), opts ...Option) string {
	var buf bytes.Buffer
	opts = append([]Option{WithWriter(&buf)}, opts...)
	r := New(s.fs, opts...)
	if configure != nil {
		configure(r)
	}
	s.Require().NoError(r.Render(), "Render() unexpected error")
	return buf.String()
}

func (s *RendererSuite) TestRenderKeepsDeclaredOrder() {
	got := s.renderToString(func(r *Renderer) {
		r.Group("yara", "YARA Options", "yara-name", "yara-dir")
	})

	s.Contains(got, "Usage: taggler [options]", "heading missing")
	s.Contains(got, "YARA Options:", "group title missing")

	// The explicitly declared order should override pflag's lexicographical order.
	nameIdx := strings.Index(got, "--yara-name")
	dirIdx := strings.Index(got, "--yara-dir")
	s.Require().GreaterOrEqual(nameIdx, 0, "yara-name flag missing")
	s.Require().GreaterOrEqual(dirIdx, 0, "yara-dir flag missing")
	s.Less(nameIdx, dirIdx, "declared order not preserved: yara-name at %d, yara-dir at %d", nameIdx, dirIdx)
}

func (s *RendererSuite) TestShorthandAndDefaultValue() {
	got := s.renderToString(func(r *Renderer) {
		r.Group("all", "All Options", "workers", "verbose", "yara-dir")
	})

	s.Contains(got, "-w, --workers int", "shorthand with type missing")
	s.Contains(got, "(default: 4)", "numeric default missing")
	s.Contains(got, `(default: "yara")`, "quoted string default missing")

	// A bool default of false is noise and should not be displayed.
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "--verbose") && strings.Contains(line, "default") {
			s.Fail("bool false default should be omitted", "got: %s", line)
		}
	}
}

func (s *RendererSuite) TestHiddenFlagIsSkipped() {
	got := s.renderToString(func(r *Renderer) {
		r.Group("all", "All Options", "output", "version")
		r.Ungrouped("Other Options")
	})
	s.NotContains(got, "--version", "hidden flag should not be rendered")
}

func (s *RendererSuite) TestUngroupedFallback() {
	got := s.renderToString(func(r *Renderer) {
		r.Group("yara", "YARA Options", "yara-dir")
		r.Ungrouped("Other Options")
	})

	s.Require().Contains(got, "Other Options:", "fallback group missing")
	for _, name := range []string{"--stdout", "--verbose", "--workers", "--output"} {
		s.Contains(got, name, "ungrouped flag %s missing from output", name)
	}
}

func (s *RendererSuite) TestEmptyGroupHasNoTitle() {
	got := s.renderToString(func(r *Renderer) {
		r.Group("yara", "YARA Options", "yara-dir")
		r.Group("empty", "Empty Options")
	})
	s.NotContains(got, "Empty Options", "empty group should not print a title")
}

func (s *RendererSuite) TestAnnotationBasedGrouping() {
	s.Require().NoError(SetGroup(s.fs, "yara", "yara-dir", "yara-name"), "SetGroup() error")

	got := s.renderToString(func(r *Renderer) {
		r.Group("yara", "YARA Options")
		r.Ungrouped("Other Options")
	})
	s.Contains(got, "--yara-dir", "annotation-driven grouping failed")
	s.Contains(got, "--yara-name", "annotation-driven grouping failed")
	s.Contains(got, "Other Options:", "remaining flags should fall back")
}

func (s *RendererSuite) TestSetGroupRejectsUnknownFlag() {
	s.Error(SetGroup(s.fs, "yara", "not-exist"), "SetGroup() should fail for an unknown flag name")
}

func (s *RendererSuite) TestStrictValidationReportsUnknownFlag() {
	var buf bytes.Buffer
	r := New(s.fs, WithWriter(&buf), WithStrictValidation())
	r.Group("yara", "YARA Options", "typo-name")
	s.Error(r.Render(), "Render() should fail in strict mode for an unknown flag")
}

func (s *RendererSuite) TestStrictValidationReportsDuplicateGroup() {
	var buf bytes.Buffer
	r := New(s.fs, WithWriter(&buf), WithStrictValidation())
	r.Group("a", "A", "output")
	r.Group("b", "B", "output")
	s.Error(r.Render(), "Render() should fail when a flag is claimed twice")
}

func (s *RendererSuite) TestFixedColumnWidth() {
	got := s.renderToString(func(r *Renderer) {
		r.Group("all", "All Options", "output", "yara-dir")
	}, WithColumnWidth(30))

	var first, second = -1, -1
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "  --output") {
			first = strings.Index(line, "Output file")
		}
		if strings.HasPrefix(line, "  --yara-dir") {
			second = strings.Index(line, "Specify")
		}
	}
	s.Require().Positive(first, "output description not found")
	s.Equal(first, second, "usage text should align at the same column")
}

func (s *RendererSuite) TestApplyBindsUsage() {
	var buf bytes.Buffer
	New(s.fs, WithWriter(&buf)).
		Group("yara", "YARA Options", "yara-dir").
		Apply()

	s.Require().NotNil(s.fs.Usage, "Apply() should bind FlagSet.Usage")
	s.fs.Usage()
	s.Contains(buf.String(), "YARA Options:", "Usage() did not render groups")
}
