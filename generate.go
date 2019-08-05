// Copyright 2019, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

//go:generate go run generate_main.go

package ciigo

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/shuLhan/share/lib/memfs"
)

//
// Generate a static Go file to be used for building binary.
//
// It will convert all markup files inside root directory into HTML files,
// recursively; and read all the HTML files and files in "content/assets" and
// convert them into Go file in "out".
//
func Generate(root, out string) {
	htmlg := newHTMLGenerator()
	markupFiles := listMarkupFiles(root)

	htmlg.convertMarkupFiles(markupFiles, false)

	mfs, err := memfs.New(nil, defExcludes, true)
	if err != nil {
		log.Fatal("ciigo: Generate: " + err.Error())
	}

	err = mfs.Mount(root)
	if err != nil {
		log.Fatal("ciigo: Generate: " + err.Error())
	}

	err = mfs.GoGenerate("", out)
	if err != nil {
		log.Fatal("ciigo: Generate: " + err.Error())
	}
}

//
// listMarkupFiles find any markup files inside the content directory,
// recursively.
//
func listMarkupFiles(dir string) (markupFiles []*markupFile) {
	d, err := os.Open(dir)
	if err != nil {
		log.Fatal("ciigo: listMarkupFiles: os.Open: ", err)
	}

	fis, err := d.Readdir(0)
	if err != nil {
		log.Fatal("generate: " + err.Error())
	}

	for _, fi := range fis {
		name := fi.Name()

		if name == dirAssets {
			continue
		}
		if fi.IsDir() && name[0] != '.' {
			newdir := filepath.Join(dir, fi.Name())
			markupFiles = append(markupFiles, listMarkupFiles(newdir)...)
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		if !isExtensionMarkup(ext) {
			continue
		}
		if fi.Size() == 0 {
			continue
		}

		markupf := &markupFile{
			kind:     markupKind(ext),
			path:     filepath.Join(dir, name),
			info:     fi,
			basePath: filepath.Join(dir, strings.TrimSuffix(name, ext)),
		}
		markupFiles = append(markupFiles, markupf)
	}

	return markupFiles
}
