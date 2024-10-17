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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestCaseData struct {
	Input, Output   string
	Limit, StartsAt uint
	Indentation     string
}

var cases = []TestCaseData{
	{
		// A simple word passes through.
		"foo",
		"foo",
		4,
		0,
		"",
	},
	{
		// A single word that is too long passes through.
		// We do not break words.
		"foobarbaz",
		"foobarbaz",
		4,
		0,
		"",
	},
	{
		// Lines are broken at whitespace.
		"foo bar baz",
		"foo\nbar\nbaz",
		4,
		0,
		"",
	},
	{
		// Words fill a line
		"foo bar baz",
		"foo bar\nbaz",
		7,
		0,
		"",
	},
	{
		// Column count do not include trailing spaces
		"foo bar baz",
		"foo bar\nbaz",
		8,
		0,
		"",
	},
	{
		// Lines are broken at whitespace, even if words
		// are too long. We do not break words.
		"foo bars bazzes",
		"foo\nbars\nbazzes",
		4,
		0,
		"",
	},
	{
		// A word that would run beyond the width is wrapped.
		"fo sop",
		"fo\nsop",
		4,
		0,
		"",
	},
	{
		// Do not break on non-breaking space.
		"foo bar\u00A0baz",
		"foo\nbar\u00A0baz",
		10,
		0,
		"",
	},
	{
		// Whitespace prefixing an explicit line break passes through.
		// A tab counts as one character.
		"foo\nb\t r\n baz",
		"foo\nb\t r\n baz",
		//"foo\t r baz"
		4,
		0,
		"",
	},
	{
		// Trailing whitespace is removed if it doesn't fit the width.
		// Runs of whitespace on which a line is broken are removed.
		"foo    \nb   ar   ",
		"foo\nb\nar",
		4,
		0,
		"",
	},
	{
		// An explicit line break at the end of the input is preserved.
		"foo bar baz\n",
		"foo\nbar\nbaz\n",
		4,
		0,
		"",
	},
	{
		// Explicit break are always preserved.
		"\nfoo bar\n\n\nbaz\n",
		"\nfoo\nbar\n\n\nbaz\n",
		4,
		0,
		"",
	},
	{
		// Spaces after a newline filling lines completely
		"\n\n foo\n\n\t bar",
		"\n\n foo\n\n\t bar",
		4,
		0,
		"",
	},
	{
		// Complete example:
		// " This is a list: \n\n\t* foo\n",
		// " This\nis a\nlist:\n\n\t* foo\n",
		//" This\nis a\nlist:\n\n\n\t* foo\n"
		" This is a list: \n\n\t* foo\n\t* bar\n\n\n\t* baz  \nBAM    ",
		" This\nis a\nlist:\n\n\t* foo\n\t* bar\n\n\n\t* baz\nBAM",
		6,
		0,
		"",
	},
	{
		// Multi-byte characters
		strings.Repeat("\u2584 ", 4),
		"\u2584 \u2584" + "\n" + "\u2584 \u2584",
		4,
		0,
		"",
	},
	{
		// Example with start (first line indentation)
		"aa bb cc dd ee ff gg",
		"aa\nbb cc\ndd ee\nff gg",
		// "aa\nbb\ncc\ndd\nee\nff\ngg"
		5,
		2,
		"",
	},
	{
		// Example with indentation
		"aa bb cc dd ee ff gg",
		"  aa bb\n  cc dd\n  ee ff\n  gg",
		5,
		0,
		"  ",
	},
	{
		// Start skips adding indentation at first line because it is assumed that
		// indentation has already happened on it.
		"aa bb cc dd ee ff gg",
		"aa\nxxbb cc\nxxdd ee\nxxff gg",
		5,
		2,
		"xx",
	},
	{
		"aa 人間 cc dd ee ff gg",
		"aa 人間\ncc dd\nee ff\ngg",
		5,
		0,
		"",
	},
	{
		// Long Text
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras in blandit odio, eget gravida eros. In tincidunt, dolor nec blandit elementum, lacus metus semper lacus, id elementum augue ipsum in est. Vivamus tempor orci eget augue faucibus efficitur.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit. Cras in blandit odio, eget gravida eros. In\ntincidunt, dolor nec blandit elementum, lacus metus semper lacus, id elementum augue ipsum in est.\nVivamus tempor orci eget augue faucibus efficitur.",
		100,
		0,
		"",
	},
}

