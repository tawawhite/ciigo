// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import "strings"

const (
	styleNone            int64 = iota
	styleSectionColophon       = 1 << (iota - 1)
	styleSectionAbstract
	styleSectionPreface
	styleSectionDedication
	styleSectionPartIntroduction
	styleSectionAppendix
	styleSectionGlossary
	styleSectionBibliography
	styleSectionIndex
	styleParagraphLead
	styleParagraphNormal
	styleNumberingArabic
	styleNumberingDecimal
	styleNumberingLoweralpha
	styleNumberingUpperalpha
	styleNumberingLowerroman
	styleNumberingUpperroman
	styleNumberingLowergreek
	styleDescriptionHorizontal
	styleDescriptionQandA
)

var adocStyles map[string]int64 = map[string]int64{
	"colophon":     styleSectionColophon,
	"abstract":     styleSectionAbstract,
	"preface":      styleSectionPreface,
	"dedication":   styleSectionDedication,
	"partintro":    styleSectionPartIntroduction,
	"appendix":     styleSectionAppendix,
	"glossary":     styleSectionGlossary,
	"bibliography": styleSectionBibliography,
	"index":        styleSectionIndex,
	".lead":        styleParagraphLead,
	".normal":      styleParagraphNormal,
	"arabic":       styleNumberingArabic,
	"decimal":      styleNumberingDecimal,
	"loweralpha":   styleNumberingLoweralpha,
	"upperalpha":   styleNumberingUpperalpha,
	"lowerroman":   styleNumberingLowerroman,
	"upperroman":   styleNumberingUpperroman,
	"lowergreek":   styleNumberingLowergreek,
	"horizontal":   styleDescriptionHorizontal,
	"qanda":        styleDescriptionQandA,
}

//
// parseStyle parse line that start with "[" and end with "]".
//
func parseStyle(line string) int64 {
	line = strings.ToLower(strings.Trim(line, "[]"))
	parts := strings.Split(line, ",")
	styleName := strings.Trim(parts[0], "\"")
	return adocStyles[styleName]
}
