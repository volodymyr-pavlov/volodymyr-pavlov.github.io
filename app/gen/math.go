package gen

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Priorities of the math parser and renderer. The parser runs before goldmark's own inline parsers, which start at 100,
// and the renderer before the default HTML renderer, registered at 1000.
const (
	mathParserPriority   = 50
	mathRendererPriority = 50
)

// kindMath is the kind of a [mathNode].
var kindMath = ast.NewNodeKind("Math")

// mathNode is a formula written in TeX between dollar signs. It has no children: its source is kept as it was
// written, with none of the Markdown escapes applied, so a backslash stays a backslash.
type mathNode struct {
	ast.BaseInline
	tex     string
	display bool
}

func (*mathNode) Kind() ast.NodeKind { return kindMath }

func (n *mathNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, map[string]string{"tex": n.tex}, nil)
}

// mathExtension makes goldmark read $...$ as a formula in the text and $$...$$ as a formula on a line of its own, and
// draw both as MathML. A dollar sign that starts no formula stays text, so "$5 and $10" is a price.
type mathExtension struct{}

func (mathExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(&mathParser{}, mathParserPriority)))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(&mathRenderer{}, mathRendererPriority)))
}

// mathParser finds formulas. A formula in the text opens with a dollar sign that has no space after it and closes with
// one that has no space before it and no digit after it, on the same line. A formula on its own line is held by two
// dollar signs on each side and may run over several lines.
type mathParser struct{}

func (*mathParser) Trigger() []byte { return []byte{'$'} }

func (*mathParser) Parse(_ ast.Node, block text.Reader, _ parser.Context) ast.Node {
	line, _ := block.PeekLine()

	if bytes.HasPrefix(line, []byte("$$")) {
		return parseDisplay(block)
	}

	return parseInline(block, line)
}

// parseInline reads $...$ from line, which starts at the opening dollar sign, and returns nil if it holds no formula.
func parseInline(block text.Reader, line []byte) ast.Node {
	if len(line) < 3 || isSpace(line[1]) {
		return nil
	}

	for i := 2; i < len(line); i++ {
		if line[i] != '$' || line[i-1] == '\\' || isSpace(line[i-1]) {
			continue
		}

		if i+1 < len(line) && line[i+1] >= '0' && line[i+1] <= '9' {
			continue
		}

		block.Advance(i + 1)

		return &mathNode{tex: string(line[1:i])}
	}

	return nil
}

// parseDisplay reads $$...$$ from the position of the block, over as many lines as it takes, and returns nil, with the
// reader where it was, if the closing pair is missing.
func parseDisplay(block text.Reader) ast.Node {
	startLine, startPos := block.Position()
	block.Advance(2)

	var body bytes.Buffer

	for {
		line, _ := block.PeekLine()
		if line == nil {
			block.SetPosition(startLine, startPos)

			return nil
		}

		if end := bytes.Index(line, []byte("$$")); end >= 0 {
			body.Write(line[:end])
			block.Advance(end + 2)

			if len(bytes.TrimSpace(body.Bytes())) == 0 {
				block.SetPosition(startLine, startPos)

				return nil
			}

			return &mathNode{
				tex:     body.String(),
				display: true,
			}
		}

		body.Write(line)
		block.AdvanceLine()
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// mathRenderer writes a [mathNode] as MathML. A formula that cannot be drawn stops the build with the error of the converter, so that nothing is published half drawn.
type mathRenderer struct{}

func (*mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMath, renderMath)
}

func renderMath(w util.BufWriter, _ []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	n := node.(*mathNode)

	markup, err := texToMathML(n.tex, n.display)
	if err != nil {
		return ast.WalkStop, err
	}

	_, err = w.WriteString(markup)

	return ast.WalkContinue, err
}
