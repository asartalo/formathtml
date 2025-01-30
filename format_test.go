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
			input: `<div><pre><code>Foo bar
silk <span class="foo">bar</span></pre></code></div>`,
			expected: `<div>
  <pre><code>Foo bar
silk <span class="foo">bar</span></code></pre>
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
			name:  "paragraph text node shorter than wrap limit remain on the same line with its tags",
			input: `<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>`,
			expected: `<p>Lorem ipsum dolor sit amet, consectetur adipiscing elit.</p>
`,
		},
		{
			name:  "paragraph 'inline' elements remain on the same line if its content length is less than limit",
			input: `<div><p>Lorem ipsum <strong>dolor sit amet</strong>, consectetur adipiscing elit.</p></div>`,
			expected: `<div>
  <p>Lorem ipsum <strong>dolor sit amet</strong>, consectetur adipiscing elit.</p>
</div>
`,
		},
		{
			name:  "paragraph child elements are properly spaced",
			input: `<p>This <span> include </span> spaces please. This<i>is </i>weird. <em> Boo</em>.</p>`,
			expected: `<p>This <span> include </span> spaces please. This<i>is </i>weird. <em> Boo</em>.</p>
`,
		},
		{
			name:  "paragraph empty child element attributes are properly wrapped",
			input: `<p>See <b classs="red">image tag</b>. Something <img src="https://this.url.is/okay">What now? Some more text so this would wrap.</p>`,
			expected: `<p>
  See <b classs="red">image tag</b>. Something <img src="https://this.url.is/okay">What now? Some more
  text so this would wrap.
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
		{
			name:  "script tags with src attributes stay in one line",
			input: `<script src="https://example.com/script.js"></script>`,
			expected: `<script src="https://example.com/script.js"></script>
`,
		},
		{
			name:  "paragraph with text and inline br elements break on those lines",
			input: `<div><p>Lorem ipsum dolor sit amet,<br>consectetur adipiscing elit.<br>Cras in blandit odio, eget gravida eros.</p></div>`,
			expected: `<div>
  <p>
    Lorem ipsum dolor sit amet,<br>
    consectetur adipiscing elit.<br>
    Cras in blandit odio, eget gravida eros.
  </p>
</div>
`,
		},
		{
			name: "paragraph with inline br and line break formatting are properly indented",
			input: `<p>
    Lorem ipsum dolor sit amet,<br>
    consectetur adipiscing elit.<br>
    Cras in blandit odio, eget gravida eros.
  </p>
`,
			expected: `<p>
  Lorem ipsum dolor sit amet,<br>
  consectetur adipiscing elit.<br>
  Cras in blandit odio, eget gravida eros.
</p>
`,
		},
		{
			name:     "Escaped sequences are retained",
			input:    `<div>&lt;div&gt;Hello&lt;/div&gt;</div>` + "\n",
			expected: `<div>&lt;div&gt;Hello&lt;/div&gt;</div>` + "\n",
		},
		{
			name:     "Escaped sequences in paragraphs retained",
			input:    `<p>&lt;div&gt;Hello&lt;/div&gt;</p>` + "\n",
			expected: `<p>&lt;div&gt;Hello&lt;/div&gt;</p>` + "\n",
		},
		{
			name:     "Escaped sequences in pre tags are retained",
			input:    `<pre>&lt;div&gt;Hello&lt;/div&gt;</pre>` + "\n",
			expected: `<pre>&lt;div&gt;Hello&lt;/div&gt;</pre>` + "\n",
		},
		{
			name:  "Noscript code are not escaped",
			input: `<noscript><div>Hello</div></noscript>` + "\n",
			expected: `<noscript>
  <div>Hello</div>
</noscript>` + "\n",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r := strings.NewReader(test.input)
			w := new(strings.Builder)

			if err := Fragment(w, r); err != nil {
				t.Fatalf("failed to format: %v", err)
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
