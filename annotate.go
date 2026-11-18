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
	"fmt"

	flag "github.com/spf13/pflag"
)

// SetGroup annotates the named flags with a group, so that a Renderer can
// collect them automatically from their annotations.
//
// Unlike calling FlagSet.SetAnnotation directly, which silently returns when
// the flag name is misspelled and swallows the error entirely, this function
// checks that each flag exists first and returns an error on a misspelled name.
func SetGroup(fs *flag.FlagSet, group string, names ...string) error {
	for _, name := range names {
		if fs.Lookup(name) == nil {
			return fmt.Errorf("flaggroup: cannot annotate unknown flag %q (group %q)", name, group)
		}
		fs.SetAnnotation(name, AnnotationKey, []string{group})
	}
	return nil
}

// MustSetGroup behaves like SetGroup but panics on error.
// It is meant for places where a configuration mistake should abort the program
// immediately, such as an init function or the start of main.
func MustSetGroup(fs *flag.FlagSet, group string, names ...string) {
	if err := SetGroup(fs, group, names...); err != nil {
		panic(err)
	}
}

// GroupsOf returns the names of all groups a flag belongs to,
// or nil if the flag is not grouped.
func GroupsOf(f *flag.Flag) []string {
	if f == nil {
		return nil
	}
	return f.Annotations[AnnotationKey]
}
