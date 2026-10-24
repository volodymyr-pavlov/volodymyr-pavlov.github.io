package gen

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// errTexSyntax reports TeX that is malformed: a brace that is not closed, a command without its argument, or a script
// that has nothing to attach to.
var errTexSyntax = errors.New("tex: malformed formula")

// errTexUnsupported reports a command or a character that this package does not draw.
var errTexUnsupported = errors.New("tex: unsupported in a formula")

// texToMathML turns a small, useful part of TeX math into MathML, so that the blog shows formulas with no script and no
// font of its own. It returns the MathML element that draws src. With display set the formula stands on a line of its own,
// otherwise it sits in the text. The TeX source stays in the element as an annotation, so a reader can copy it.
//
// It returns an error wrapping [errTexSyntax] for malformed TeX and one wrapping [errTexUnsupported] for a command or
// character it does not draw.
func texToMathML(src string, display bool) (string, error) {
	p := &texParser{src: []rune(src)}

	nodes, err := p.row(texStopEnd)
	if err != nil {
		return "", fmt.Errorf("%w in %q", err, src)
	}

	var b strings.Builder
	b.WriteString("<math")
	if display {
		b.WriteString(` display="block"`)
	}
	b.WriteString("><semantics>")
	b.WriteString(texWrap(nodes))
	b.WriteString(`<annotation encoding="application/x-tex">`)
	b.WriteString(texEscape(strings.TrimSpace(src)))
	b.WriteString("</annotation></semantics></math>")

	return b.String(), nil
}

// texStop says what ends a row of nodes: the end of the text, a closing brace, or \right.
type texStop int

const (
	texStopEnd texStop = iota
	texStopBrace
	texStopRight
)

// texParser reads TeX from src, with pos at the next rune to read.
type texParser struct {
	src []rune
	pos int
}

// row reads nodes until the stop is reached, and consumes a closing brace but leaves \right for the caller.
func (p *texParser) row(until texStop) ([]string, error) {
	var nodes []string

	for {
		p.skipSpace()

		if p.pos >= len(p.src) {
			if until != texStopEnd {
				return nil, fmt.Errorf("%w: a group is not closed", errTexSyntax)
			}

			return nodes, nil
		}

		switch {
		case p.src[p.pos] == '}':
			if until != texStopBrace {
				return nil, fmt.Errorf("%w: a closing brace has no opening one", errTexSyntax)
			}

			p.pos++

			return nodes, nil
		case p.at(`\right`):
			if until != texStopRight {
				return nil, fmt.Errorf("%w: \\right has no \\left", errTexSyntax)
			}

			return nodes, nil
		}

		atom, err := p.atom()
		if err != nil {
			return nil, err
		}

		atom, err = p.scripts(atom)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, atom)
	}
}

// scripts reads the subscript and the superscript that follow base, in either order, and joins them to it.
func (p *texParser) scripts(base string) (string, error) {
	var sub, sup string

	for {
		p.skipSpace()

		if p.pos >= len(p.src) || (p.src[p.pos] != '_' && p.src[p.pos] != '^') {
			break
		}

		mark := p.src[p.pos]
		p.pos++

		arg, err := p.argument()
		if err != nil {
			return "", err
		}

		switch {
		case mark == '_' && sub == "":
			sub = arg
		case mark == '^' && sup == "":
			sup = arg
		default:
			return "", fmt.Errorf("%w: a double %c", errTexSyntax, mark)
		}
	}

	switch {
	case sub != "" && sup != "":
		return "<msubsup>" + base + sub + sup + "</msubsup>", nil
	case sub != "":
		return "<msub>" + base + sub + "</msub>", nil
	case sup != "":
		return "<msup>" + base + sup + "</msup>", nil
	}

	return base, nil
}

// argument reads what a command or a script applies to: a group in braces, or a single token.
func (p *texParser) argument() (string, error) {
	p.skipSpace()

	if p.pos >= len(p.src) {
		return "", fmt.Errorf("%w: an argument is missing at the end", errTexSyntax)
	}

	if p.src[p.pos] == '{' {
		p.pos++

		nodes, err := p.row(texStopBrace)
		if err != nil {
			return "", err
		}

		return texWrap(nodes), nil
	}

	if unicode.IsDigit(p.src[p.pos]) {
		digit := p.src[p.pos]
		p.pos++

		return "<mn>" + string(digit) + "</mn>", nil
	}

	return p.atom()
}

