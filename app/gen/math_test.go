package gen

import (
	"errors"
	"strings"
	"testing"
)

// TestMathInText proves that a formula between dollar signs becomes MathML in the paragraph, that the backslashes and
// underscores in it reach the converter as they were written, and that the text around it is left alone.
func TestMathInText(t *testing.T) {
	got, err := markdownHTML(`The weight $w_1 x_1 + w_2 x_2 \, b$ and _emphasis_ stay apart.`)
	if err != nil {
		t.Fatal(err)
	}

	html := string(got)
	for _, want := range []string{
		"<math><semantics>",
		"<msub><mi>w</mi><mn>1</mn></msub>",
		`<mspace width="0.1667em"/>`,
		"<em>emphasis</em> stay apart.",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("the page lacks %q:\n%s", want, html)
		}
	}

	if strings.Contains(html, "<em>1 x") || strings.Contains(html, "$") {
		t.Errorf("a dollar sign or an emphasis got into the formula:\n%s", html)
	}
}

// TestMathOverALineBreak proves that a formula in the text may be wrapped over two lines of the Markdown, as an editor
// wraps it at a space, and still be drawn whole, while two prices on two lines are still text.
func TestMathOverALineBreak(t *testing.T) {
	got, err := markdownHTML("The weights $\\mathbf{w} = (w_1, w_2,\nb)$ and more.")
	if err != nil {
		t.Fatal(err)
	}

	html := string(got)
	if !strings.Contains(html, "<math><semantics>") || !strings.Contains(html, "<msub><mi>w</mi><mn>2</mn></msub>") ||
		strings.Contains(html, "$") {
		t.Errorf("the wrapped formula is not drawn whole:\n%s", html)
	}

	got, err = markdownHTML("It costs $5 and\nsaves $3 in all.")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(got), "<math") || !strings.Contains(string(got), "$5 and") {
		t.Errorf("two prices became a formula:\n%s", got)
	}
}

// TestMathOnItsOwnLines proves that $$ ... $$ makes a block formula, even over several lines, and that its commands
// reach the converter whole.
func TestMathOnItsOwnLines(t *testing.T) {
	got, err := markdownHTML("Before.\n\n$$\ny = 1 \\text{ if } x\n$$\n\nAfter.")
	if err != nil {
		t.Fatal(err)
	}

	html := string(got)
	if !strings.Contains(html, `<math display="block"><semantics>`) {
		t.Errorf("no block formula:\n%s", html)
	}

	if !strings.Contains(html, "<mtext>\u00a0if\u00a0</mtext>") || strings.Contains(html, "$$") {
		t.Errorf("the formula is not drawn:\n%s", html)
	}
}

// TestDollarsThatAreNotMath proves that prices, a lone dollar sign, an unclosed pair and dollar signs in code are text.
func TestDollarsThatAreNotMath(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "two prices",
			source: "It costs $5 and $10 in all.",
			want:   "It costs $5 and $10 in all.",
		},
		{
			name:   "a lone dollar sign",
			source: "Send me $ please.",
			want:   "Send me $ please.",
		},
		{
			name:   "an unclosed pair",
			source: "Start $$ and never end.",
			want:   "Start $$ and never end.",
		},
		{
			name:   "an escaped dollar sign",
			source: `Pay \$x\$ now.`,
			want:   "Pay $x$ now.",
		},
		{
			name:   "a code span",
			source: "Run `echo $HOME and $PATH` here.",
			want:   "<code>echo $HOME and $PATH</code>",
		},
		{
			name:   "a code block",
			source: "```sh\necho $x$ and $$y$$\n```",
			want:   "echo $x$ and $$y$$",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := markdownHTML(tt.source)
			if err != nil {
				t.Fatal(err)
			}

			if strings.Contains(string(got), "<math") || !strings.Contains(string(got), tt.want) {
				t.Errorf("markdownHTML(%q) = %s, want text with %q and no formula", tt.source, got, tt.want)
			}
		})
	}
}

// TestMathThatCannotBeDrawn proves that a formula with a command the converter does not know stops the rendering with
// its error, so a page is never published with a formula half drawn.
func TestMathThatCannotBeDrawn(t *testing.T) {
	_, err := markdownHTML(`A formula $\unknown{x}$ here.`)
	if !errors.Is(err, errTexUnsupported) {
		t.Errorf("markdownHTML returned %v, want errTexUnsupported", err)
	}
}
