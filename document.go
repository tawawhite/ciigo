// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/shuLhan/share/lib/ascii"
	"github.com/shuLhan/share/lib/parser"
)

//
// Document represent content of asciidoc that has been parsed.
//
type Document struct {
	file string
	p    *parser.Parser

	Author       string
	Title        string
	RevNumber    string
	RevSeparator string
	RevDate      string
	LastUpdated  string
	attributes   map[string]string
	lineNum      int

	header  *adocNode
	content *adocNode

	nodeCurrent *adocNode
	nodeParent  *adocNode
	prevKind    int
	kind        int
}

//
// Open the ascidoc file and parse it.
//
func Open(file string) (doc *Document, err error) {
	fi, err := os.Stat(file)
	if err != nil {
		return nil, err
	}

	raw, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("ciigo.Open %s: %w", file, err)
	}

	doc = &Document{
		file:        file,
		LastUpdated: fi.ModTime().Round(time.Second).Format("2006-01-02 15:04:05 Z0700"),
		attributes:  make(map[string]string),
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

	spaces, line, c := doc.parseHeader()

	doc.nodeParent = &adocNode{
		kind: nodeKindPreamble,
	}
	doc.content.addChild(doc.nodeParent)
	doc.nodeCurrent = &adocNode{
		kind: nodeKindUnknown,
	}

	for {
		if len(line) == 0 {
			spaces, line, c = doc.line()
			if len(line) == 0 && c == 0 {
				return nil
			}
		}

		switch doc.kind {
		case lineKindEmpty:
			if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
			}
			continue
		case lineKindBlockComment:
			doc.parseIgnoreCommentBlock()
			line = ""
			continue
		case lineKindComment:
			line = ""
			continue
		case lineKindHorizontalRule:
			if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
			}
			doc.nodeCurrent.kind = doc.kind
			doc.terminateCurrentNode()
			line = ""
			continue
		case lineKindPageBreak:
			if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
			}
			doc.nodeCurrent.kind = doc.kind
			doc.terminateCurrentNode()
			line = ""
			continue
		case lineKindAttribute:
			if doc.parseAttribute(line, true) {
				doc.terminateCurrentNode()
			} else if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.nodeCurrent.raw.WriteString(line)
			}
			line = ""
			continue
		case lineKindStyle:
			styleName, styleKind := parseStyle(line)
			if styleKind != styleNone {
				doc.nodeCurrent.style |= styleKind
				if isStyleAdmonition(styleKind) {
					doc.nodeCurrent.setStyleAdmonition(styleName, styleKind)
				}
				line = ""
				continue
			}

		case lineKindStyleClass:
			doc.nodeCurrent.parseStyleClass(line)
			line = ""
			continue

		case lineKindText, lineKindListContinue:
			if doc.nodeCurrent.kind == nodeKindUnknown {
				doc.nodeCurrent.kind = nodeKindParagraph
				doc.nodeCurrent.raw.WriteString(line)
				doc.nodeCurrent.raw.WriteByte('\n')
				line, c = doc.consumeLinesUntil(
					doc.nodeCurrent,
					lineKindEmpty,
					[]int{
						nodeKindBlockListingDelimiter,
						nodeKindBlockLiteralDelimiter,
						nodeKindBlockLiteralNamed,
						lineKindListContinue,
					})
				doc.terminateCurrentNode()
			} else {
				doc.nodeCurrent.raw.WriteString(line)
				line = ""
			}
			continue

		case lineKindBlockTitle:
			doc.nodeCurrent.rawTitle = line[1:]
			line = ""
			continue

		case lineKindAdmonition:
			doc.nodeCurrent.kind = lineKindAdmonition
			doc.nodeCurrent.parseLineAdmonition(line)
			line, c = doc.consumeLinesUntil(
				doc.nodeCurrent,
				lineKindEmpty,
				[]int{
					nodeKindBlockListingDelimiter,
					nodeKindBlockLiteralDelimiter,
					nodeKindBlockLiteralNamed,
					lineKindListContinue,
				})
			doc.terminateCurrentNode()
			continue

		case nodeKindSectionL1, nodeKindSectionL2,
			nodeKindSectionL3, nodeKindSectionL4,
			nodeKindSectionL5:

			if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
			}
			doc.nodeCurrent.kind = doc.kind
			doc.nodeCurrent.raw.WriteString(
				// BUG: "= =a" could become "a", it should be "=a"
				strings.TrimLeft(line, "= \t"),
			)

			var expParent = doc.kind - 1
			for doc.nodeParent.kind != expParent {
				doc.nodeParent = doc.nodeParent.parent
				if doc.nodeParent == nil {
					doc.nodeParent = doc.content
					break
				}
			}
			doc.nodeParent.addChild(doc.nodeCurrent)
			doc.nodeParent = doc.nodeCurrent
			doc.nodeCurrent = &adocNode{
				kind: nodeKindUnknown,
			}
			line = ""
			continue

		case nodeKindLiteralParagraph:
			if doc.nodeCurrent.kind == nodeKindUnknown &&
				doc.nodeCurrent.IsStyleAdmonition() {
				doc.nodeCurrent.kind = nodeKindParagraph
				doc.nodeCurrent.raw.WriteString(spaces + line)
				doc.nodeCurrent.raw.WriteByte('\n')
				line, c = doc.consumeLinesUntil(
					doc.nodeCurrent,
					lineKindEmpty,
					[]int{
						nodeKindBlockListingDelimiter,
						nodeKindBlockLiteralDelimiter,
						nodeKindBlockLiteralNamed,
						lineKindListContinue,
					})
			} else {
				doc.nodeCurrent.kind = doc.kind
				doc.nodeCurrent.raw.WriteString(spaces + line)
				doc.nodeCurrent.raw.WriteByte('\n')
				line, c = doc.consumeLinesUntil(
					doc.nodeCurrent,
					lineKindEmpty,
					[]int{
						nodeKindBlockListingDelimiter,
						nodeKindBlockLiteralDelimiter,
						nodeKindBlockLiteralNamed,
					})
			}
			doc.terminateCurrentNode()

		case nodeKindBlockLiteralDelimiter:
			doc.nodeCurrent.kind = doc.kind
			line, c = doc.consumeLinesUntil(
				doc.nodeCurrent,
				doc.kind, nil)
			doc.terminateCurrentNode()

		case nodeKindBlockLiteralNamed:
			doc.nodeCurrent.kind = doc.kind
			line, c = doc.consumeLinesUntil(
				doc.nodeCurrent,
				lineKindEmpty, nil)
			doc.terminateCurrentNode()

		case nodeKindBlockListingDelimiter:
			doc.nodeCurrent.kind = doc.kind
			line, c = doc.consumeLinesUntil(
				doc.nodeCurrent,
				doc.kind, nil)
			doc.terminateCurrentNode()
			doc.nodeParent.debug(1)

		case nodeKindListOrderedItem:
			line, c = doc.parseListOrdered(doc.nodeParent,
				doc.nodeCurrent.rawTitle, line)
			doc.terminateCurrentNode()
			continue

		case nodeKindListUnorderedItem:
			line, c = doc.parseListUnordered(doc.nodeParent, line)
			doc.terminateCurrentNode()
			continue

		case nodeKindListDescriptionItem:
			line, c = doc.parseListDescription(doc.nodeParent, line)
			doc.terminateCurrentNode()
			continue

		case nodeKindBlockImage:
			if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
			}
			if doc.nodeCurrent.parseImage(line) {
				doc.nodeCurrent.kind = doc.kind
				doc.terminateCurrentNode()
			} else {
				doc.nodeCurrent.kind = nodeKindParagraph
				doc.nodeCurrent.raw.WriteString("image::" + line)
				doc.nodeCurrent.raw.WriteByte('\n')
			}

		case nodeKindBlockOpen, nodeKindBlockExample:
			doc.nodeCurrent.kind = doc.kind
			doc.parseBlock(doc.nodeCurrent, doc.kind)
			doc.terminateCurrentNode()

		case nodeKindBlockVideo:
			if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
			}
			if doc.nodeCurrent.parseVideo(line) {
				doc.nodeCurrent.kind = doc.kind
				doc.terminateCurrentNode()
			} else {
				doc.nodeCurrent.kind = nodeKindParagraph
				doc.nodeCurrent.raw.WriteString("video::" + line)
				doc.nodeCurrent.raw.WriteByte('\n')
			}

		case nodeKindBlockAudio:
			if doc.nodeCurrent.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
			}
			if doc.nodeCurrent.parseBlockAudio(line) {
				doc.nodeCurrent.kind = doc.kind
				doc.terminateCurrentNode()
			} else {
				doc.nodeCurrent.kind = nodeKindParagraph
				doc.nodeCurrent.raw.WriteString("audio::" + line)
				doc.nodeCurrent.raw.WriteByte('\n')
			}
		}
		line = ""
	}
	doc.terminateCurrentNode()

	return nil
}