// atom reads one token: a letter, a number, an operator, a group or a command.
func (p *texParser) atom() (string, error) {
	c := p.src[p.pos]

	switch {
	case c == '{':
		return p.argument()
	case c == '_' || c == '^':
		return "<mrow></mrow>", nil
	case c == '\\':
		return p.command()
	case unicode.IsLetter(c):
		p.pos++

		return "<mi>" + texEscape(string(c)) + "</mi>", nil
	case unicode.IsDigit(c):
		return p.number(), nil
	}

	if op, ok := texOperators[c]; ok {
		p.pos++

		switch c {
		case '.':
			return `<mo lspace="0em">.</mo>`, nil
		case '/':
			return `<mo lspace="0em" rspace="0em">/</mo>`, nil
		}

		return "<mo>" + texEscape(op) + "</mo>", nil
	}

	return "", fmt.Errorf("%w: the character %q", errTexUnsupported, c)
}

// number reads digits with at most one decimal point that has a digit on both sides.
func (p *texParser) number() string {
	start := p.pos
	dot := false

	for p.pos < len(p.src) {
		c := p.src[p.pos]

		switch {
		case unicode.IsDigit(c):
			p.pos++
		case c == '.' && !dot && p.pos+1 < len(p.src) && unicode.IsDigit(p.src[p.pos+1]):
			dot = true
			p.pos++
		default:
			return "<mn>" + string(p.src[start:p.pos]) + "</mn>"
		}
	}

	return "<mn>" + string(p.src[start:p.pos]) + "</mn>"
}

// command reads a backslash command and its arguments.
func (p *texParser) command() (string, error) {
	p.pos++ // the backslash

	if p.pos >= len(p.src) {
		return "", fmt.Errorf("%w: a backslash ends the formula", errTexSyntax)
	}

	// A command is a run of letters, or one symbol such as \, or \;.
	start := p.pos
	if unicode.IsLetter(p.src[p.pos]) {
		for p.pos < len(p.src) && unicode.IsLetter(p.src[p.pos]) {
			p.pos++
		}
	} else {
		p.pos++
	}

	name := string(p.src[start:p.pos])

	switch name {
	case "frac":
		num, err := p.argument()
		if err != nil {
			return "", err
		}

		den, err := p.argument()
		if err != nil {
			return "", err
		}

		return "<mfrac>" + num + den + "</mfrac>", nil
	case "sqrt":
		arg, err := p.argument()
		if err != nil {
			return "", err
		}

		return "<msqrt>" + arg + "</msqrt>", nil
	case "text", "mathrm":
		return p.text(name)
	case "mathbf":
		return p.bold()
	case "left":
		return p.fenced()
	}

	if width, ok := texSpaces[name]; ok {
		return `<mspace width="` + width + `"/>`, nil
	}

	if sym, ok := texGreek[name]; ok {
		return "<mi>" + sym + "</mi>", nil
	}

	if sym, ok := texSymbols[name]; ok {
		return "<mo>" + sym + "</mo>", nil
	}

	if _, ok := texFunctions[name]; ok {
		return "<mi>" + name + "</mi>", nil
	}

	return "", fmt.Errorf("%w: the command \\%s", errTexUnsupported, name)
}

// text reads the braces of \text or \mathrm and draws what is in them as upright text. Spaces at the ends of \text
// are kept as non-breaking ones, because MathML drops the plain ones.
func (p *texParser) text(name string) (string, error) {
	p.skipSpace()

	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		return "", fmt.Errorf("%w: \\%s needs a group in braces", errTexSyntax, name)
	}

	p.pos++
	start := p.pos

	for p.pos < len(p.src) && p.src[p.pos] != '}' {
		if p.src[p.pos] == '{' || p.src[p.pos] == '\\' {
			return "", fmt.Errorf("%w: \\%s holds plain text only", errTexUnsupported, name)
		}

		p.pos++
	}

	if p.pos >= len(p.src) {
		return "", fmt.Errorf("%w: \\%s is not closed", errTexSyntax, name)
	}

	content := string(p.src[start:p.pos])
	p.pos++

	trimmed := strings.TrimLeft(content, " ")
	lead := strings.Repeat(" ", len(content)-len(trimmed))
	body := strings.TrimRight(trimmed, " ")
	trail := strings.Repeat(" ", len(trimmed)-len(body))

	return "<mtext>" + lead + texEscape(body) + trail + "</mtext>", nil
}

// bold reads the braces of \mathbf, which hold letters and digits, and draws them with the bold letters of Unicode,
// because browsers draw no other bold in MathML.
func (p *texParser) bold() (string, error) {
	p.skipSpace()

	if p.pos >= len(p.src) || p.src[p.pos] != '{' {
		return "", fmt.Errorf("%w: \\mathbf needs a group in braces", errTexSyntax)
	}

	p.pos++

	var b strings.Builder

	for ; p.pos < len(p.src) && p.src[p.pos] != '}'; p.pos++ {
		c := p.src[p.pos]

		switch {
		case c >= 'A' && c <= 'Z':
			c += 0x1D400 - 'A'
		case c >= 'a' && c <= 'z':
			c += 0x1D41A - 'a'
		case c >= '0' && c <= '9':
			c += 0x1D7CE - '0'
		default:
			return "", fmt.Errorf("%w: %q inside \\mathbf", errTexUnsupported, c)
		}

		b.WriteString("<mi>" + string(c) + "</mi>")
	}

	if p.pos >= len(p.src) {
		return "", fmt.Errorf("%w: \\mathbf is not closed", errTexSyntax)
	}

	p.pos++

	return b.String(), nil
}

