// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"strings"

	"github.com/shuLhan/share/lib/ascii"
	"github.com/shuLhan/share/lib/parser"
)

func isLineDescriptionItem(line string) bool {
	x := strings.Index(line, ":: ")
	if x > 0 {
		return true
	}
	x = strings.Index(line, "::\t")
	if x > 0 {
		return true
	}
	x = strings.Index(line, "::")
	if x > 0 {
		return true
	}
	return false
}

func isTitle(line string) bool {
	if line[0] == '=' || line[0] == '#' {
		if line[1] == ' ' || line[1] == '\t' {
			return true
		}
	}
	return false
}

//
// parseBlockAttribute parse list of attributes in between "[" "]".
//
//	BLOCK_ATTRS = BLOCK_ATTR *("," BLOCK_ATTR)
//
//	BLOCK_ATTR  = ATTR_NAME ( "=" (DQUOTE) ATTR_VALUE (DQUOTE) )
//
//	ATTR_NAME   = WORD
//
//	ATTR_VALUE  = STRING
//
// The attribute may not have a value.
//
// If the attribute value contains space or comma, it must be wrapped with
// double quote.
// The double quote on value will be removed when stored on output.
//
// It will return nil if input is not a valid block attribute.
//
func parseBlockAttribute(in string) (out []string) {
	p := parser.New(in, `[,="]`)
	tok, c := p.Token()
	if c != '[' {
		return nil
	}
	if len(tok) > 0 {
		return nil
	}

	for c != 0 {
		tok, c = p.Token()
		tok = strings.TrimSpace(tok)
		if c == ',' || c == ']' {
			if len(tok) > 0 {
				out = append(out, tok)
			}
			if c == ']' {
				break
			}
			continue
		}
		if c != '=' {
			// Ignore invalid attribute.
			for c != ',' && c != 0 {
				tok, c = p.Token()
			}
			continue
		}
		key := tok
		tok, c = p.Token()
		tok = strings.TrimSpace(tok)
		if c == '"' {
			tok, c = p.ReadEnclosed('"', '"')
			tok = strings.TrimSpace(tok)
			out = append(out, key+"="+tok)
		} else {
			out = append(out, key+"="+tok)
		}

		for c != ',' && c != 0 {
			tok, c = p.Token()
		}
	}
	return out
}

//
// whatKindOfLine return the kind of line.
// It will return lineKindText if the line does not match with known syntax.
//
func whatKindOfLine(line string) (kind int, spaces, got string) {
	kind = lineKindText
	if len(line) == 0 {
		return lineKindEmpty, spaces, line
	}
	if strings.HasPrefix(line, "////") {
		// Check for comment block first, since we use HasPrefix to
		// check for single line comment.
		return lineKindBlockComment, spaces, line
	}
	if strings.HasPrefix(line, "//") {
		// Use HasPrefix to allow single line comment without space,
		// for example "//comment".
		return lineKindComment, spaces, line
	}
	if line == "'''" || line == "---" || line == "- - -" ||
		line == "***" || line == "* * *" {
		return lineKindHorizontalRule, spaces, line
	}
	if line == "<<<" {
		return lineKindPageBreak, spaces, line
	}
	if line == "--" {
		return nodeKindBlockOpen, spaces, line
	}
	if strings.HasPrefix(line, "image::") {
		line = strings.TrimRight(line[7:], " \t")
		return nodeKindBlockImage, spaces, line
	}
	if strings.HasPrefix(line, "video::") {
		line = strings.TrimRight(line[7:], " \t")
		return nodeKindBlockVideo, "", line
	}
	if strings.HasPrefix(line, "audio::") {
		line = strings.TrimRight(line[7:], " \t")
		return nodeKindBlockAudio, "", line
	}

	var (
		x        int
		r        rune
		hasSpace bool
	)
	for x, r = range line {
		if r == ' ' || r == '\t' {
			hasSpace = true
			continue
		}
		break
	}
	if hasSpace {
		spaces = line[:x]
		line = line[x:]

		// A line idented with space only allowed on list item,
		// otherwise it would be set as literal paragraph.

		if isLineDescriptionItem(line) {
			return nodeKindListDescriptionItem, spaces, line
		}

		if line[0] != '*' && line[0] != '.' {
			return nodeKindLiteralParagraph, spaces, line
		}
	}

	if line[0] == ':' {
		kind = lineKindAttribute
	} else if line[0] == '[' {
		newline := strings.TrimRight(line, " \t")
		if newline[len(newline)-1] == ']' {
			if line == "[literal]" {
				kind = nodeKindBlockLiteralNamed
			} else if line[1] == '.' {
				kind = lineKindStyleClass
			} else {
				kind = lineKindStyle
			}
			return kind, spaces, line
		}
	} else if line[0] == '=' {
		subs := strings.Fields(line)
		switch subs[0] {
		case "==":
			kind = nodeKindSectionL1
		case "===":
			kind = nodeKindSectionL2
		case "====":
			kind = nodeKindSectionL3
		case "=====":
			kind = nodeKindSectionL4
		case "======":
			kind = nodeKindSectionL5
		}
	} else if line[0] == '.' {
		if len(line) <= 1 {
			kind = lineKindText
		} else if line == "...." {
			kind = nodeKindBlockLiteralDelimiter
		} else if ascii.IsAlnum(line[1]) {
			kind = lineKindBlockTitle
		} else {
			x := 0
			for ; x < len(line); x++ {
				if line[x] == '.' {
					continue
				}
				if line[x] == ' ' || line[x] == '\t' {
					kind = nodeKindListOrderedItem
					return kind, spaces, line
				}
			}
		}
	} else if line[0] == '*' {
		if len(line) <= 1 {
			kind = lineKindText
		} else if line == "****" {
			kind = nodeKindBlockSidebar
		} else {
			x := 0
			for ; x < len(line); x++ {
				if line[x] == '*' {
					continue
				}
				if line[x] == ' ' || line[x] == '\t' {
					kind = nodeKindListUnorderedItem
					return kind, spaces, line
				}
			}
		}
	} else if line == "+" {
		kind = lineKindListContinue
	} else if line == "----" {
		kind = nodeKindBlockListingDelimiter
	} else if isLineDescriptionItem(line) {
		kind = nodeKindListDescriptionItem
	}
	return kind, spaces, line
}