//
// parseListOrdered parser the content as list until it found line that is not
// list-item.
// On success it will return non-empty line and terminator character.
//
func (doc *Document) parseListOrdered(parent *adocNode, title, line string) (
	got string, c rune,
) {
	list := &adocNode{
		kind:     nodeKindListOrdered,
		rawTitle: title,
	}
	listItem := &adocNode{
		kind: nodeKindListOrderedItem,
	}
	listItem.parseListOrdered(line)
	list.level = listItem.level
	list.addChild(listItem)
	parent.addChild(list)

	line = ""

	for {
		if len(line) == 0 {
			_, line, c = doc.line()
			if len(line) == 0 && c == 0 {
				break
			}
		}

		if doc.kind == lineKindBlockComment {
			doc.parseIgnoreCommentBlock()
			line = ""
			continue
		}
		if doc.kind == lineKindComment {
			line = ""
			continue
		}
		if doc.kind == lineKindListContinue {
			var node *adocNode
			node, line, c = doc.parseListBlock()
			if node != nil {
				listItem.addChild(node)
			}
			continue
		}
		if doc.kind == lineKindEmpty {
			// Keep going, maybe next line is still a list.
			continue
		}
		if doc.kind == lineKindStyle {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}
		if doc.kind == nodeKindListOrderedItem {
			node := &adocNode{
				kind: nodeKindListOrderedItem,
			}
			node.parseListOrdered(line)
			if listItem.level == node.level {
				list.addChild(node)
				listItem = node
				line = ""
				continue
			}

			// Case:
			// ... Parent
			// . child
			// ... Next list
			parentListItem := parent
			for parentListItem != nil {
				if parentListItem.kind == doc.kind && parentListItem.level == node.level {
					return line, c
				}
				parentListItem = parentListItem.parent
			}

			line, c = doc.parseListOrdered(listItem, "", line)
			continue
		}
		if doc.kind == nodeKindListUnorderedItem {
			node := &adocNode{
				kind: nodeKindListUnorderedItem,
			}
			node.parseListUnordered(line)

			// Case:
			// . Parent
			// * child
			// . Next list
			parentListItem := parent
			for parentListItem != nil {
				if parentListItem.kind == doc.kind && parentListItem.level == node.level {
					return line, c
				}
				parentListItem = parentListItem.parent
			}

			line, c = doc.parseListUnordered(listItem, line)
			continue
		}
		if doc.kind == nodeKindListDescriptionItem {
			node := &adocNode{
				kind: doc.kind,
			}
			node.parseListDescription(line)

			parentListItem := parent
			for parentListItem != nil {
				if parentListItem.kind == doc.kind && parentListItem.level == node.level {
					return line, c
				}
				parentListItem = parentListItem.parent
			}

			line, c = doc.parseListDescription(listItem, line)
			continue
		}

		if doc.kind == lineKindAdmonition {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}
		if doc.kind == lineKindText {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}
		if doc.kind == nodeKindBlockLiteralNamed {
			if doc.prevKind == lineKindEmpty {
				break
			}
			node := &adocNode{
				kind: doc.kind,
			}
			line, c = doc.consumeLinesUntil(node,
				lineKindEmpty,
				[]int{
					nodeKindListOrderedItem,
					nodeKindListUnorderedItem,
				})
			listItem.addChild(node)
			continue
		}
		if doc.kind == nodeKindBlockListingDelimiter ||
			doc.kind == nodeKindBlockExample {
			break
		}
		if doc.kind == nodeKindSectionL1 ||
			doc.kind == nodeKindSectionL2 ||
			doc.kind == nodeKindSectionL3 ||
			doc.kind == nodeKindSectionL4 ||
			doc.kind == nodeKindSectionL5 {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}

		listItem.raw.WriteString(strings.TrimSpace(line))
		listItem.raw.WriteByte('\n')
		line = ""
	}

	return line, c
}

