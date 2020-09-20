// Copyright 2020, Shulhan <ms@kilabit.info>. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package ciigo

import (
	"html/template"
)

func newHTMLTemplate() (tmpl *template.Template, err error) {
	tmpl = template.New("HTML")
	tmpl, err = tmpl.Parse(`
{{- define "BEGIN" -}}
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="generator" content="ciigo">
	{{- if .Author}}
<meta name="author" content="{{.Author}}">
	{{- end -}}
	{{- if .Title}}
<title>{{.Title}}</title>
	{{- end}}
<style>

</style>
</head>
<body class="article">
{{- end -}}

{{- define "END"}}
</div>
<div id="footer">
<div id="footer-text">
	{{- if .RevNumber}}
Version {{.RevNumber}}<br>
	{{- end}}
Last updated {{.LastUpdated}}
</div>
</div>
</body>
</html>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_HEADER"}}
<div id="header">
	{{- if .Title}}
<h1>{{.Title}}</h1>
	{{- end}}
<div class="details">
	{{- if .Author}}
<span id="author" class="author">{{.Author}}</span><br>
	{{- end}}
	{{- if .RevNumber}}
<span id="revnumber">version {{.RevNumber}}{{.RevSeparator}}</span>
	{{- end}}
	{{- if .RevDate}}
<span id="revdate">{{.RevDate}}</span>
	{{- end}}
</div>
</div>
<div id="content">
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_PREAMBLE"}}
<div id="preamble">
<div class="sectionbody">
{{- end}}

{{- define "END_PREAMBLE"}}
</div>
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_SECTION_L1"}}
<div class="sect1">
<h2 id="{{.GenerateID .Content}}">{{- .Content -}}</h2>
<div class="sectionbody">
{{- end}}
{{- define "END_SECTION_L1"}}
</div>
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_SECTION_L2"}}
<div class="sect2">
<h3 id="{{.GenerateID .Content}}">{{- .Content -}}</h3>
{{- end}}
{{- define "END_SECTION"}}
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_SECTION_L3"}}
<div class="sect3">
<h4 id="{{.GenerateID .Content}}">{{- .Content -}}</h4>
<div class="sectionbody">
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_SECTION_L4"}}
<div class="sect4">
<h5 id="{{.GenerateID .Content}}">{{- .Content -}}</h5>
<div class="sectionbody">
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_SECTION_L5"}}
<div class="sect5">
<h6 id="{{.GenerateID .Content}}">{{- .Content -}}</h6>
<div class="sectionbody">
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BLOCK_TITLE"}}
	{{- with $title := .Title}}
<div class="title">{{$title}}</div>
	{{- end}}
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "PARAGRAPH"}}
<div class="paragraph">
{{- template "BLOCK_TITLE" .}}
<p>{{- .Content -}}</p>
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BLOCK_LITERAL"}}
<div class="literalblock">
<div class="content">
<pre>{{.Content -}}</pre>
</div>
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BLOCK_LISTING"}}
<div class="listingblock">
<div class="content">
<pre>{{.Content -}}</pre>
</div>
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_LIST_ORDERED"}}
{{- $class := .GetListOrderedClass}}
{{- $type := .GetListOrderedType}}
<div class="olist {{$class}}">
{{- template "BLOCK_TITLE" .}}
<ol class="{{$class}}"{{- if $type}} type="{{$type}}"{{end}}>
{{- end}}
{{- define "END_LIST_ORDERED"}}
</ol>
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_LIST_UNORDERED"}}
<div class="ulist">
{{- template "BLOCK_TITLE" .}}
<ul>
{{- end}}
{{define "END_LIST_UNORDERED"}}
</ul>
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_LIST_DESCRIPTION"}}
	{{- if .IsStyleQandA}}
<div class="qlist qanda">
{{- template "BLOCK_TITLE" .}}
<ol>
	{{- else if .IsStyleHorizontal}}
<div class="hdlist">
{{- template "BLOCK_TITLE" .}}
<table>
	{{- else}}
<div class="dlist">
{{- template "BLOCK_TITLE" .}}
<dl>
	{{- end}}
{{- end}}
{{- define "END_LIST_DESCRIPTION"}}
	{{- if .IsStyleQandA}}
</ol>
	{{- else if .IsStyleHorizontal}}
</table>
	{{- else}}
</dl>
	{{- end}}
</div>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_LIST_ITEM"}}
<li>
<p>{{- .Content -}}</p>
{{- end}}
{{- define "END_LIST_ITEM"}}
</li>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "BEGIN_LIST_DESCRIPTION_ITEM"}}
	{{- if .IsStyleQandA}}
<li>
<p><em>{{.Terminology}}</em></p>
	{{- else if .IsStyleHorizontal}}
<tr>
<td class="hdlist1">
{{.Terminology}}
</td>
<td class="hdlist2">
	{{- else}}
<dt class="hdlist1">{{- .Terminology -}}</dt>
<dd>
	{{- end}}
	{{- with $content := .Content}}
<p>{{- $content -}}</p>
	{{- end}}
{{- end}}
{{- define "END_LIST_DESCRIPTION_ITEM"}}
	{{- if .IsStyleQandA}}
</li>
	{{- else if .IsStyleHorizontal}}
</td>
</tr>
	{{- else}}
</dd>
	{{- end}}
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{- define "HORIZONTAL_RULE"}}
<hr>
{{- end}}
{{/*----------------------------------------------------------------------*/}}
{{/*----------------------------------------------------------------------*/}}
`)
	return tmpl, err
}