func wUnit(typ WordWrapType, value string) WrapUnit {
	return WrapUnit{value: []byte(value), typ: typ}
}

func values(units []WrapUnit) string {
	str := ""
	for _, unit := range units {
		if len(str) > 0 {
			str += ", "
		}
		str += fmt.Sprintf("\"%s\"", string(unit.value))
	}

	return str
}

func TestFeedWordsForWrapping(t *testing.T) {
	fmt.Println("Feeder")

	testData := []struct {
		Input  string
		Result []WrapUnit
	}{
		{
			Input: "foo",
			Result: []WrapUnit{
				WordUnit("foo"),
			},
		},
		{
			Input: "foo bar",
			Result: []WrapUnit{
				WordUnit("foo"),
				SpaceUnit(" "),
				WordUnit("bar"),
			},
		},
		{
			Input: "\nfoo\n  bar",
			Result: []WrapUnit{
				newlineUnit,
				WordUnit("foo"),
				newlineUnit,
				SpaceUnit("  "),
				WordUnit("bar"),
			},
		},
		{
			Input: "foo bar\u00A0baz",
			Result: []WrapUnit{
				WordUnit("foo"),
				SpaceUnit(" "),
				WordUnit("bar\u00A0baz"),
			},
		},
		{
			Input: "foo\n\nbar",
			Result: []WrapUnit{
				WordUnit("foo"),
				newlineUnit,
				newlineUnit,
				WordUnit("bar"),
			},
		},
	}

	for _, data := range testData {
		var actual []WrapUnit
		FeedWordsForWrapping(data.Input, func(unit WrapUnit) uint {
			actual = append(actual, unit)
			return 0
		})

		assert.Equal(t, data.Result, actual,
			"Expected %s but got %s",
			values(data.Result), values(actual),
		)
	}
}

func TestWordWrapper(t *testing.T) {
	for i, tc := range cases {
		buf := bytes.NewBuffer([]byte{})
		wrapper := NewWordWrapper(buf, WrapOptions{
			Limit:       tc.Limit,
			StartsAt:    tc.StartsAt,
			Indentation: tc.Indentation,
		})

		wrapper.WrapString(tc.Input)

		actual := buf.String()
		assert.Equal(
			t,
			tc.Output,
			actual,
			"Case %d Input:\n\n`%s`\n\nExpected Output:\n\n`%s`\n\nActual Output:\n\n`%s`",
			i, tc.Input, tc.Output, actual,
		)
	}
}

func TestWordWrapperManual(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	wrapper := NewWordWrapper(buf, WrapOptions{
		Limit:       5,
		StartsAt:    2,
		Indentation: "xx",
	})

	units := []WrapUnit{
		WordUnit("aa"),
		SpaceUnit("  "),
		WordUnit("bb"),
		SpaceUnit(" "),
		WordUnit("cc"),
		SpaceUnit(" "),
		WordUnit("dd"),
		SpaceUnit(" "),
		WordUnit("ee"),
		SpaceUnit(" "),
		WordUnit("ff"),
		SpaceUnit(" "),
		WordUnit("gg"),
		SpaceUnit(" "),
		WordUnit("hh"),
		WordUnit("ii"),
		SpaceUnit(" "),
		WordUnit("jj"),
		WordUnit("kk"),
		WordUnit("ll"),
		SpaceUnit(" "),
		WordUnit("mm"),
		SpaceUnit(" "),
		WordUnit("nn"),
		WordUnit("oo"),
	}

	for _, unit := range units {
		wrapper.AddUnit(unit)
	}

	wrapper.FinalFlush()

	actual := buf.String()
	expected := "aa\nxxbb cc\nxxdd ee\nxxff gg\nxxhhii\nxxjjkkll\nxxmm\nxxnnoo"
	assert.Equal(t, expected, actual)
}