func (doc *Document) parseListUnordered(parent *adocNode, line string) (
	got string, c rune,
) {
	list := &adocNode{
		kind:     nodeKindListUnordered,
		rawTitle: doc.nodeCurrent.rawTitle,
	}
	listItem := &adocNode{
		kind: nodeKindListUnorderedItem,
	}
	listItem.parseListUnordered(line)
	list.level = listItem.level
	list.addChild(listItem)
	parent.addChild(list)

	line = ""

	for {
		if len(line) == 0 {
			_, line, c = doc.line()
			if len(line) == 0 && c == 0 {
				break
			}
		}

		if doc.kind == lineKindBlockComment {
			doc.parseIgnoreCommentBlock()
			line = ""
			continue
		}
		if doc.kind == lineKindComment {
			line = ""
			continue
		}
		if doc.kind == lineKindListContinue {
			var node *adocNode
			node, line, c = doc.parseListBlock()
			if node != nil {
				listItem.addChild(node)
			}
			continue
		}
		if doc.kind == lineKindEmpty {
			// Keep going, maybe next line is still a list.
			continue
		}
		if doc.kind == lineKindStyle {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}

		if doc.kind == nodeKindListOrderedItem {
			node := &adocNode{
				kind: nodeKindListOrderedItem,
			}
			node.parseListOrdered(line)

			// Case:
			// . Parent
			// * child
			// . Next list
			parentListItem := parent
			for parentListItem != nil {
				if parentListItem.kind == doc.kind && parentListItem.level == node.level {
					return line, c
				}
				parentListItem = parentListItem.parent
			}

			line, c = doc.parseListOrdered(listItem, "", line)
			continue
		}

		if doc.kind == nodeKindListUnorderedItem {
			node := &adocNode{
				kind: nodeKindListUnorderedItem,
			}
			node.parseListUnordered(line)
			if listItem.level == node.level {
				list.addChild(node)
				listItem = node
				line = ""
				continue
			}

			// Case:
			// *** Parent
			// * child
			// *** Next list
			parentListItem := parent
			for parentListItem != nil {
				if parentListItem.kind == doc.kind && parentListItem.level == node.level {
					return line, c
				}
				parentListItem = parentListItem.parent
			}

			line, c = doc.parseListUnordered(listItem, line)
			continue
		}
		if doc.kind == nodeKindListDescriptionItem {
			node := &adocNode{
				kind: doc.kind,
			}
			node.parseListDescription(line)

			parentListItem := parent
			for parentListItem != nil {
				if parentListItem.kind == doc.kind && parentListItem.level == node.level {
					return line, c
				}
				parentListItem = parentListItem.parent
			}

			line, c = doc.parseListDescription(listItem, line)
			continue
		}

		if doc.kind == lineKindAdmonition {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}
		if doc.kind == lineKindText {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}
		if doc.kind == nodeKindBlockLiteralNamed {
			if doc.prevKind == lineKindEmpty {
				break
			}
			node := &adocNode{
				kind: doc.kind,
			}
			line, c = doc.consumeLinesUntil(node,
				lineKindEmpty,
				[]int{
					nodeKindListOrderedItem,
					nodeKindListUnorderedItem,
				})
			listItem.addChild(node)
			continue
		}
		if doc.kind == nodeKindBlockListingDelimiter ||
			doc.kind == nodeKindBlockExample {
			break
		}
		if doc.kind == nodeKindSectionL1 ||
			doc.kind == nodeKindSectionL2 ||
			doc.kind == nodeKindSectionL3 ||
			doc.kind == nodeKindSectionL4 ||
			doc.kind == nodeKindSectionL5 {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}

		listItem.raw.WriteString(strings.TrimSpace(line))
		listItem.raw.WriteByte('\n')
		line = ""
	}

	return line, c
}

