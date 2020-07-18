// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"bytes"
	"fmt"
	"io"
)

const (
	nodeKindDocHeader int = iota
	nodeKindDocContent
	nodeKindPreamble
	nodeKindParagraph
	nodeKindLiteralParagraph
	nodeKindLiteralBlock
)

//
// adocNode is the building block of asciidoc document.
//
type adocNode struct {
	kind int
	raw  bytes.Buffer // unparsed content of node.

	parent *adocNode
	child  *adocNode
	next   *adocNode
	prev   *adocNode
}

//
// newAdocNode create new node and set the kind based on content of single
// line.
//
func newAdocNode(line string) (node *adocNode) {
	node = &adocNode{
		kind: nodeKindParagraph,
	}

	if line[0] == ' ' || line[0] == '\t' {
		node.kind = nodeKindLiteralParagraph
		node.raw.WriteString(line[1:])
		node.raw.WriteByte('\n')

	} else if line == "[literal]" {
		node.kind = nodeKindLiteralParagraph

	} else if line == "...." {
		node.kind = nodeKindLiteralBlock

	} else {
		node.raw.WriteString(line)
		node.raw.WriteByte('\n')
	}

	return node
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

func (node *adocNode) toHTML(w io.Writer) (err error) {
	switch node.kind {
	case nodeKindParagraph:
		_, err = fmt.Fprintf(w, `
<div class="paragraph">
	<p>%s</p>
</div>
`, node.raw.String())

	case nodeKindLiteralParagraph, nodeKindLiteralBlock:
		_, err = fmt.Fprintf(w, `
<div class="literalblock">
	<div class="content">
		<pre>%s</pre>
	</div>
</div>
`, node.raw.String())
	}

	if node.child != nil {
		err = node.child.toHTML(w)
		if err != nil {
			return err
		}
	}
	if node.next != nil {
		err = node.next.toHTML(w)
		if err != nil {
			return err
		}
	}

	return nil
}
