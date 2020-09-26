// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"testing"

	"github.com/shuLhan/share/lib/test"
)

func TestParseBlockAttribute(t *testing.T) {
	cases := []struct {
		in  string
		exp map[string]string
	}{{
		in:  "",
		exp: nil,
	}, {
		in:  "[]",
		exp: make(map[string]string),
	}, {
		in: `[a]`,
		exp: map[string]string{
			"a": "1",
		},
	}, {
		in: `[a=2]`,
		exp: map[string]string{
			"a": "2",
		},
	}, {
		in: `[a=2,b="c, d",e,f=3]`,
		exp: map[string]string{
			"a": "2",
			"b": "c, d",
			"e": "1",
			"f": "3",
		},
	}}

	for _, c := range cases {
		got := parseBlockAttribute(c.in)
		test.Assert(t, "parseBlockAttribute", c.exp, got, true)
	}
}
