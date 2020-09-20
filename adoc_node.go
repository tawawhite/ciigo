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
	nodeKindBlockSidebar              // "****"
	nodeKindListOrdered               // 15: Wrapper.
	nodeKindListOrderedItem           // Line start with ". "
	nodeKindListUnordered             // Wrapper.
	nodeKindListUnorderedItem         // Line start with "* "
	nodeKindListDescription           // Wrapper.
	nodeKindListDescriptionItem       // 20: Line that has "::" + WSP
	nodeKindFigure                    //
	nodeKindImage                     //
	lineKindEmpty                     //
	lineKindBlockTitle                // Line start with ".<alnum>"
	lineKindBlockComment              // 25: Block start and end with "////"
	lineKindComment                   // Line start with "//"
	lineKindAttribute                 // Line start with ":"
	lineKindListContinue              // A single "+" line
	lineKindStyle                     // Line start with "["
	lineKindText                      //
)

//
// adocNode is the building block of asciidoc document.
//
type adocNode struct {
	kind     int
	level    int          // The number of dot for ordered list, or star '*' for unordered list.
	raw      bytes.Buffer // unparsed content of node.
	rawTerm  bytes.Buffer
	rawTitle string
	style    int64

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

func (node *adocNode) parseListUnordered(line string) {
	x := 0
	for ; x < len(line); x++ {
		if line[x] == '*' {
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

func (node *adocNode) parseListDescription(line string) {
	var (
		x int
		c rune
	)
	for x, c = range line {
		if c == ':' {
			break
		}
		node.rawTerm.WriteRune(c)
	}
	line = line[x:]
	for x, c = range line {
		if c == ':' {
			node.level++
			continue
		}
		break
	}
	// Skip leading spaces...
	if x < len(line)-1 {
		line = line[x:]
	} else {
		line = ""
	}
	for x, c = range line {
		if c == ' ' || c == '\t' {
			continue
		}
		break
	}
	node.level -= 2
	if len(line) > 0 {
		node.raw.WriteString(line[x:])
	}
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
		title := strings.TrimSpace(node.raw.String())
		_, err = fmt.Fprintf(w, `<div class="sect1">
<h2 id="%s">%s</h2>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindSectionL2:
		title := strings.TrimSpace(node.raw.String())
		_, err = fmt.Fprintf(w, `<div class="sect2">
<h3 id="%s">%s</h3>
`, toID(title), title)

	case nodeKindSectionL3:
		title := strings.TrimSpace(node.raw.String())
		_, err = fmt.Fprintf(w, `<div class="sect3">
<h4 id="%s">%s</h4>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindSectionL4:
		title := strings.TrimSpace(node.raw.String())
		_, err = fmt.Fprintf(w, `<div class="sect4">
<h5 id="%s">%s</h5>
<div class="sectionbody">
`, toID(title), title)

	case nodeKindSectionL5:
		title := strings.TrimSpace(node.raw.String())
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

	case nodeKindLiteralParagraph, nodeKindBlockLiteralNamed,
		nodeKindBlockLiteralDelimiter:
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

	case nodeKindListUnordered:
		_, err = fmt.Fprintf(w, `<div class="ulist">
`)
		err = node.toHTMLBlockTitle(w)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, `<ul>
`)
		if err != nil {
			return err
		}

	case nodeKindListDescription:
		err = node.htmlBeginListDescription(w)

	case nodeKindListOrderedItem, nodeKindListUnorderedItem:
		_, err = fmt.Fprintf(w, `<li>
<p>%s</p>
`, strings.TrimSpace(node.raw.String()))

	case nodeKindListDescriptionItem:
		err = node.htmlBeginListDescriptionItem(w)
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

	case nodeKindSectionL1:
		_, err = fmt.Fprintf(w, `</div>
</div>
`)
	case nodeKindSectionL2, nodeKindSectionL3,
		nodeKindSectionL4, nodeKindSectionL5:
		_, err = fmt.Fprintf(w, `</div>
`)
	case nodeKindListOrderedItem, nodeKindListUnorderedItem:
		_, err = fmt.Fprintf(w, `</li>
`)
	case nodeKindListDescriptionItem:
		err = node.htmlEndListDescriptionItem(w)
	case nodeKindListOrdered:
		_, err = fmt.Fprintf(w, `</ol>
</div>
`)
	case nodeKindListUnordered:
		_, err = fmt.Fprintf(w, `</ul>
</div>
`)
	case nodeKindListDescription:
		err = node.htmlEndListDescription(w)
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

func (node *adocNode) htmlBeginListDescription(w io.Writer) (err error) {
	if node.style&styleDescriptionHorizontal > 0 {
		_, err = fmt.Fprintf(w, `<div class="hdlist">
`)

		err = node.toHTMLBlockTitle(w)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "<table>\n")
	} else {
		_, err = fmt.Fprintf(w, `<div class="dlist">
`)

		err = node.toHTMLBlockTitle(w)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "<dl>\n")
	}
	return err
}

func (node *adocNode) htmlEndListDescription(w io.Writer) (err error) {
	if node.style&styleDescriptionHorizontal > 0 {
		_, err = fmt.Fprintf(w, "</table>\n</div>\n")
	} else {
		_, err = fmt.Fprintf(w, "</dl>\n</div>\n")
	}
	return err
}

func (node *adocNode) htmlBeginListDescriptionItem(w io.Writer) (err error) {
	if node.style&styleDescriptionHorizontal > 0 {
		_, err = fmt.Fprintf(w, `<tr>
<td class="hdlist1">
%s
</td>
<td class="hdlist2">
`, node.rawTerm.String())
		if err != nil {
			return err
		}
		if node.raw.Len() > 0 {
			_, err = fmt.Fprintf(w, `<p>%s</p>
`, node.raw.String())
		}
	} else {
		_, err = fmt.Fprintf(w, `<dt class="hdlist1">%s</dt>
<dd>
`, node.rawTerm.String())
		if err != nil {
			return err
		}
		if node.raw.Len() > 0 {
			_, err = fmt.Fprintf(w, `<p>%s</p>
`, strings.TrimSpace(node.raw.String()))
		}
	}
	return err
}

func (node *adocNode) htmlEndListDescriptionItem(w io.Writer) (err error) {
	if node.style&styleDescriptionHorizontal > 0 {
		_, err = fmt.Fprintf(w, `</td>
</tr>
`)
	} else {
		_, err = fmt.Fprintf(w, `</dd>
`)
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
			if id[len(id)-1] != '_' {
				id = append(id, '_')
			}
		}
	}
	return strings.TrimRight(string(id), "_")
}
