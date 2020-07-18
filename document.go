// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/shuLhan/share/lib/parser"
)

//
// Document represent content of asciidoc that has been parsed.
//
type Document struct {
	file string
	p    *parser.Parser

	authors   string
	revnumber string
	revdate   string
	metadata  map[string]string
	title     string

	header   *adocNode
	preamble *adocNode
	content  *adocNode

	// currNode point to current active node.
	// Its nil if its for the first time.
	currNode    *adocNode
	currLineNum int
	prevNode    *adocNode
}

//
// Open the ascidoc file and parse it.
//
func Open(file string) (doc *Document, err error) {
	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("ciigo.Open %s: %w", file, err)
	}

	doc = &Document{
		file:     file,
		metadata: make(map[string]string),
		content: &adocNode{
			kind: nodeKindDocContent,
		},
	}

	err = doc.Parse(raw)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

//
// Parse the content of asciidoc document.
//
func (doc *Document) Parse(content []byte) (err error) {
	doc.p = parser.New(string(content), "\n")

	line, c := doc.parseDocHeader()
	if doc.header != nil {
		doc.preamble = &adocNode{
			kind: nodeKindPreamble,
		}
		doc.currNode = doc.preamble
		doc.prevNode = doc.preamble
	}

	for {
		if len(line) == 0 {
			line, c = doc.line()
			if len(line) == 0 && c == 0 {
				// EOF
				break
			}

			if len(line) == 0 {
				continue
			}
		}

		// Check for comment block first, since we use HasPrefix to
		// check for single line comment.
		if line == "////" {
			doc.parseIgnoreCommentBlock()
			line = ""
			continue
		}
		if strings.HasPrefix(line, "//") {
			line = ""
			continue
		}

		if line[0] == ':' {
			if doc.parseMetadata(line) {
				line = ""
				continue
			}
		}

		node := newAdocNode(line)

		switch node.kind {
		case nodeKindParagraph:
			doc.consumeLinesUntil(node, "")
		case nodeKindLiteralParagraph:
			doc.consumeLinesUntil(node, "")
		case nodeKindLiteralBlock:
			doc.consumeLinesUntil(node, "....")
		}

		doc.currNode.addChild(node)

		line = ""
	}
	return nil
}

func (doc *Document) consumeLinesUntil(node *adocNode, term string) {
	for {
		line, c := doc.line()
		if len(line) == 0 && c == 0 {
			return
		}
		if line == term {
			return
		}
		node.raw.WriteString(line)
		node.raw.WriteByte('\n')
	}
}

func (doc *Document) line() (line string, c rune) {
	doc.currLineNum++
	return doc.p.Line()
}

func (doc *Document) parseDocHeader() (line string, c rune) {
	const (
		stateTitle int = iota
		stateAuthor
		stateRevision
		stateEnd
	)
	state := stateTitle
	for {
		line, c = doc.line()
		if len(line) == 0 && c == 0 {
			break
		}
		if len(line) == 0 {
			// Only allow empty line if state is title.
			if state == stateTitle {
				continue
			}
			return line, c
		}

		if strings.HasPrefix(line, "////") {
			doc.parseIgnoreCommentBlock()
			continue
		}
		if strings.HasPrefix(line, "//") {
			continue
		}
		if line[0] == ':' {
			if doc.parseMetadata(line) {
				continue
			}
		}
		switch state {
		case stateTitle:
			if !isTitle(line) {
				return line, c
			}
			doc.header = &adocNode{
				kind: nodeKindDocHeader,
			}
			doc.header.raw.WriteString(strings.TrimSpace(line[2:]))
			doc.title = doc.header.raw.String()
			state = stateAuthor
		case stateAuthor:
			doc.authors = line
			state = stateRevision
		case stateRevision:
			idx := strings.IndexByte(line, ',')
			if idx > 0 {
				doc.revnumber = line[:idx]
				doc.revdate = line[idx+1:]
			} else {
				doc.revnumber = line
			}
			state = stateEnd
		case stateEnd:
			return line, c
		}
	}
	return "", 0
}

//
// parseMetadata parse document metadata and return true if its valid.
//
func (doc *Document) parseMetadata(line string) bool {
	var rawkey []rune

	line = line[1:]
	for x, c := range line {
		if c == ':' {
			key := strings.TrimSpace(string(rawkey))
			doc.metadata[key] = strings.TrimSpace(line[x+1:])
			return true
		}
		rawkey = append(rawkey, c)
	}
	return false
}

func (doc *Document) parseIgnoreCommentBlock() {
	for {
		line, c := doc.p.Line()
		if line == "////" {
			return
		}
		if len(line) == 0 && c == 0 {
			return
		}
	}
}

//
// ToHTML convert the asciidoc to HTML.
//
func (doc *Document) ToHTML(w io.Writer) (err error) {
	err = doc.htmlBegin(w)
	if err != nil {
		return err
	}

	if doc.header != nil {
		_, err = fmt.Fprintf(w, `
<div id="header">
  <h1>%s</h1>
  <div class="details">
    <span id="author" class="author">%s</span>
    <br>
    <span id="revnumber">%s,</span>
    <span id="revdate">%s</span>
  </div>
</div>
`, doc.header.raw.String(), doc.authors, doc.revnumber, doc.revdate)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, `<div id="content">`)
	if err != nil {
		return err
	}

	if doc.preamble != nil {
		_, err = fmt.Fprintf(w, `<div id="preamble"><div class="sectionbody">`)
		if err != nil {
			return err
		}

		err = doc.preamble.toHTML(w)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, `</div></div>`)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, `</div>`)
	if err != nil {
		return err
	}

	err = doc.htmlEnd(w)

	return err
}

func (doc *Document) htmlBegin(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<meta name="generator" content="ciigo">
	<meta name="author" content="%s">
	<title>%s</title>
</head>
<body class="article">
`, doc.authors, doc.title)
	return err
}

func (doc *Document) htmlEnd(w io.Writer) (err error) {
	_, err = fmt.Fprintf(w, `</body></html>`)
	return err
}

func isTitle(line string) bool {
	return strings.HasPrefix(line, "= ") ||
		strings.HasPrefix(line, "=\t") ||
		strings.HasPrefix(line, "# ") ||
		strings.HasPrefix(line, "#\t")
}
