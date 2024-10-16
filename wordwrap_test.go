// The following code was taken from https://github.com/mitchellh/go-wordwrap
// by Mitchell Hashimoto with customizations for this library. The following
// license text was copied herein.

// The MIT License (MIT)
//
// Copyright (c) 2014 Mitchell Hashimoto
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package formathtml

import (
	"bytes"
	"strings"
	"testing"
)

func TestWrapString(t *testing.T) {
	cases := []struct {
		Input, Output string
		Limit         uint
		StartsAt      uint
		Indentation   string
	}{
		// A simple word passes through.
		{
			"foo",
			"foo",
			4,
			0,
			"",
		},
		// A single word that is too long passes through.
		// We do not break words.
		{
			"foobarbaz",
			"foobarbaz",
			4,
			0,
			"",
		},
		// Lines are broken at whitespace.
		{
			"foo bar baz",
			"foo\nbar\nbaz",
			4,
			0,
			"",
		},
		// Lines are broken at whitespace, even if words
		// are too long. We do not break words.
		{
			"foo bars bazzes",
			"foo\nbars\nbazzes",
			4,
			0,
			"",
		},
		// A word that would run beyond the width is wrapped.
		{
			"fo sop",
			"fo\nsop",
			4,
			0,
			"",
		},
		// Do not break on non-breaking space.
		{
			"foo bar\u00A0baz",
			"foo\nbar\u00A0baz",
			10,
			0,
			"",
		},
		// Whitespace that trails a line and fits the width
		// passes through, as does whitespace prefixing an
		// explicit line break. A tab counts as one character.
		{
			"foo\nb\t r\n baz",
			"foo\nb\t r\n baz",
			4,
			0,
			"",
		},
		// Trailing whitespace is removed if it doesn't fit the width.
		// Runs of whitespace on which a line is broken are removed.
		{
			"foo    \nb   ar   ",
			"foo\nb\nar",
			4,
			0,
			"",
		},
		// An explicit line break at the end of the input is preserved.
		{
			"foo bar baz\n",
			"foo\nbar\nbaz\n",
			4,
			0,
			"",
		},
		// Explicit break are always preserved.
		{
			"\nfoo bar\n\n\nbaz\n",
			"\nfoo\nbar\n\n\nbaz\n",
			4,
			0,
			"",
		},
		// Complete example:
		{
			" This is a list: \n\n\t* foo\n\t* bar\n\n\n\t* baz  \nBAM    ",
			" This\nis a\nlist: \n\n\t* foo\n\t* bar\n\n\n\t* baz\nBAM",
			6,
			0,
			"",
		},
		// Multi-byte characters
		{
			strings.Repeat("\u2584 ", 4),
			"\u2584 \u2584" + "\n" +
				strings.Repeat("\u2584 ", 2),
			4,
			0,
			"",
		},
		// Example with start (first line indentation)
		{
			"aa bb cc dd ee ff gg",
			"aa\nbb cc\ndd ee\nff gg",
			5,
			2,
			"",
		},
		// Example with indentation
		{
			"aa bb cc dd ee ff gg",
			"  aa bb\n  cc dd\n  ee ff\n  gg",
			5,
			0,
			"  ",
		},
		// Start skips adding indentation at first line because it is assumed that
		// indentation has already happened on it.
		{
			"aa bb cc dd ee ff gg",
			"aa\nxxbb cc\nxxdd ee\nxxff gg",
			5,
			2,
			"xx",
		},
	}

	for i, tc := range cases {
		buf := bytes.NewBuffer([]byte{})
		WrapString(tc.Input, buf, WrapOptions{
			Limit:       tc.Limit,
			StartsAt:    tc.StartsAt,
			Indentation: tc.Indentation,
		})
		actual := buf.String()
		if actual != tc.Output {
			t.Fatalf("Case %d Input:\n\n`%s`\n\nExpected Output:\n\n`%s`\n\nActual Output:\n\n`%s`", i, tc.Input, tc.Output, actual)
		}
	}
}
