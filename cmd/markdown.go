package main

import (
	"bytes"
	"go/scanner"
	"go/token"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(
		html.WithUnsafe(),
		renderer.WithNodeRenderers(util.Prioritized(&codeRenderer{}, codeRendererPriority)),
	),
)

// codeRendererPriority puts codeRenderer ahead of goldmark's default HTML renderer, registered at 1000.
const codeRendererPriority = 100

// codeRenderer replaces goldmark's fenced code block output so Go code can be highlighted.
type codeRenderer struct{}

func (*codeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, renderFencedCode)
}

func renderFencedCode(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.FencedCodeBlock)
	var code []byte
	for i := 0; i < n.Lines().Len(); i++ {
		line := n.Lines().At(i)
		code = append(code, line.Value(source)...)
	}

	var buf bytes.Buffer
	buf.WriteString("<pre><code")
	language := n.Language(source)
	if language != nil {
		buf.WriteString(` class="language-`)
		buf.Write(util.EscapeHTML(language))
		buf.WriteString(`"`)
	}
	buf.WriteString(">")
	if string(language) == "go" {
		highlightGo(&buf, code)
	} else {
		buf.Write(util.EscapeHTML(code))
	}
	buf.WriteString("</code></pre>\n")

	_, err := w.Write(buf.Bytes())
	return ast.WalkSkipChildren, err
}

// highlightGo writes escaped Go source with keywords, strings and comments wrapped in classed spans.
func highlightGo(buf *bytes.Buffer, code []byte) {
	// The scanner drops carriage returns from comment and raw string literals, which would shift offsets.
	code = bytes.ReplaceAll(code, []byte("\r\n"), []byte("\n"))
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(code))
	var s scanner.Scanner
	s.Init(file, code, nil, scanner.ScanComments)

	last := 0
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		class := goClass(tok)
		if class == "" {
			continue
		}
		start := file.Offset(pos)
		buf.Write(util.EscapeHTML(code[last:start]))
		buf.WriteString(`<span class="` + class + `">`)
		buf.Write(util.EscapeHTML([]byte(lit)))
		buf.WriteString(`</span>`)
		last = start + len(lit)
	}
	buf.Write(util.EscapeHTML(code[last:]))
}

// goClass returns the CSS class for a Go token, or "" when the token is left uncoloured.
func goClass(tok token.Token) string {
	switch {
	case tok.IsKeyword():
		return "keyword"
	case tok == token.STRING || tok == token.CHAR:
		return "string"
	case tok == token.COMMENT:
		return "comment"
	}
	return ""
}

type headingIDs struct{}

func (*headingIDs) Generate(value []byte, _ ast.NodeKind) []byte { return []byte(slug(string(value))) }

func (*headingIDs) Put([]byte) {}

func slug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), "-")
}

func render(source string) (string, error) {
	var buf bytes.Buffer
	ctx := parser.NewContext(parser.WithIDs(&headingIDs{}))
	if err := md.Convert([]byte(source), &buf, parser.WithContext(ctx)); err != nil {
		return "", err
	}
	return buf.String(), nil
}
