package main

import (
	"bytes"
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

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
