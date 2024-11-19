package formathtml

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const indentString = "  "
const paragraphLength = 100

type NodePrinter func(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error)
type Conditional func(n *html.Node, level int, col uint) bool
type ConditionalAndContext[T comparable] func(n *html.Node, value T) bool

func conditionWithContext[T comparable](value T, cond ConditionalAndContext[T]) Conditional {
	return func(n *html.Node, level int, col uint) bool {
		return cond(n, value)
	}
}

// Document formats a HTML document.
func Document(w io.Writer, r io.Reader) (err error) {
	node, err := html.Parse(r)
	if err != nil {
		return err
	}
	return Nodes(w, []*html.Node{node})
}

// Fragment formats a fragment of a HTML document.
func Fragment(w io.Writer, r io.Reader) (err error) {
	context := &html.Node{
		Type: html.ElementNode,
	}
	nodes, err := html.ParseFragment(r, context)
	if err != nil {
		return err
	}
	return Nodes(w, nodes)
}

// Nodes formats a slice of HTML nodes.
func Nodes(w io.Writer, nodes []*html.Node) (err error) {
	colAfter := uint(0)
	for _, node := range nodes {
		if colAfter, err = printNode(w, node, 0, colAfter); err != nil {
			return
		}
	}
	return
}

// Is this node a tag with no end tag such as <meta> or <br>?
// http://www.w3.org/TR/html-markup/syntax.html#syntax-elements
func isEmptyElement(n *html.Node, _ int, _ uint) bool {
	switch n.DataAtom {
	case atom.Area, atom.Base, atom.Br, atom.Col, atom.Command, atom.Embed,
		atom.Hr, atom.Img, atom.Input, atom.Keygen, atom.Link,
		atom.Meta, atom.Param, atom.Source, atom.Track, atom.Wbr:
		return true
	}

	return false
}

func isBreakElement(n *html.Node, _ int, _ uint) bool {
	return n.DataAtom == atom.Br
}

func isNonEmptyElement(n *html.Node, level int, col uint) bool {
	return !isEmptyElement(n, level, col)
}

func isSpecialContentElement(n *html.Node, _ int, _ uint) bool {
	if n != nil {
		switch n.DataAtom {
		case atom.Style,
			atom.Script:
			return true
		}
	}
	return false
}

func isChildOfSpecialContentElement(n *html.Node, level int, col uint) bool {
	return isSpecialContentElement(n.Parent, level, col)
}

func isScriptWithSrcAttribute(n *html.Node, _ int, _ uint) bool {
	return n.DataAtom == atom.Script && hasSrcAttribute(n)
}

func hasSrcAttribute(n *html.Node) bool {
	for _, a := range n.Attr {
		if a.Key == "src" {
			return true
		}
	}
	return false
}

func isParagraphLike(n *html.Node, _ int, _ uint) bool {
	switch n.DataAtom {
	case atom.P, atom.Caption, atom.Figcaption:
		return true
	}

	return false
}

func isPre(n *html.Node, _ int, _ uint) bool {
	return n.DataAtom == atom.Pre
}

func isEmptyTextNode(n *html.Node, _ int, _ uint) bool {
	return n.Type == html.TextNode && strings.TrimSpace(n.Data) == ""
}

func getFirstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

func hasSingleTextChild(n *html.Node, _ int, _ uint) bool {
	return n != nil && n.FirstChild != nil && n.FirstChild == n.LastChild && n.FirstChild.Type == html.TextNode
}

func isSingleTextChild(n *html.Node, level int, col uint) bool {
	return hasSingleTextChild(n.Parent, level, col)
}

func isHtmlElement(n *html.Node, _ int, _ uint) bool {
	return n.DataAtom == atom.Html
}

func isChildOfParagraph(n *html.Node, level int, col uint) bool {
	return isParagraphLike(n.Parent, level, col)
}

func noNextSibling(n *html.Node, _ int, _ uint) bool {
	return n.NextSibling == nil
}

func noPrevSibling(n *html.Node, _ int, _ uint) bool {
	return n.PrevSibling == nil
}

func nextSiblingIsNotPunctuation(n *html.Node, _ int, _ uint) bool {
	return !unicode.IsPunct(getFirstRune(n.NextSibling.Data))
}

func nextSiblingIsElementNode(n *html.Node, _ int, _ uint) bool {
	return n.NextSibling.Type == html.ElementNode
}