// fenced reads \left X ... \right Y and draws it as a row whose delimiters stretch.
func (p *texParser) fenced() (string, error) {
	open, err := p.delimiter("left")
	if err != nil {
		return "", err
	}

	nodes, err := p.row(texStopRight)
	if err != nil {
		return "", err
	}

	p.pos += len(`\right`)

	closing, err := p.delimiter("right")
	if err != nil {
		return "", err
	}

	return "<mrow>" + open + strings.Join(nodes, "") + closing + "</mrow>", nil
}

// delimiter reads the character after \left or \right; a dot means none.
func (p *texParser) delimiter(after string) (string, error) {
	p.skipSpace()

	if p.pos >= len(p.src) {
		return "", fmt.Errorf("%w: \\%s has no delimiter", errTexSyntax, after)
	}

	c := p.src[p.pos]
	p.pos++

	if c == '.' {
		return "", nil
	}

	if !strings.ContainsRune("()[]|", c) {
		return "", fmt.Errorf("%w: the delimiter %q after \\%s", errTexUnsupported, c, after)
	}

	return `<mo fence="true" stretchy="true">` + string(c) + "</mo>", nil
}

// at reports whether the text at the position starts with s, and that s is not the start of a longer command.
func (p *texParser) at(s string) bool {
	rs := []rune(s)
	if p.pos+len(rs) > len(p.src) {
		return false
	}

	for i, r := range rs {
		if p.src[p.pos+i] != r {
			return false
		}
	}

	next := p.pos + len(rs)

	return next >= len(p.src) || !unicode.IsLetter(p.src[next])
}

func (p *texParser) skipSpace() {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
}

// texWrap returns one node for a row: the node itself if it is alone, else an mrow.
func texWrap(nodes []string) string {
	if len(nodes) == 1 {
		return nodes[0]
	}

	return "<mrow>" + strings.Join(nodes, "") + "</mrow>"
}

// texEscape writes the characters that XML reserves as entities.
func texEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}

// texOperators maps the characters that stand for themselves as operators or punctuation.
var texOperators = map[rune]string{
	'+':  "+",
	'-':  "−",
	'=':  "=",
	'<':  "<",
	'>':  ">",
	'(':  "(",
	')':  ")",
	'[':  "[",
	']':  "]",
	',':  ",",
	'.':  ".",
	';':  ";",
	':':  ":",
	'!':  "!",
	'/':  "/",
	'|':  "|",
	'\'': "′",
	'*':  "∗",
}

// texSpaces maps the spacing commands to their width.
var texSpaces = map[string]string{
	",":     "0.1667em",
	":":     "0.2222em",
	";":     "0.2778em",
	"quad":  "1em",
	"qquad": "2em",
}

// texGreek maps the names of Greek letters to the letters.
var texGreek = map[string]string{
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ", "epsilon": "ε", "varepsilon": "ε", "zeta": "ζ", "eta": "η",
	"theta": "θ", "lambda": "λ", "mu": "μ", "nu": "ν", "xi": "ξ", "pi": "π", "rho": "ρ", "sigma": "σ", "tau": "τ",
	"phi": "φ", "chi": "χ", "psi": "ψ", "omega": "ω",
	"Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ", "Pi": "Π", "Sigma": "Σ", "Phi": "Φ", "Psi": "Ψ",
	"Omega": "Ω",
	"infty": "∞", "partial": "∂", "nabla": "∇",
}

// texSymbols maps the names of operators and relations to their characters.
var texSymbols = map[string]string{
	"leftarrow": "←", "rightarrow": "→", "to": "→", "Rightarrow": "⇒", "Leftarrow": "⇐", "leftrightarrow": "↔",
	"cdot": "·", "times": "×", "div": "÷", "pm": "±", "mp": "∓",
	"le": "≤", "leq": "≤", "ge": "≥", "geq": "≥", "ne": "≠", "neq": "≠", "approx": "≈", "equiv": "≡", "sim": "∼",
	"in": "∈", "notin": "∉", "subset": "⊂", "cup": "∪", "cap": "∩",
	"sum": "∑", "prod": "∏", "int": "∫",
	"cdots": "⋯", "ldots": "…", "dots": "…",
}

// texFunctions are the names of operators that TeX sets upright.
var texFunctions = map[string]struct{}{
	"max": {}, "min": {}, "log": {}, "ln": {}, "exp": {}, "sin": {}, "cos": {}, "tan": {}, "arg": {}, "det": {},
	"sup": {}, "inf": {}, "lim": {},
}
