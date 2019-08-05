// Copyright 2019, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package main

import (
	"github.com/shuLhan/ciigo"
)

func main() {
	srv := ciigo.NewServer("./content", ":8080")

	srv.Start()
}
