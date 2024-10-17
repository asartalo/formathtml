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
	"fmt"
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

type WordWrapType int

const (
	NullUnit WordWrapType = iota
	Spaces
	NewLine
	Word
)

func lableWType(t WordWrapType) string {
	switch t {
	case Spaces:
		return "Spaces"
	case NewLine:
		return "NewLine"
	case Word:
		return "Word"
	}

	return "NullUnit"
}

type WrapUnit struct {
	value []byte
	typ   WordWrapType
	width uint
}

func (unit WrapUnit) Merge(other WrapUnit) WrapUnit {
	if unit.typ == NullUnit {
		return other
	}

	if unit.typ != other.typ {
		return unit
	}

	if unit.typ == NewLine {
		return unit
	}

	return WrapUnit{
		value: append(unit.value, other.value...),
		typ:   unit.typ,
		width: unit.width + other.width,
	}
}

func (unit WrapUnit) IsNull() bool {
	return unit.typ == NullUnit
}

var newlineUnit = WrapUnit{typ: NewLine, value: newlineBytes, width: 0}
var nullUnit = WrapUnit{typ: NullUnit}

func wordToFeed(typ WordWrapType, str string) WrapUnit {
	switch typ {
	case NewLine:
		return newlineUnit
	case Word:
		return WordUnit(str)
	case Spaces:
		return SpaceUnit(str)
	}
	return nullUnit
}

func FeedWordsForWrapping(s string, eater func(unit WrapUnit) uint) {
	str := ""
	var lastWordType WordWrapType

	for _, char := range s {
		var currentWordType WordWrapType
		if char == '\n' {
			currentWordType = NewLine
		} else if unicode.IsSpace(char) && char != nbsp {
			currentWordType = Spaces
		} else {
			currentWordType = Word
		}

		if lastWordType != NullUnit {
			if lastWordType != currentWordType || char == '\n' {
				eater(wordToFeed(lastWordType, str))
				str = ""
			}
		}

		str += string(char)

		lastWordType = currentWordType
	}

	if len(str) > 0 {
		eater(wordToFeed(lastWordType, str))
	}
}

type UnitPair struct {
	Word              WrapUnit
	LeadSpace         WrapUnit
	precededByNewLine bool
}

func NewUnitPair(precededByNewLine bool) *UnitPair {
	return &UnitPair{
		Word:              nullUnit,
		LeadSpace:         nullUnit,
		precededByNewLine: precededByNewLine,
	}
}

func (pair *UnitPair) isPrecededByNewLine() bool {
	return pair.precededByNewLine
}

func (pair *UnitPair) IsNull() bool {
	return pair.Word.IsNull() && pair.LeadSpace.IsNull()
}

func (pair *UnitPair) HasWord() bool {
	return !pair.Word.IsNull()
}

func (pair *UnitPair) Width() uint {
	return pair.Word.width + pair.LeadSpace.width
}

func (pair *UnitPair) WordWidth() uint {
	return pair.Word.width
}

func (pair *UnitPair) AddSpace(unit WrapUnit) bool {
	if unit.typ != Spaces {
		return false
	}

	pair.LeadSpace = pair.LeadSpace.Merge(unit)
	return true
}

func (pair *UnitPair) AddWord(unit WrapUnit) bool {
	if unit.typ != Word {
		return false
	}

	pair.Word = pair.Word.Merge(unit)
	return true
}

func (pair *UnitPair) Write(writer io.Writer, withSpace bool) int {
	var spaceLength int
	var wrote []byte

	if withSpace {
		spaceLength, _ = writer.Write(pair.LeadSpace.value)
		wrote = append(wrote, pair.LeadSpace.value...)
	}
	wordLength, _ := writer.Write(pair.Word.value)
	wrote = append(wrote, pair.Word.value...)

	return spaceLength + wordLength
}

type Line struct {
	pairs []*UnitPair
	width uint
	limit uint
}

func NewLineObject(start uint, limit uint) *Line {
	return &Line{
		width: start,
		limit: limit,
	}
}

func (l *Line) AppendPair(pair *UnitPair) {
	if pair.IsNull() {
		return
	}

	var widthToUse uint
	if len(l.pairs) == 0 && !pair.isPrecededByNewLine() {
		widthToUse = pair.WordWidth()
	} else {
		widthToUse = pair.Width()
	}

	l.pairs = append(l.pairs, pair)
	l.width += widthToUse
}

func (l *Line) IsLastPair(pair *UnitPair) bool {
	return l.LastPair() == pair
}

func (l *Line) LastPair() *UnitPair {
	length := len(l.pairs)
	if length == 0 {
		return nil
	}

	return l.pairs[length-1]
}

func (l *Line) Width() uint {
	return l.width
}

func (l *Line) Preview() string {
	lastIndex := len(l.pairs) - 1
	b := []byte{}

	for i, pair := range l.pairs {
		if i == lastIndex && !pair.HasWord() { // do not print trailing spaces
			break
		}

		b = append(b, pair.LeadSpace.value...)
		b = append(b, pair.Word.value...)
	}

	return string(b)
}

func (l *Line) Write(writer io.Writer) int {
	lastIndex := len(l.pairs) - 1
	written := 0

	for i, pair := range l.pairs {
		if i == lastIndex && !pair.HasWord() { // do not print trailing spaces
			break
		}

		length := pair.Write(writer, i > 0 || pair.isPrecededByNewLine())
		written += length
	}

	return written
}