func (doc *Document) parseListDescription(parent *adocNode, line string) (
	got string, c rune,
) {
	list := &adocNode{
		kind:     nodeKindListDescription,
		rawTitle: doc.nodeCurrent.rawTitle,
		style:    doc.nodeCurrent.style,
	}
	listItem := &adocNode{
		kind:  nodeKindListDescriptionItem,
		style: list.style,
	}
	listItem.parseListDescription(line)
	list.level = listItem.level
	list.addChild(listItem)
	parent.addChild(list)

	line = ""
	for {
		if len(line) == 0 {
			_, line, c = doc.line()
			if len(line) == 0 && c == 0 {
				break
			}
		}
		if doc.kind == lineKindBlockComment {
			doc.parseIgnoreCommentBlock()
			line = ""
			continue
		}
		if doc.kind == lineKindComment {
			line = ""
			continue
		}
		if doc.kind == lineKindListContinue {
			var node *adocNode
			node, line, c = doc.parseListBlock()
			if node != nil {
				listItem.addChild(node)
			}
			continue
		}
		if doc.kind == lineKindEmpty {
			// Keep going, maybe next line is still a list.
			continue
		}
		if doc.kind == lineKindStyle {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}

		if doc.kind == nodeKindListOrderedItem {
			line, c = doc.parseListOrdered(listItem, "", line)
			continue
		}
		if doc.kind == nodeKindListUnorderedItem {
			line, c = doc.parseListUnordered(listItem, line)
			continue
		}
		if doc.kind == nodeKindListDescriptionItem {
			node := &adocNode{
				kind:  nodeKindListDescriptionItem,
				style: list.style,
			}
			node.parseListDescription(line)
			if listItem.level == node.level {
				list.addChild(node)
				listItem = node
				line = ""
				continue
			}
			parentListItem := parent
			for parentListItem != nil {
				if parentListItem.kind == doc.kind && parentListItem.level == node.level {
					return line, c
				}
				parentListItem = parentListItem.parent
			}

			line, c = doc.parseListDescription(listItem, line)
			continue
		}

		if doc.kind == lineKindText {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}
		if doc.kind == nodeKindBlockLiteralNamed {
			if doc.prevKind == lineKindEmpty {
				break
			}
			node := &adocNode{
				kind: doc.kind,
			}
			line, c = doc.consumeLinesUntil(node,
				lineKindEmpty,
				[]int{
					nodeKindListOrderedItem,
					nodeKindListUnorderedItem,
				})
			listItem.addChild(node)
			continue
		}
		if doc.kind == nodeKindBlockListingDelimiter ||
			doc.kind == nodeKindBlockExample {
			break
		}
		if doc.kind == nodeKindSectionL1 ||
			doc.kind == nodeKindSectionL2 ||
			doc.kind == nodeKindSectionL3 ||
			doc.kind == nodeKindSectionL4 ||
			doc.kind == nodeKindSectionL5 {
			if doc.prevKind == lineKindEmpty {
				break
			}
		}

		listItem.raw.WriteString(strings.TrimSpace(line))
		listItem.raw.WriteByte('\n')
		line = ""
	}
	return line, c
}

