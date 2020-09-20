// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"bytes"
	"fmt"
	"html/template"
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
	nodeKindBlockImage                // "image::"
	lineKindEmpty                     //
	lineKindBlockTitle                // Line start with ".<alnum>"
	lineKindBlockComment              // Block start and end with "////"
	lineKindComment                   // 25: Line start with "//"
	lineKindAttribute                 // Line start with ":"
	lineKindHorizontalRule            // "'''", "---", "- - -", "***", "* * *"
	lineKindListContinue              // A single "+" line
	lineKindPageBreak                 // "<<<"
	lineKindStyle                     // Line start with "["
	lineKindStyleClass                // Custom style "[.x.y]"
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
	classes  []string
	Alt      string
	Width    string
	Height   string

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

func (node *adocNode) parseImage(line string) bool {
	attrBegin := strings.IndexByte(line, '[')
	if attrBegin < 0 {
		return false
	}
	attrEnd := strings.IndexByte(line, ']')
	if attrEnd < 0 {
		return false
	}
	name := strings.TrimRight(line[:attrBegin], " \t")
	node.raw.WriteString(name)

	attrs := strings.Split(line[attrBegin+1:attrEnd], ",")
	for x, attr := range attrs {
		switch x {
		case 0:
			node.Alt = strings.TrimSpace(attrs[0])
			if len(node.Alt) == 0 {
				dot := strings.IndexByte(name, '.')
				if dot > 0 {
					node.Alt = name[:dot]
				}
			}
		case 1:
			node.Width = attrs[1]
		case 2:
			node.Height = attrs[2]
		default:
			kv := strings.SplitN(attr, "=", 2)
			if len(kv) != 2 {
				continue
			}
			var (
				ok  bool
				val = strings.Trim(kv[1], `"`)
			)
			switch kv[0] {
			case "float", "align", "role":
				ok = true
				if val == "center" {
					val = "text-center"
				}
			}
			if ok {
				if len(val) > 0 {
					node.classes = append(node.classes, val)
				}
			}
		}
	}
	return true
}

func (node *adocNode) parseStyleClass(line string) {
	line = strings.Trim(line, "[]")
	parts := strings.Split(line, ".")
	for _, class := range parts {
		class = strings.TrimSpace(class)
		if len(class) > 0 {
			node.classes = append(node.classes, class)
		}
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

func (node *adocNode) toHTML(tmpl *template.Template, w io.Writer) (err error) {
	switch node.kind {
	case nodeKindPreamble:
		err = tmpl.ExecuteTemplate(w, "BEGIN_PREAMBLE", nil)
	case nodeKindSectionL1:
		err = tmpl.ExecuteTemplate(w, "BEGIN_SECTION_L1", node)
	case nodeKindSectionL2:
		err = tmpl.ExecuteTemplate(w, "BEGIN_SECTION_L2", node)
	case nodeKindSectionL3:
		err = tmpl.ExecuteTemplate(w, "BEGIN_SECTION_L3", node)
	case nodeKindSectionL4:
		err = tmpl.ExecuteTemplate(w, "BEGIN_SECTION_L4", node)
	case nodeKindSectionL5:
		err = tmpl.ExecuteTemplate(w, "BEGIN_SECTION_L5", node)
	case nodeKindParagraph:
		err = tmpl.ExecuteTemplate(w, "PARAGRAPH", node)
	case nodeKindLiteralParagraph, nodeKindBlockLiteralNamed, nodeKindBlockLiteralDelimiter:
		err = tmpl.ExecuteTemplate(w, "BLOCK_LITERAL", node)
	case nodeKindBlockListingDelimiter:
		err = tmpl.ExecuteTemplate(w, "BLOCK_LISTING", node)
	case nodeKindListOrdered:
		err = tmpl.ExecuteTemplate(w, "BEGIN_LIST_ORDERED", node)
	case nodeKindListUnordered:
		err = tmpl.ExecuteTemplate(w, "BEGIN_LIST_UNORDERED", node)
	case nodeKindListDescription:
		err = tmpl.ExecuteTemplate(w, "BEGIN_LIST_DESCRIPTION", node)
	case nodeKindListOrderedItem, nodeKindListUnorderedItem:
		err = tmpl.ExecuteTemplate(w, "BEGIN_LIST_ITEM", node)
	case nodeKindListDescriptionItem:
		err = tmpl.ExecuteTemplate(w, "BEGIN_LIST_DESCRIPTION_ITEM", node)
	case lineKindHorizontalRule:
		err = tmpl.ExecuteTemplate(w, "HORIZONTAL_RULE", nil)
	case lineKindPageBreak:
		err = tmpl.ExecuteTemplate(w, "PAGE_BREAK", nil)
	case nodeKindBlockImage:
		err = tmpl.ExecuteTemplate(w, "BLOCK_IMAGE", node)
	}
	if err != nil {
		return err
	}

	if node.child != nil {
		err = node.child.toHTML(tmpl, w)
		if err != nil {
			return err
		}
	}

	switch node.kind {
	case nodeKindPreamble:
		err = tmpl.ExecuteTemplate(w, "END_PREAMBLE", nil)
	case nodeKindSectionL1:
		err = tmpl.ExecuteTemplate(w, "END_SECTION_L1", nil)
	case nodeKindSectionL2, nodeKindSectionL3, nodeKindSectionL4, nodeKindSectionL5:
		err = tmpl.ExecuteTemplate(w, "END_SECTION", nil)
	case nodeKindListOrderedItem, nodeKindListUnorderedItem:
		err = tmpl.ExecuteTemplate(w, "END_LIST_ITEM", nil)
	case nodeKindListDescriptionItem:
		err = tmpl.ExecuteTemplate(w, "END_LIST_DESCRIPTION_ITEM", node)
	case nodeKindListOrdered:
		err = tmpl.ExecuteTemplate(w, "END_LIST_ORDERED", nil)
	case nodeKindListUnordered:
		err = tmpl.ExecuteTemplate(w, "END_LIST_UNORDERED", nil)
	case nodeKindListDescription:
		err = tmpl.ExecuteTemplate(w, "END_LIST_DESCRIPTION", node)
	}
	if err != nil {
		return err
	}

	if node.next != nil {
		err = node.next.toHTML(tmpl, w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (node *adocNode) GetListOrderedClass() string {
	switch node.level {
	case 2:
		return "loweralpha"
	case 3:
		return "lowerroman"
	case 4:
		return "upperalpha"
	case 5:
		return "upperroman"
	}
	return "arabic"
}

func (node *adocNode) GetListOrderedType() string {
	switch node.level {
	case 2:
		return "a"
	case 3:
		return "i"
	case 4:
		return "A"
	case 5:
		return "I"
	}
	return ""
}

func (node *adocNode) GenerateID(str string) string {
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

func (node *adocNode) Classes() string {
	if len(node.classes) == 0 {
		return ""
	}
	return " " + strings.Join(node.classes, " ")
}

func (node *adocNode) Content() string {
	return strings.TrimSpace(node.raw.String())
}

func (node *adocNode) IsStyleHorizontal() bool {
	return node.style&styleDescriptionHorizontal > 0
}

func (node *adocNode) IsStyleQandA() bool {
	return node.style&styleDescriptionQandA > 0
}

func (node *adocNode) Terminology() string {
	return node.rawTerm.String()
}

func (node *adocNode) Title() string {
	return node.rawTitle
}