func printNode(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	colAfter = col
	switch n.Type {
	case html.TextNode:
		return printTextNode(w, n, level, col)
	case html.ElementNode:
		return printElementNode(w, n, level, col)
	case html.CommentNode:
		return printCommentNode(w, n, level, col)
	case html.DoctypeNode:
		return printDoctypeNode(w, n, level, col)
	case html.DocumentNode:
		return printChildren(w, n, level, col)
	}
	return
}

func printDoctypeNode(w io.Writer, n *html.Node, _ int, _ uint) (colAfter uint, err error) {
	if err = html.Render(w, n); err != nil {
		return
	}

	return printNewLine(w, n, 0, 0)
}

func printCommentNode(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	if colAfter, err = printIndent(w, n, level, col); err != nil {
		return
	}

	colAfter = uint(7 + utf8.RuneCountInString(n.Data))
	_, err = fmt.Fprintf(w, "<!--%s-->\n", n.Data)

	return
}

func printTextNode(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	s := n.Data
	s = strings.TrimSpace(s)
	if s != "" {
		colAfter, err = runPrinters(
			printIf(
				allAre(
					not(isChildOfSpecialContentElement),
					not(isSingleTextChild),
					conditionWithContext(s, func(n *html.Node, str string) bool {
						return noPrevSibling(n, level, col) || !unicode.IsPunct(getFirstRune(s))
					}),
				),
				printIndent,
			),
		)(w, n, level, col)
		if err != nil {
			return
		}

		if isChildOfSpecialContentElement(n, level, colAfter) {
			scanner := bufio.NewScanner(strings.NewReader(s))
			for scanner.Scan() {
				t := scanner.Text()
				if _, err = fmt.Fprintln(w); err != nil {
					return
				}
				colAfter = 0 // after a new line
				if colAfter, err = printIndent(w, n, level, colAfter); err != nil {
					return
				}
				if _, err = fmt.Fprint(w, t); err != nil {
					return
				}
			}
			if err = scanner.Err(); err != nil {
				return
			}
			if _, err = fmt.Fprintln(w); err != nil {
				return
			}
		} else {
			if _, err = fmt.Fprint(w, s); err != nil {
				return
			}
			if !isSingleTextChild(n, level, colAfter) {
				if colAfter, err = printNewLine(w, n, level, colAfter); err != nil {
					return
				}
			}
		}
	}
	return
}

// The <pre> tag indicates that the text within it should always be formatted
// as is. See https://github.com/ericchiang/pup/issues/33
func printPreChild(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	switch n.Type {
	case html.TextNode:
		return runPrinters(
			printData,
			printDelegateChildren(printPreChild),
		)(w, n, level, col)

	case html.ElementNode:
		return runPrinters(
			printOpeningTag,
			printIf(isNonEmptyElement, printDelegateChildren(printPreChild)),
			printIf(isNonEmptyElement, printClosingTag),
		)(w, n, level, col)

	case html.CommentNode:
		return printCommentNode(w, n, level, col)

	case html.DoctypeNode, html.DocumentNode:
		return printDelegateChildren(printPreChild)(w, n, level, col)
	}

	return
}

func printOpeningTag(w io.Writer, n *html.Node, _ int, col uint) (colAfter uint, err error) {
	colAfter = col + uint(len(n.Data)+2) // 2 is for the angled brackets on both ends
	if _, err = fmt.Fprintf(w, "<%s", n.Data); err != nil {
		return
	}

	for _, a := range n.Attr {
		val := html.EscapeString(a.Val)
		colAfter += uint(len(a.Key) + len(val))
		if _, err = fmt.Fprintf(w, ` %s="%s"`, a.Key, val); err != nil {
			return
		}
	}

	_, err = fmt.Fprint(w, ">")

	return
}

func passOpeningTag(n *html.Node, wrapper *WordWrapper) (colAfter uint, err error) {
	wrapper.AddWord("<" + n.Data)
	for _, a := range n.Attr {
		val := html.EscapeString(a.Val)
		wrapper.AddSpaces(" ")
		wrapper.AddWord(fmt.Sprintf(`%s="%s"`, a.Key, val))
	}
	wrapper.AddSpaces("") // allows breaking if adding end bracket would exceed limit
	wrapper.AddWord(">")

	return wrapper.Column, nil
}

func printClosingTag(w io.Writer, n *html.Node, _ int, col uint) (colAfter uint, err error) {
	colAfter = col + uint(2+len(n.Data))
	_, err = fmt.Fprintf(w, "</%s>", n.Data)
	return
}

