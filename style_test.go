// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"testing"

	"github.com/shuLhan/share/lib/test"
)

func TestStyleConst(t *testing.T) {
	test.Assert(t, "styleSectionColophon", 1, styleSectionColophon, true)
	test.Assert(t, "styleSectionAbstract", 2, styleSectionAbstract, true)
	test.Assert(t, "styleSectionPreface", 4, styleSectionPreface, true)
	test.Assert(t, "styleSectionDedication", 8, styleSectionDedication, true)
}
