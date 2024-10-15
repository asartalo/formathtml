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
	"io"
	"unicode"
	"unicode/utf8"
)

const nbsp = 0xA0

type WrapOptions struct {
	Limit       uint
	StartsAt    uint
	Indentation string
}

func runeToUtf8(r rune) []byte {
	size := utf8.RuneLen(r)

	bs := make([]byte, size)

	utf8.EncodeRune(bs[0:], r)

	return bs
}

// WrapString wraps the given string within lim width in characters.
//
// Wrapping is currently naive and only happens at white-space. A future
// version of the library will implement smarter wrapping. This means that
// pathological cases can dramatically reach past the limit, such as a very
// long word. startsAt
func WrapString(s string, writer io.Writer, options WrapOptions) {
	// Initialize a buffer with a slightly larger size to account for breaks
	lim := options.Limit
	indentation := options.Indentation
	startsAt := options.StartsAt
	indentBytes := []byte(indentation)

	var current = startsAt
	var wordBuf, spaceBuf bytes.Buffer
	var wordBufLen, spaceBufLen uint

	if len(indentation) > 0 && startsAt == 0 {
		writer.Write(indentBytes)
	}

	for _, char := range s {
		if char == '\n' {
			if wordBuf.Len() == 0 {
				if current+spaceBufLen > lim {
					current = 0
				} else {
					current += spaceBufLen
					spaceBuf.WriteTo(writer)
				}
				spaceBuf.Reset()
				spaceBufLen = 0
			} else {
				current += spaceBufLen + wordBufLen
				spaceBuf.WriteTo(writer)
				spaceBuf.Reset()
				spaceBufLen = 0
				wordBuf.WriteTo(writer)
				wordBuf.Reset()
				wordBufLen = 0
			}
			writer.Write(runeToUtf8(char))
			current = 0
		} else if unicode.IsSpace(char) && char != nbsp {
			if spaceBuf.Len() == 0 || wordBuf.Len() > 0 {
				current += spaceBufLen + wordBufLen
				spaceBuf.WriteTo(writer)
				spaceBuf.Reset()
				spaceBufLen = 0
				wordBuf.WriteTo(writer)
				wordBuf.Reset()
				wordBufLen = 0
			}

			spaceBuf.WriteRune(char)
			spaceBufLen++
		} else {
			wordBuf.WriteRune(char)
			wordBufLen++

			if current+wordBufLen+spaceBufLen > lim && wordBufLen < lim {
				writer.Write(runeToUtf8('\n'))
				writer.Write(indentBytes)
				current = 0
				spaceBuf.Reset()
				spaceBufLen = 0
			}
		}
	}

	if wordBuf.Len() == 0 {
		if current+spaceBufLen <= lim {
			spaceBuf.WriteTo(writer)
		}
	} else {
		spaceBuf.WriteTo(writer)
		wordBuf.WriteTo(writer)
	}
}