func (doc *Document) parseBlock(parent *adocNode, term int) {
	node := &adocNode{
		kind: nodeKindUnknown,
	}
	var (
		spaces, line string
		c            rune
	)
	for {
		if len(line) == 0 {
			spaces, line, c = doc.line()
			if len(line) == 0 && c == 0 {
				return
			}
		}

		switch doc.kind {
		case lineKindEmpty:
			if node.kind != nodeKindUnknown {
				parent.addChild(node)
				node = &adocNode{}
			}
			continue
		case lineKindBlockComment:
			doc.parseIgnoreCommentBlock()
			line = ""
			continue
		case lineKindComment:
			line = ""
			continue
		case lineKindHorizontalRule:
			if node.kind != nodeKindUnknown {
				parent.addChild(node)
				node = &adocNode{}
			}
			node.kind = doc.kind
			parent.addChild(node)
			node = &adocNode{}
			line = ""
			continue
		case lineKindPageBreak:
			if doc.nodeCurrent.kind != nodeKindUnknown {
				parent.addChild(node)
				node = &adocNode{}
			}
			doc.nodeCurrent.kind = doc.kind
			parent.addChild(node)
			node = &adocNode{}
			line = ""
			continue

		case lineKindAttribute:
			if doc.parseAttribute(line, true) {
				line = ""
				continue
			}
			if node.kind == nodeKindUnknown {
				node.kind = nodeKindParagraph
				node.raw.WriteString(line)
				node.raw.WriteByte('\n')
				line, c = doc.consumeLinesUntil(
					node,
					lineKindEmpty,
					[]int{
						term,
						nodeKindBlockListingDelimiter,
						nodeKindBlockLiteralDelimiter,
						nodeKindBlockLiteralNamed,
						lineKindListContinue,
					})
				parent.addChild(node)
				node = &adocNode{}
			} else {
				node.raw.WriteString(line)
				line = ""
			}
			continue

		case lineKindStyle:
			styleName, styleKind := parseStyle(line)
			if styleKind != styleNone {
				node.style |= styleKind
				if isStyleAdmonition(styleKind) {
					node.setStyleAdmonition(styleName, styleKind)
				}
				line = ""
				continue
			}

		case lineKindStyleClass:
			node.parseStyleClass(line)
			line = ""
			continue

		case lineKindText, lineKindListContinue,
			nodeKindSectionL1, nodeKindSectionL2,
			nodeKindSectionL3, nodeKindSectionL4,
			nodeKindSectionL5:
			if node.kind == nodeKindUnknown {
				node.kind = nodeKindParagraph
				node.raw.WriteString(line)
				node.raw.WriteByte('\n')
				line, c = doc.consumeLinesUntil(
					node,
					lineKindEmpty,
					[]int{
						term,
						nodeKindBlockListingDelimiter,
						nodeKindBlockLiteralDelimiter,
						nodeKindBlockLiteralNamed,
						lineKindListContinue,
					})
				parent.addChild(node)
				node = &adocNode{}
			} else {
				node.raw.WriteString(line)
				line = ""
			}
			continue

		case lineKindBlockTitle:
			node.rawTitle = line[1:]
			line = ""
			continue

		case nodeKindLiteralParagraph:
			node.kind = doc.kind
			node.raw.WriteString(spaces + line)
			node.raw.WriteByte('\n')
			line, c = doc.consumeLinesUntil(
				node,
				lineKindEmpty,
				[]int{
					term,
					nodeKindBlockListingDelimiter,
					nodeKindBlockLiteralNamed,
					nodeKindBlockLiteralDelimiter,
				})
			parent.addChild(node)
			node = &adocNode{}
			continue

		case nodeKindBlockLiteralDelimiter:
			node.kind = doc.kind
			line, c = doc.consumeLinesUntil(node, doc.kind, nil)
			parent.addChild(node)
			node = &adocNode{}

		case nodeKindBlockLiteralNamed:
			node.kind = doc.kind
			line, c = doc.consumeLinesUntil(node, lineKindEmpty, nil)
			parent.addChild(node)
			node = &adocNode{}

		case nodeKindBlockListingDelimiter:
			node.kind = doc.kind
			line, c = doc.consumeLinesUntil(node, doc.kind, nil)
			parent.addChild(node)
			node = &adocNode{}

		case nodeKindListOrderedItem:
			line, c = doc.parseListOrdered(parent, "", line)
			parent.addChild(node)
			node = &adocNode{}
			continue

		case nodeKindListUnorderedItem:
			line, c = doc.parseListUnordered(parent, line)
			parent.addChild(node)
			node = &adocNode{}
			continue

		case nodeKindListDescriptionItem:
			line, c = doc.parseListDescription(parent, line)
			parent.addChild(node)
			node = &adocNode{}
			continue

		case nodeKindBlockImage:
			if node.kind != nodeKindUnknown {
				doc.terminateCurrentNode()
				parent.addChild(node)
				node = &adocNode{}
			}
			if node.parseImage(line) {
				node.kind = doc.kind
				parent.addChild(node)
				node = &adocNode{}
			} else {
				node.kind = nodeKindParagraph
				node.raw.WriteString("image::" + line)
				node.raw.WriteByte('\n')
				line, c = doc.consumeLinesUntil(
					node,
					lineKindEmpty,
					[]int{
						term,
						nodeKindBlockListingDelimiter,
						nodeKindBlockLiteralDelimiter,
						nodeKindBlockLiteralNamed,
						lineKindListContinue,
					})
				parent.addChild(node)
				node = &adocNode{}
			}
		case term:
			return
		}
		line = ""
	}
}

