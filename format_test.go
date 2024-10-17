package formathtml

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestFragmentFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "missing closing tags are inserted",
			input: `<li>`,
			expected: `<li>
</li>
`,
		},
		{
			name:  "html attribute escaping is normalized",
			input: `<ol> <li style="&amp;&#38;"> A </li> <li> B </li> </ol> `,
			expected: `<ol>
  <li style="&amp;&amp;">A</li>
  <li>B</li>
</ol>
`,
		},
		{
			name:  "bare ampersands are escaped",
			input: `<ol> <li style="&"> A </li> <li> B </li> </ol> `,
			expected: `<ol>
  <li style="&amp;">A</li>
  <li>B</li>
</ol>
`,
		},
		{
			name:  "html elements are indented",
			input: `<ol> <li class="name"> A </li> <li> B </li> </ol> `,
			expected: `<ol>
  <li class="name">A</li>
  <li>B</li>
</ol>
`,
		},
		{
			name:     "text fragments are supported",
			input:    `test 123`,
			expected: `test 123` + "\n",
		},
		{
			name:  "phrasing content element children are kept on the same line, including punctuation",
			input: `<ul><li><a href="http://example.com">Test</a>.</li></ul>`,
			expected: `<ul>
  <li>
    <a href="http://example.com">Test</a>.
  </li>
</ul>
`,
		},
		{
			name: "style content is indented consistently",
			input: `<style>
body {
  text-color: red;
}
</style>`,
			expected: `<style>
  body {
    text-color: red;
  }
</style>
`,
		},
		{
			name: "pre formats as is",
			input: `<div><pre>Foo bar
silk <span class="foo">bar</span></pre></div>`,
			expected: `<div>
  <pre>Foo bar
silk <span class="foo">bar</span></pre>
</div>
`,
		},
		{
			name:  "paragraph with long text wraps at about 100-character limit",
			input: `<div><p> Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras in blandit odio, eget gravida eros. In tincidunt, dolor nec blandit elementum, lacus metus semper lacus, id elementum augue ipsum in est. Vivamus tempor orci eget augue faucibus efficitur. </p></div>`,
			expected: `<div>
  <p>
    Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras in blandit odio, eget gravida eros. In
    tincidunt, dolor nec blandit elementum, lacus metus semper lacus, id elementum augue ipsum in est.
    Vivamus tempor orci eget augue faucibus efficitur.
  </p>
</div>
`,
		},
		{
			name:  "paragraph 'inline' elements are paragraph formatted",
			input: `<div><p>Lorem ipsum <strong>dolor sit amet</strong>, consectetur adipiscing elit.</p></div>`,
			expected: `<div>
  <p>
    Lorem ipsum <strong>dolor sit amet</strong>, consectetur adipiscing elit.
  </p>
</div>
`,
		},
		{
			name:  "paragraph child elements are properly spaced",
			input: `<p>This <span> include </span> spaces please. This<i>is </i>weird. <em> Boo</em>.</p>`,
			expected: `<p>
  This <span> include </span> spaces please. This<i>is </i>weird. <em> Boo</em>.
</p>
`,
		},
		{
			name:  "paragraph empty child element attributes are properly wrapped",
			input: `<p>See <b classs="red">image tag</b>. Something <img src="https://this.url.is/okay">What now?</p>`,
			expected: `<p>
  See <b classs="red">image tag</b>. Something <img src="https://this.url.is/okay">What now?
</p>
`,
		},
		{
			name:  "paragraph child element attributes are properly wrapped",
			input: `<p>See <b classs="red">image tag</b>. Something <img src="https://this.url.is/too-long-aaaaaaaaaaaaa-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-aaaaaaaaaaaaaaaaaaaaaaa-aaaaaaaaaaaaaaaaaaaaa-aaaaaaaaa-aaaaaaaaaaaaaaaaaaaa-aaa" >What now?</p>`,
			expected: `<p>
  See <b classs="red">image tag</b>. Something <img
  src="https://this.url.is/too-long-aaaaaaaaaaaaa-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-aaaaaaaaaaaaaaaaaaaaaaa-aaaaaaaaaaaaaaaaaaaaa-aaaaaaaaa-aaaaaaaaaaaaaaaaaaaa-aaa"
  >What now?
</p>
`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if strings.Contains(test.input, "Something") {
				Klog = true
			}
			r := strings.NewReader(test.input)
			w := new(strings.Builder)

			if err := Fragment(w, r); err != nil {
				t.Fatalf("failed to format: %v", err)
			}
			if strings.Contains(test.input, "Something") {
				Klog = false
			}
			assert.Equal(t, test.expected, w.String())
		})
	}
}

func TestDocumentFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "respects doctype declarations",
			input: `<!DOCTYPE html>
<html><head><link rel="stylesheet" href="/style.css"></head><body><h1>Hello</h1></body></html>
`,
			expected: `<!DOCTYPE html>
<html>
<head>
  <link rel="stylesheet" href="/style.css">
</head>
<body>
  <h1>Hello</h1>
</body>
</html>
`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			r := strings.NewReader(test.input)
			w := new(strings.Builder)
			if err := Document(w, r); err != nil {
				t.Fatalf("failed to format: %v", err)
			}
			if diff := cmp.Diff(test.expected, w.String()); diff != "" {
				t.Error(diff)
			}
		})
	}
}