func (l *Line) IsPrecededByNewLine() bool {
	if len(l.pairs) == 0 {
		return false
	}

	return l.pairs[0].isPrecededByNewLine()
}

func (l *Line) NotEmpty() bool {
	return l.width > 0
}

func (l *Line) Fits(width uint) bool {
	return len(l.pairs) == 0 || l.width+width <= l.limit
}

func (l *Line) PairFits(pair *UnitPair) bool {
	return l.Fits(pair.Width())
}

func (l *Line) Filled() bool {
	return l.width >= l.limit
}

func (l *Line) PopLast() *UnitPair {
	length := len(l.pairs)

	if length == 0 {
		return nil
	}

	lastPair := l.LastPair()
	l.pairs = l.pairs[:(length - 1)]
	l.width -= lastPair.Width()

	return lastPair
}

type WordWrapper struct {
	WrapOptions
	Writer           io.Writer
	Column           uint
	started          bool
	flushed          bool
	indentationBytes []byte
	lastUnit         WrapUnit
	currentLine      *Line
	currentPair      *UnitPair
	filledLineLast   bool
}

func NewWordWrapper(writer io.Writer, options WrapOptions) *WordWrapper {
	return &WordWrapper{
		WrapOptions:      options,
		Writer:           writer,
		indentationBytes: []byte(options.Indentation),
		lastUnit:         nullUnit,
		currentPair:      NewUnitPair(true),
		currentLine:      NewLineObject(options.StartsAt, options.Limit),
	}
}

func (ww *WordWrapper) WrapString(s string) {
	FeedWordsForWrapping(s, ww.AddUnit)
	ww.FinalFlush()
}

func (ww *WordWrapper) FinalFlush() {
	if ww.currentPair.HasWord() && !ww.currentLine.IsLastPair(ww.currentPair) {
		ww.appendPair(ww.currentPair)
	}

	if ww.currentLine.NotEmpty() {
		ww.flushLine()
	}
}

var newlineBytes = []byte("\n")
var spaceBytes = []byte(" ")

func WordUnit(word string) WrapUnit {
	return WrapUnit{
		value: []byte(word),
		typ:   Word,
		width: uint(utf8.RuneCountInString(word)),
	}
}

func SpaceUnit(spaces string) WrapUnit {
	return WrapUnit{value: []byte(spaces), typ: Spaces, width: uint(utf8.RuneCountInString(spaces))}
}

func (ww *WordWrapper) AddWord(word string) uint {
	return ww.AddUnit(WordUnit(word))
}

func (ww *WordWrapper) AddSpaces(spaces string) uint {
	return ww.AddUnit(SpaceUnit(spaces))
}

func (ww *WordWrapper) AddNewLine() uint {
	return ww.AddUnit(newlineUnit)
}

func unitValues(units []WrapUnit) string {
	str := ""
	for _, unit := range units {
		str += fmt.Sprintf(" \"%s\" (%s)\n", string(unit.value), lableWType(unit.typ))
	}

	return str
}

func (ww *WordWrapper) AddUnit(unit WrapUnit) uint {
	aNewLine := !ww.started || ww.lastUnit.typ == NewLine

	switch unit.typ {
	case NullUnit:
		return 0

	case NewLine:
		if ww.currentPair.HasWord() && !ww.currentLine.IsLastPair(ww.currentPair) {
			ww.appendPair(ww.currentPair)
		}
		if ww.lastUnit.typ != NewLine {
			ww.flushLine()
			ww.currentPair = NewUnitPair(true)
		}

		ww.writeNewLine()

	case Spaces:
		if ww.lastUnit.typ != Spaces {
			if !ww.currentLine.PairFits(ww.currentPair) {
				ww.flushLine()
			}
			if ww.currentPair.HasWord() {
				ww.appendPair(ww.currentPair)
			}
			if ww.currentLine.Filled() {
				ww.flushLine()
			}
			ww.currentPair = NewUnitPair(aNewLine)
			ww.currentPair.AddSpace(unit)
		}

	case Word:
		ww.currentPair.AddWord(unit)
		if !ww.currentLine.PairFits(ww.currentPair) {
			ww.flushLine()
		}
	}

	ww.started = true
	ww.lastUnit = unit
	return 0
}

func (ww *WordWrapper) appendPair(pair *UnitPair) {
	ww.currentLine.AppendPair(pair)
}

func (ww *WordWrapper) writeNewLine() {
	ww.Writer.Write(newlineBytes)
}

func (ww *WordWrapper) flushLine() {
	if !ww.currentLine.NotEmpty() {
		return
	}

	if ww.flushed && !ww.currentLine.IsPrecededByNewLine() {
		ww.writeNewLine()
	}

	if ww.flushed || ww.StartsAt == 0 {
		ww.Writer.Write(ww.indentationBytes)
	}
	ww.filledLineLast = false
	ww.currentLine.Write(ww.Writer)
	ww.currentLine = NewLineObject(0, ww.Limit)
	ww.flushed = true
}

func discardTrailingSpaces(line []WrapUnit) []WrapUnit {
	lastIndex := len(line) - 1
	for i := lastIndex; i > -1; i-- {
		if line[i].typ != Spaces {
			return line[:i+1]
		}
	}

	return line
}