//
// parseListBlock parse block after list continuation "+" until we found
// empty line or non-list line.
//
func (doc *Document) parseListBlock() (node *adocNode, line string, c rune) {
	for {
		_, line, c = doc.line()
		if len(line) == 0 && c == 0 {
			break
		}

		if doc.kind == lineKindAdmonition {
			node = &adocNode{
				kind: lineKindAdmonition,
			}
			node.parseLineAdmonition(line)
			line, c = doc.consumeLinesUntil(
				node,
				lineKindEmpty,
				[]int{
					nodeKindBlockListingDelimiter,
					nodeKindBlockLiteralDelimiter,
					nodeKindBlockLiteralNamed,
					lineKindListContinue,
				})
			break
		}
		if doc.kind == lineKindBlockComment {
			doc.parseIgnoreCommentBlock()
			continue
		}
		if doc.kind == lineKindComment {
			continue
		}
		if doc.kind == lineKindEmpty {
			return node, line, c
		}
		if doc.kind == lineKindListContinue {
			continue
		}
		if doc.kind == nodeKindLiteralParagraph {
			node = &adocNode{
				kind: doc.kind,
			}
			node.raw.WriteString(strings.TrimLeft(line, " \t"))
			node.raw.WriteByte('\n')
			line, c = doc.consumeLinesUntil(
				node,
				lineKindEmpty,
				[]int{
					lineKindListContinue,
					nodeKindListOrderedItem,
					nodeKindListUnorderedItem,
				})
			break
		}
		if doc.kind == lineKindText {
			node = &adocNode{
				kind: nodeKindParagraph,
			}
			node.raw.WriteString(line)
			node.raw.WriteByte('\n')
			line, c = doc.consumeLinesUntil(node,
				lineKindEmpty,
				[]int{
					lineKindListContinue,
					nodeKindListOrderedItem,
					nodeKindListUnorderedItem,
					nodeKindListDescriptionItem,
				})
			break
		}
		if doc.kind == nodeKindBlockListingDelimiter {
			node = &adocNode{
				kind: doc.kind,
			}
			doc.consumeLinesUntil(node, doc.kind, nil)
			line = ""
			break
		}
		if doc.kind == nodeKindListOrderedItem {
			break
		}
		if doc.kind == nodeKindListUnorderedItem {
			break
		}
		if doc.kind == nodeKindListDescriptionItem {
			break
		}
	}
	return node, line, c
}