func passClosingTag(n *html.Node, wrapper *WordWrapper) (colAfter uint, err error) {
	wrapper.AddWord("</" + n.Data + ">")
	return wrapper.Column, nil
}

func printNewLine(w io.Writer, _ *html.Node, _ int, _ uint) (uint, error) {
	_, err := fmt.Fprint(w, "\n")
	return uint(0), err
}

func printData(w io.Writer, n *html.Node, _ int, col uint) (colAfter uint, err error) {
	colAfter = col + uint(utf8.RuneCountInString(n.Data))
	_, err = fmt.Fprint(w, n.Data)
	return
}

func printDelegateChildren(childPrinter NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
		colAfter = col
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if colAfter, err = childPrinter(w, c, level, colAfter); err != nil {
				return
			}
		}

		return
	}
}

func runPrinters(printers ...NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
		colAfter = col
		for _, printer := range printers {
			if colAfter, err = printer(w, n, level, colAfter); err != nil {
				return
			}
		}

		return
	}
}

func incrementLevel(addLevel int, printer NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int, col uint) (uint, error) {
		return printer(w, n, level+addLevel, col)
	}
}

func not(cf Conditional) Conditional {
	return func(n *html.Node, level int, col uint) bool {
		return !cf(n, level, col)
	}
}

func anyIs(cfs ...Conditional) Conditional {
	return func(n *html.Node, level int, col uint) bool {
		for _, cf := range cfs {
			if cf(n, level, col) {
				return true
			}
		}

		return false
	}
}

func allAre(cfs ...Conditional) Conditional {
	return func(n *html.Node, level int, col uint) bool {
		for _, cf := range cfs {
			if !cf(n, level, col) {
				return false
			}
		}

		return true
	}
}

func printIf(cf Conditional, printer NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
		colAfter = col
		if cf(n, level, col) {
			return printer(w, n, level, col)
		}

		return
	}
}

func printIfElse(cf Conditional, printerIfTrue, printerIfFalse NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int, col uint) (uint, error) {
		if cf(n, level, col) {
			return printerIfTrue(w, n, level, col)
		}

		return printerIfFalse(w, n, level, col)
	}
}

func printElementNode(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	switch {
	case isPre(n, level, col):
		return runPrinters(
			printIndent,
			printOpeningTag,
			printDelegateChildren(printPreChild),
			printClosingTag,
			printNewLine,
		)(w, n, level, col)

	case isParagraphLike(n, level, col):
		return printParagraphLikeNode(w, n, level, col)

	case isEmptyElement(n, level, col):
		return runPrinters(
			printIndent,
			printOpeningTag,
			printNewLine,
		)(w, n, level, col)

	case isScriptWithSrcAttribute(n, level, col):
		return runPrinters(
			printIndent,
			printOpeningTag,
			printClosingTag,
			printNewLine,
		)(w, n, level, col)

	default:
		return runPrinters(
			printIndent,
			printOpeningTag,
			printIf(not(hasSingleTextChild), printNewLine),
			printIfElse(
				isHtmlElement, printChildren, incrementLevel(1, printChildren),
			),
			printIf(
				anyIs(isSpecialContentElement, not(hasSingleTextChild)),
				printIndent,
			),
			printClosingTag,
			printIf(
				anyIs(noNextSibling, nextSiblingIsNotPunctuation, nextSiblingIsElementNode),
				printNewLine,
			),
		)(w, n, level, col)
	}
}

func printParagraphLikeNode(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	return runPrinters(
		printIndent,
		printOpeningTag,
		paragraphElementContents,
		printClosingTag,
		printNewLine,
	)(w, n, level, col)
}

func paragraphElementContents(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	lw := NewLineOrPassWriter(w)
	colPrep, err := runPrinters(
		printNewLine,
		incrementLevel(1, printParagraphChildren),
		func(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
			lw.Drain()
			return col, err
		},
	)(lw, n, level, col)
	if err != nil {
		return colPrep, err
	}

	return printIf(
		func(_ *html.Node, _ int, _ uint) bool {
			return lw.IsEndOfFirstLineReached()
		},
		runPrinters(
			printNewLine,
			printIndent,
		),
	)(w, n, level, colPrep)
}

