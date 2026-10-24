package gen

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

// TestTexToMathML proves the MathML of formulas that cover each construct: letters and numbers, scripts in both orders,
// fractions, roots, Greek letters and symbols, spaces, text, bold letters and fences, and that the TeX stays in the
// element as an annotation.
func TestTexToMathML(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		display bool
		want    string
	}{
		{
			name: "letters, numbers and operators",
			src:  "w_1 x + 0.5 = b",
			want: "<mrow>" + "<msub><mi>w</mi><mn>1</mn></msub><mi>x</mi><mo>+</mo><mn>0.5</mn><mo>=</mo><mi>b</mi>" + "</mrow>",
		},
		{
			name: "a full stop takes no space on its left",
			src:  "b - y.",
			want: "<mrow><mi>b</mi><mo>\u2212</mo><mi>y</mi>" + `<mo lspace="0em">.</mo></mrow>`,
		},
		{
			name: "a slash takes no space",
			src:  "R/g",
			want: `<mrow><mi>R</mi><mo lspace="0em" rspace="0em">/</mo><mi>g</mi></mrow>`,
		},
		{
			name: "a minus is the minus sign",
			src:  "t - y",
			want: "<mrow>" + "<mi>t</mi><mo>−</mo><mi>y</mi>" + "</mrow>",
		},
		{
			name: "both scripts, in either order",
			src:  "x_i^2 y^2_i",
			want: "<mrow><msubsup><mi>x</mi><mi>i</mi><mn>2</mn></msubsup>" +
				"<msubsup><mi>y</mi><mi>i</mi><mn>2</mn></msubsup></mrow>",
		},
		{
			name: "a script in braces and a digit after a script",
			src:  "x_{ab}1 z_12",
			want: "<mrow>" + "<msub><mi>x</mi><mrow><mi>a</mi><mi>b</mi></mrow></msub><mn>1</mn>" +
				"<msub><mi>z</mi><mn>1</mn></msub><mn>2</mn>" + "</mrow>",
		},
		{
			name: "a fraction and a root",
			src:  `\frac{a}{b} \sqrt{x}`,
			want: "<mrow>" + "<mfrac><mi>a</mi><mi>b</mi></mfrac><msqrt><mi>x</mi></msqrt>" + "</mrow>",
		},
		{
			name: "Greek letters, an arrow and a function",
			src:  `w \leftarrow \eta \gamma \max`,
			want: "<mrow>" + "<mi>w</mi><mo>←</mo><mi>η</mi><mi>γ</mi><mi>max</mi>" + "</mrow>",
		},
		{
			name: "spaces",
			src:  `a \, b \qquad c`,
			want: "<mrow>" + `<mi>a</mi><mspace width="0.1667em"/><mi>b</mi><mspace width="2em"/><mi>c</mi>` + "</mrow>",
		},
		{
			name: "text keeps the spaces at its ends",
			src:  `y = 1 \text{ if } x`,
			want: "<mrow>" + "<mi>y</mi><mo>=</mo><mn>1</mn><mtext> if </mtext><mi>x</mi>" + "</mrow>",
		},
		{
			name: "text is escaped",
			src:  `\text{a < b & c}`,
			want: "<mtext>a &lt; b &amp; c</mtext>",
		},
		{
			name: "bold letters are the bold letters of Unicode",
			src:  `\mathbf{w}`,
			want: "<mi>\U0001D430</mi>",
		},
		{
			name: "a fence stretches",
			src:  `\left( a + b \right)`,
			want: `<mrow><mo fence="true" stretchy="true">(</mo><mi>a</mi><mo>+</mo><mi>b</mi>` +
				`<mo fence="true" stretchy="true">)</mo></mrow>`,
		},
		{
			name: "a sum with limits",
			src:  `\sum_{i=1}^{n} x_i`,
			want: "<mrow>" + "<msubsup><mo>∑</mo><mrow><mi>i</mi><mo>=</mo><mn>1</mn></mrow><mi>n</mi></msubsup>" +
				"<msub><mi>x</mi><mi>i</mi></msub>" + "</mrow>",
		},
		{
			name:    "a display formula",
			src:     "x",
			display: true,
			want:    "<mi>x</mi>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := texToMathML(tt.src, tt.display)
			if err != nil {
				t.Fatal(err)
			}

			open := "<math><semantics>"
			if tt.display {
				open = `<math display="block"><semantics>`
			}

			want := open + tt.want + `<annotation encoding="application/x-tex">` + texEscape(tt.src) +
				"</annotation></semantics></math>"
			if got != want {
				t.Errorf("texToMathML(%q)\n got  %s\n want %s", tt.src, got, want)
			}
		})
	}
}

// TestTexRefusesWhatItCannotDraw proves that malformed TeX gives errTexSyntax, that a command or character it does not
// draw gives errTexUnsupported, and that the error names the formula.
func TestTexRefusesWhatItCannotDraw(t *testing.T) {
	tests := []struct {
		src  string
		want error
	}{
		{`{a`, errTexSyntax},
		{`a}`, errTexSyntax},
		{`x_`, errTexSyntax},
		{`x_1_2`, errTexSyntax},
		{`\frac{a}`, errTexSyntax},
		{`\text{a`, errTexSyntax},
		{`\right)`, errTexSyntax},
		{`\left( a`, errTexSyntax},
		{`a \`, errTexSyntax},
		{`\begin{cases} a \end{cases}`, errTexUnsupported},
		{`\unknown`, errTexUnsupported},
		{`a \\ b`, errTexUnsupported},
		{`a & b`, errTexUnsupported},
		{`\text{a \alpha}`, errTexUnsupported},
		{`\mathbf{\alpha}`, errTexUnsupported},
		{`\left< a \right>`, errTexUnsupported},
	}
	for _, tt := range tests {
		_, err := texToMathML(tt.src, false)
		if !errors.Is(err, tt.want) {
			t.Errorf("texToMathML(%q) returned %v, want %v", tt.src, err, tt.want)

			continue
		}

		if !strings.Contains(err.Error(), strconv.Quote(tt.src)) {
			t.Errorf("the error for %q does not name the formula: %v", tt.src, err)
		}
	}
}