func (doc *Document) consumeLinesUntil(node *adocNode, term int, terms []int) (
	line string, c rune,
) {
	spaces := ""
	for {
		spaces, line, c = doc.line()
		if len(line) == 0 && c == 0 {
			break
		}
		if doc.kind == lineKindBlockComment {
			doc.parseIgnoreCommentBlock()
			continue
		}
		if doc.kind == lineKindComment {
			continue
		}
		if doc.kind == term {
			return "", 0
		}
		for _, t := range terms {
			if t == doc.kind {
				return line, c
			}
		}
		if node.kind == nodeKindParagraph && len(spaces) > 0 {
			node.raw.WriteByte(' ')
		}
		node.raw.WriteString(line)
		node.raw.WriteByte('\n')
	}
	return line, c
}

func (doc *Document) terminateCurrentNode() {
	if doc.nodeCurrent.kind != nodeKindUnknown {
		doc.nodeParent.addChild(doc.nodeCurrent)
	}
	doc.nodeCurrent = &adocNode{
		kind: nodeKindUnknown,
	}
}

func (doc *Document) line() (spaces, line string, c rune) {
	doc.prevKind = doc.kind
	doc.lineNum++
	line, c = doc.p.Line()
	doc.kind, spaces, line = whatKindOfLine(line)
	fmt.Printf("line %3d: kind %3d: %s\n", doc.lineNum, doc.kind, line)
	return spaces, line, c
}