func printParagraphChildren(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	child := n.FirstChild
	colAfter = col

	wrapper := NewWordWrapper(w, WrapOptions{
		Limit:       paragraphLength,
		StartsAt:    col,
		Indentation: indentAtLevel(level),
	})

	for child != nil {
		if colAfter, err = printParagraphNode(w, child, level, wrapper); err != nil {
			return
		}
		child = child.NextSibling
	}

	wrapper.FinalFlush()

	return
}

func printParagraphNode(w io.Writer, n *html.Node, level int, wrapper *WordWrapper) (colAfter uint, err error) {
	switch n.Type {
	case html.TextNode:
		return printParagraphTextNode(w, n, level, wrapper)
	case html.ElementNode:
		return printParagraphElementNode(w, n, level, wrapper)
	case html.CommentNode:
		return printCommentNode(w, n, level, wrapper.Column)
	case html.DoctypeNode:
		return printDoctypeNode(w, n, level, wrapper.Column)
	case html.DocumentNode:
		return printChildren(w, n, level, wrapper.Column)
	}

	return
}

var asciiSpace = [256]uint8{'\t': 1, '\n': 1, '\v': 1, '\f': 1, '\r': 1, ' ': 1}

func nonSpaceLeftIndex(s string) int {
	// Fast path for ASCII: look for the first ASCII non-space byte
	start := 0
	for ; start < len(s); start++ {
		c := s[start]
		if c >= utf8.RuneSelf {
			// If we run into a non-ASCII byte, fall back to the
			// slower unicode-aware method on the remaining bytes
			if !unicode.IsSpace(rune(c)) {
				break
			}
		}
		if asciiSpace[c] == 0 {
			break
		}
	}

	return start
}

func spaceIndexRight(start int, s string) int {
	// Now look for the first ASCII non-space byte from the end
	stop := len(s)
	for ; stop > start; stop-- {
		c := s[stop-1]
		if c >= utf8.RuneSelf {
			if !unicode.IsSpace(rune(c)) {
				break
			}
		}
		if asciiSpace[c] == 0 {
			break
		}
	}

	return stop
}

func trimSpace(s string) string {
	start := nonSpaceLeftIndex(s)
	stop := spaceIndexRight(start, s)

	return s[start:stop]
}

func trimSpaceLeft(s string) string {
	start := nonSpaceLeftIndex(s)

	return s[start:]
}

func trimSpaceRight(s string) string {
	stop := spaceIndexRight(0, s)
	return s[:stop]
}

func printParagraphTextNode(_ io.Writer, n *html.Node, level int, wrapper *WordWrapper) (colAfter uint, err error) {
	s := n.Data
	endChild := noNextSibling(n, level, colAfter)
	childOfP := isChildOfParagraph(n, level, colAfter)

	if childOfP {
		if noPrevSibling(n, level, colAfter) {
			s = trimSpaceLeft(s)
		}

		if endChild {
			s = trimSpaceRight(s)
		}
	}

	if s != "" {
		FeedWordsForWrapping(s, func(unit WrapUnit) uint {
			colAfter = wrapper.AddUnit(unit)
			return colAfter
		})

		if endChild {
			colAfter = 0
		}

		return
	}

	return
}

func isAtFirstColumn(_ *html.Node, _ int, col uint) bool {
	return col == 0
}

func printParagraphElementNode(w io.Writer, n *html.Node, level int, wrapper *WordWrapper) (colAfter uint, err error) {
	switch {

	case isBreakElement(n, level, wrapper.Column):
		passOpeningTag(n, wrapper)
		wrapper.AddGreedyNewLine()
		return wrapper.Column, nil

	case isEmptyElement(n, level, wrapper.Column):
		passOpeningTag(n, wrapper)
		return wrapper.Column, nil

	default:
		passOpeningTag(n, wrapper)
		child := n.FirstChild
		for child != nil {
			if colAfter, err = printParagraphNode(w, child, level, wrapper); err != nil {
				return
			}
			child = child.NextSibling
		}
		passClosingTag(n, wrapper)

		return wrapper.Column, nil
	}
}

func printChildren(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	child := n.FirstChild
	colAfter = col
	for child != nil {
		if colAfter, err = printNode(w, child, level, colAfter); err != nil {
			return
		}
		child = child.NextSibling
	}
	return
}

func indentAtLevel(level int) string {
	return strings.Repeat(indentString, level)
}

func printIndent(w io.Writer, _ *html.Node, level int, _ uint) (uint, error) {
	_, err := fmt.Fprint(w, indentAtLevel(level))
	return 0, err
}
