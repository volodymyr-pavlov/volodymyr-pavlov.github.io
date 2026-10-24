package gen

import (
	"bytes"
	"go/scanner"
	"go/token"
	"html/template"
	"io"
	"regexp"
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

var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM, mathExtension{}),
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

// htmlWriter writes HTML to w and keeps the first error, after which it writes nothing more.
type htmlWriter struct {
	w   io.Writer
	err error
}

// raw writes markup as it is.
func (h *htmlWriter) raw(markup string) {
	if h.err == nil {
		_, h.err = io.WriteString(h.w, markup)
	}
}

// escaped writes text with the characters that HTML reserves escaped.
func (h *htmlWriter) escaped(text string) {
	h.raw(string(util.EscapeHTML([]byte(text))))
}

// span writes text, escaped, in a span of the given class.
func (h *htmlWriter) span(class, text string) {
	h.raw(`<span class="` + class + `">`)
	h.escaped(text)
	h.raw(`</span>`)
}

// renderFencedCode writes a fenced code block, highlighting it when it is Go or YAML.
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

	h := &htmlWriter{w: w}
	h.raw("<pre><code")
	language := n.Language(source)
	if language != nil {
		h.raw(` class="language-`)
		h.escaped(string(language))
		h.raw(`"`)
	}
	h.raw(">")
	switch string(language) {
	case "go":
		highlightGo(h, code)
	case "yaml", "yml":
		highlightYAML(h, code)
	default:
		h.escaped(string(code))
	}
	h.raw("</code></pre>\n")
	return ast.WalkSkipChildren, h.err
}

// highlightGo writes escaped Go source with keywords, strings and comments wrapped in classed spans.
func highlightGo(h *htmlWriter, code []byte) {
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
		h.escaped(string(code[last:start]))
		h.span(class, lit)
		last = start + len(lit)
	}
	h.escaped(string(code[last:]))
}

// yamlKey matches the start of a line of YAML that holds a mapping key: the indent and list dashes, the key, and the
// colon that must be followed by a space or the end of the line.
var yamlKey = regexp.MustCompile(`^(\s*(?:-\s+)*)("[^"]*"|'[^']*'|[^\s#:"'-][^\s#:]*)(:)(?:\s|$)`)

// highlightYAML writes escaped YAML with mapping keys, quoted strings and comments wrapped in the classed spans that
// [highlightGo] uses. Keys are classed "keyword", quoted values "string" and comments "comment". Plain values are left
// uncoloured.
func highlightYAML(h *htmlWriter, code []byte) {
	for _, line := range strings.SplitAfter(string(code), "\n") {
		text, newline := strings.TrimSuffix(line, "\n"), ""
		if len(text) < len(line) {
			newline = "\n"
		}
		comment := ""
		if i := yamlCommentStart(text); i >= 0 {
			text, comment = text[:i], text[i:]
		}
		if m := yamlKey.FindStringSubmatchIndex(text); m != nil {
			h.escaped(text[:m[3]])
			h.span("keyword", text[m[4]:m[5]])
			h.escaped(text[m[5]:m[6]])
			text = text[m[6]:]
		}
		highlightYAMLValue(h, text)
		if comment != "" {
			h.span("comment", comment)
		}
		h.raw(newline)
	}
}

// yamlCommentStart returns the index of the "#" that starts a comment in line, or -1 if there is none. A "#" starts a
// comment only at the start of the line or after a space, and not inside quotes.
func yamlCommentStart(line string) int {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#' && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t'):
			return i
		}
	}
	return -1
}

// highlightYAMLValue writes text, wrapping each quoted string in a "string" span.
func highlightYAMLValue(h *htmlWriter, text string) {
	for {
		i := strings.IndexAny(text, `"'`)
		if i < 0 {
			break
		}
		end := strings.IndexByte(text[i+1:], text[i])
		if end < 0 {
			break
		}
		end += i + 2
		h.escaped(text[:i])
		h.span("string", text[i:end])
		text = text[end:]
	}
	h.escaped(text)
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

// headingIDs gives each heading the slug of its text as its id, in place of goldmark's counter-based ids,
// so links to a section survive edits elsewhere on the page.
type headingIDs struct{}

func (*headingIDs) Generate(value []byte, _ ast.NodeKind) []byte { return []byte(slug(string(value))) }

// Put would record an id written into the source, so Generate could avoid repeating it.
// The pages write none and the slugs are not deduplicated, so there is nothing to record.
func (*headingIDs) Put([]byte) {}

// slug lowercases s and joins its words with hyphens, dropping punctuation.
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

// markdownHTML renders Markdown, giving headings stable IDs and highlighting Go code.
func markdownHTML(source string) (template.HTML, error) {
	var buf strings.Builder
	ctx := parser.NewContext(parser.WithIDs(&headingIDs{}))
	if err := markdownRenderer.Convert([]byte(source), &buf, parser.WithContext(ctx)); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