//
// parseHeader document header consist of title and optional authors,
// revision, and zero or more attributes.
// The document attributes can be in any order, but the author and revision MUST
// be in order.
//
//	DOC_HEADER  = *(DOC_ATTRIBUTE / COMMENTS)
//	              "=" SP *ADOC_WORD LF
//	              (*DOC_ATTRIBUTE)
//	              DOC_AUTHORS LF
//	              (*DOC_ATTRIBUTE)
//	              DOC_REVISION LF
//	              (*DOC_ATTRIBUTE)
//
func (doc *Document) parseHeader() (spaces, line string, c rune) {
	const (
		stateTitle int = iota
		stateAuthor
		stateRevision
		stateEnd
	)
	state := stateTitle
	for {
		spaces, line, c = doc.line()
		if len(line) == 0 && c == 0 {
			break
		}
		if len(line) == 0 {
			// Only allow empty line if state is title.
			if state == stateTitle {
				continue
			}
			return spaces, line, c
		}

		if strings.HasPrefix(line, "////") {
			doc.parseIgnoreCommentBlock()
			continue
		}
		if strings.HasPrefix(line, "//") {
			continue
		}
		if line[0] == ':' {
			if doc.parseAttribute(line, false) {
				continue
			}
			if state != stateTitle {
				return spaces, line, c
			}
			// The line will be assumed either as author or
			// revision.
		}
		switch state {
		case stateTitle:
			if !isTitle(line) {
				return spaces, line, c
			}
			doc.header = &adocNode{
				kind: nodeKindDocHeader,
			}
			doc.header.raw.WriteString(strings.TrimSpace(line[2:]))
			doc.Title = doc.header.raw.String()
			state = stateAuthor
		case stateAuthor:
			doc.Author = line
			state = stateRevision
		case stateRevision:
			if !doc.parseHeaderRevision(line) {
				return spaces, line, c
			}
			state = stateEnd
		case stateEnd:
			return spaces, line, c
		}
	}
	return spaces, "", 0
}

//
// parseAttribute parse document attribute and return true if its valid.
//
//	DOC_ATTRIBUTE  = ":" DOC_ATTR_KEY ":" *STRING LF
//
//	DOC_ATTR_KEY   = ( "toc" / "sectanchors" / "sectlinks"
//	               /   "imagesdir" / "data-uri" / *META_KEY ) LF
//
//	META_KEY_CHAR  = (A..Z | a..z | 0..9 | '_')
//
//	META_KEY       = 1META_KEY_CHAR *(META_KEY_CHAR | '-')
//
func (doc *Document) parseAttribute(line string, strict bool) bool {
	key := make([]byte, 0, len(line))

	if !(ascii.IsAlnum(line[1]) || line[1] == '_') {
		return false
	}

	key = append(key, line[1])
	x := 2
	for ; x < len(line); x++ {
		if line[x] == ':' {
			break
		}
		if ascii.IsAlnum(line[x]) || line[x] == '_' || line[x] == '-' {
			key = append(key, line[x])
			continue
		}
		if strict {
			return false
		}
	}
	if x == len(line) {
		return false
	}

	doc.attributes[string(key)] = strings.TrimSpace(line[x+1:])

	return true
}

//
//	DOC_REVISION     = DOC_REV_VERSION [ "," DOC_REV_DATE ]
//
//	DOC_REV_VERSION  = "v" 1*DIGIT "." 1*DIGIT "." 1*DIGIT
//
//	DOC_REV_DATE     = 1*2DIGIT WSP 3*ALPHA WSP 4*DIGIT
//
func (doc *Document) parseHeaderRevision(line string) bool {
	if line[0] != 'v' {
		return false
	}

	idx := strings.IndexByte(line, ',')
	if idx > 0 {
		doc.RevNumber = line[1:idx]
		doc.RevDate = strings.TrimSpace(line[idx+1:])
		doc.RevSeparator = ","
	} else {
		doc.RevNumber = line[1:]
	}
	return true
}

func (doc *Document) parseIgnoreCommentBlock() {
	for {
		line, c := doc.p.Line()
		if strings.HasPrefix(line, "////") {
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
	tmpl, err := newHTMLTemplate()
	if err != nil {
		return err
	}

	err = tmpl.ExecuteTemplate(w, "BEGIN", doc)
	if err != nil {
		return err
	}

	err = tmpl.ExecuteTemplate(w, "BEGIN_HEADER", doc)
	if err != nil {
		return err
	}

	if doc.content.child != nil {
		err = doc.content.child.toHTML(tmpl, w)
		if err != nil {
			return err
		}
	}
	if doc.content.next != nil {
		err = doc.content.next.toHTML(tmpl, w)
		if err != nil {
			return err
		}
	}

	err = tmpl.ExecuteTemplate(w, "END", doc)

	return err
}
