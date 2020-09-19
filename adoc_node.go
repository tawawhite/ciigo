// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"
)

const (
	nodeKindUnknown               int = iota
	nodeKindDocHeader                 // Wrapper.
	nodeKindPreamble                  // Wrapper.
	nodeKindDocContent                // Wrapper.
	nodeKindSectionL1                 // Line started with "=="
	nodeKindSectionL2                 // 5: Line started with "==="
	nodeKindSectionL3                 // Line started with "===="
	nodeKindSectionL4                 // Line started with "====="
	nodeKindSectionL5                 // Line started with "======"
	nodeKindParagraph                 // Wrapper.
	nodeKindLiteralParagraph          // 10: Line start with space
	nodeKindBlockListingDelimiter     // Block start and end with "----"
	nodeKindBlockLiteralNamed         // Block start with "[literal]", end with ""
	nodeKindBlockLiteralDelimiter     // Block start and end with "...."
	nodeKindListOrdered               // Wrapper.
	nodeKindListOrderedItem           // 15: Line start with ". "
	nodeKindFigure                    //
	nodeKindImage                     //
	lineKindEmpty                     //
	lineKindBlockTitle                // Line start with ".<alnum>"
	lineKindText                      // 20:
	lineKindBlockComment              // Block start and end with "////"
	lineKindComment                   // Line start with "//"
	lineKindAttribute                 // Line start with ":"
	lineKindListContinue              // A single "+" line
)

//
// adocNode is the building block of asciidoc document.
//
type adocNode struct {
	kind     int
	level    int          // The number of dot for ordered list, or star '*' for unordered list.
	raw      bytes.Buffer // unparsed content of node.
	rawTitle string

	parent *adocNode
	child  *adocNode
	next   *adocNode
	prev   *adocNode
}

func (node *adocNode) parseListOrdered(line string) {
	x := 0
	for ; x < len(line); x++ {
		if line[x] == '.' {
			node.level++
			continue
		}
		if line[x] == ' ' || line[x] == '\t' {
			break
		}
	}
	for ; x < len(line); x++ {
		if line[x] == ' ' || line[x] == '\t' {
			continue
		}
		break
	}
	node.raw.WriteString(line[x:])
	node.raw.WriteByte('\n')
}

func (node *adocNode) addChild(child *adocNode) {
	child.parent = node
	child.next = nil
	child.prev = nil

	if node.child == nil {
		node.child = child
	} else {
		c := node.child
		for c.next != nil {
			c = c.next
		}
		c.next = child
		child.prev = c
	}
}

func (node *adocNode) debug(n int) {
	for x := 0; x < n; x++ {
		fmt.Printf("\t")
	}
	fmt.Printf("node: %3d %s\n", node.kind, node.raw.String())
	if node.child != nil {
		node.child.debug(n + 1)
	}
	if node.next != nil {
		node.next.debug(n)
	}
}

func (node *adocNode) toHTML(w io.Writer) (err error) {
	switch node.kind {
	case nodeKindPreamble:
		_, err = fmt.Fprintf(w, `<div id="preamble">
<div class="sectionbody">
`)

	case nodeKindSectionL1:
		title := node.raw.String()
		_, err = fmt.Fprintf(w, `<div class="sect1">
<h2 id="%s">%s</h2>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindSectionL2:
		title := node.raw.String()
		_, err = fmt.Fprintf(w, `<div class="sect2">
<h3 id="%s">%s</h3>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindSectionL3:
		title := node.raw.String()
		_, err = fmt.Fprintf(w, `<div class="sect3">
<h4 id="%s">%s</h4>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindSectionL4:
		title := node.raw.String()
		_, err = fmt.Fprintf(w, `<div class="sect4">
<h5 id="%s">%s</h5>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindSectionL5:
		title := node.raw.String()
		_, err = fmt.Fprintf(w, `<div class="sect5">
<h6 id="%s">%s</h6>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindParagraph:
		_, err = fmt.Fprintf(w, `<div class="paragraph">
`)
		err = node.toHTMLBlockTitle(w)
		_, err = fmt.Fprintf(w, `<p>%s</p>
</div>
`, strings.TrimSpace(node.raw.String()))

	case nodeKindLiteralParagraph, nodeKindBlockLiteralNamed, nodeKindBlockLiteralDelimiter:
		_, err = fmt.Fprintf(w, `<div class="literalblock">
<div class="content">
<pre>%s</pre>
</div>
</div>
`, strings.TrimRight(node.raw.String(), " \t\r\n"))

	case nodeKindBlockListingDelimiter:
		_, err = fmt.Fprintf(w, `<div class="listingblock">
<div class="content">
<pre>%s</pre>
</div>
</div>
`, strings.TrimSpace(node.raw.String()))

	case nodeKindListOrdered:
		class, tipe := getListOrderedClassType(node.level)

		_, err = fmt.Fprintf(w, `<div class="olist %s">
`, class)
		if err != nil {
			return err
		}
		err = node.toHTMLBlockTitle(w)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, `<ol class="%s"`, class)
		if err != nil {
			return err
		}
		if len(tipe) > 0 {
			_, err = fmt.Fprintf(w, ` type="%s"`, tipe)
		}
		_, err = fmt.Fprintf(w, `>
`)

	case nodeKindListOrderedItem:
		_, err = fmt.Fprintf(w, `<li>
<p>%s</p>
`, strings.TrimSpace(node.raw.String()))
	}
	if err != nil {
		return err
	}

	if node.child != nil {
		err = node.child.toHTML(w)
		if err != nil {
			return err
		}
	}

	switch node.kind {
	case nodeKindPreamble:
		_, err = fmt.Fprintf(w, `</div>
</div>
`)

	case nodeKindSectionL1, nodeKindSectionL2, nodeKindSectionL3,
		nodeKindSectionL4, nodeKindSectionL5:
		_, err = fmt.Fprintf(w, `</div>
</div>
`)
	case nodeKindListOrderedItem:
		_, err = fmt.Fprintf(w, `</li>
`)
	case nodeKindListOrdered:
		_, err = fmt.Fprintf(w, `</ol>
</div>
`)
	}
	if err != nil {
		return err
	}

	if node.next != nil {
		err = node.next.toHTML(w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (node *adocNode) toHTMLBlockTitle(w io.Writer) (err error) {
	if len(node.rawTitle) > 0 {
		_, err = fmt.Fprintf(w, `<div class="title">%s</div>
`, node.rawTitle)
	}
	return err
}

func getListOrderedClassType(level int) (class, tipe string) {
	switch level {
	case 2:
		return "loweralpha", "a"
	case 3:
		return "lowerroman", "i"
	case 4:
		return "upperalpha", "A"
	case 5:
		return "upperroman", "I"
	}
	return "arabic", ""
}

func toID(str string) string {
	id := make([]rune, 0, len(str)+1)
	id = append(id, '_')
	for _, c := range strings.ToLower(str) {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			id = append(id, c)
		} else {
			id = append(id, '_')
		}
	}
	return string(id)
}
