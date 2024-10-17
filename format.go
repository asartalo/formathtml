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

type NodePrinter func(w io.Writer, n *html.Node, level int) (err error)
type NodePrinterAndContext[T comparable] func(w io.Writer, n *html.Node, level int, value T) (err error)
type Conditional func(n *html.Node) bool
type ConditionalAndContext[T comparable] func(n *html.Node, value T) bool

func printWithContext[T comparable](value T, printer NodePrinterAndContext[T]) NodePrinter {
	return func(w io.Writer, n *html.Node, level int) (err error) {
		return printer(w, n, level, value)
	}
}

func conditionWithContext[T comparable](value T, cond ConditionalAndContext[T]) Conditional {
	return func(n *html.Node) bool {
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
	for _, node := range nodes {
		if err = printNode(w, node, 0); err != nil {
			return
		}
	}
	return
}

// The <pre> tag indicates that the text within it should always be formatted
// as is. See https://github.com/ericchiang/pup/issues/33
func printPreChild(w io.Writer, n *html.Node, level int) (err error) {
	switch n.Type {
	case html.TextNode:
		return runPrinters(
			printData,
			printDelegateChildren(printPreChild),
		)(w, n, level)

	case html.ElementNode:
		return runPrinters(
			printOpeningTag,
			printIf(isNonEmptyElement, printDelegateChildren(printPreChild)),
			printIf(isNonEmptyElement, printClosingTag),
		)(w, n, level)

	case html.CommentNode:
		return printCommentNode(w, n, level)

	case html.DoctypeNode, html.DocumentNode:
		return printDelegateChildren(printPreChild)(w, n, level)
	}

	return
}

// Is this node a tag with no end tag such as <meta> or <br>?
// http://www.w3.org/TR/html-markup/syntax.html#syntax-elements
func isEmptyElement(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Area, atom.Base, atom.Br, atom.Col, atom.Command, atom.Embed,
		atom.Hr, atom.Img, atom.Input, atom.Keygen, atom.Link,
		atom.Meta, atom.Param, atom.Source, atom.Track, atom.Wbr:
		return true
	}

	return false
}

func isNonEmptyElement(n *html.Node) bool {
	return !isEmptyElement(n)
}

func isSpecialContentElement(n *html.Node) bool {
	if n != nil {
		switch n.DataAtom {
		case atom.Style,
			atom.Script:
			return true
		}
	}
	return false
}

func isChildOfSpecialContentElement(n *html.Node) bool {
	return isSpecialContentElement(n.Parent)
}

func isParagraphLike(n *html.Node) bool {
	switch n.DataAtom {
	case atom.P, atom.Caption, atom.Figcaption:
		return true
	}

	return false
}

func isPre(n *html.Node) bool {
	return n.DataAtom == atom.Pre
}

func isEmptyTextNode(n *html.Node) bool {
	return n.Type == html.TextNode && strings.TrimSpace(n.Data) == ""
}

func getFirstRune(s string) rune {
	r, _ := utf8.DecodeRuneInString(s)
	return r
}

func hasSingleTextChild(n *html.Node) bool {
	return n != nil && n.FirstChild != nil && n.FirstChild == n.LastChild && n.FirstChild.Type == html.TextNode
}

func isSingleTextChild(n *html.Node) bool {
	return hasSingleTextChild(n.Parent)
}

func isHtmlElement(n *html.Node) bool {
	return n.DataAtom == atom.Html
}

func noNextSibling(n *html.Node) bool {
	return n.NextSibling == nil
}

func noPrevSibling(n *html.Node) bool {
	return n.PrevSibling == nil
}

func nextSiblingIsNotPunctuation(n *html.Node) bool {
	return !unicode.IsPunct(getFirstRune(n.NextSibling.Data))
}

func nextSiblingIsElementNode(n *html.Node) bool {
	return n.NextSibling.Type == html.ElementNode
}

func printNode(w io.Writer, n *html.Node, level int) (err error) {
	switch n.Type {
	case html.TextNode:
		return printTextNode(w, n, level)
	case html.ElementNode:
		return printElementNode(w, n, level)
	case html.CommentNode:
		return printCommentNode(w, n, level)
	case html.DoctypeNode:
		return printDoctypeNode(w, n, level)
	case html.DocumentNode:
		return printChildren(w, n, level)
	}
	return
}

func printDoctypeNode(w io.Writer, n *html.Node, _ int) (err error) {
	if err = html.Render(w, n); err != nil {
		return
	}

	return printNewLine(w, n, 0)
}

func printCommentNode(w io.Writer, n *html.Node, level int) (err error) {
	if err = printIndent(w, n, level); err != nil {
		return
	}

	_, err = fmt.Fprintf(w, "<!--%s-->\n", n.Data)

	return
}

func printTextNode(w io.Writer, n *html.Node, level int) (err error) {
	s := n.Data
	s = strings.TrimSpace(s)
	if s != "" {
		if !isChildOfSpecialContentElement(n) && !isSingleTextChild(n) &&
			(noPrevSibling(n) || conditionWithContext(s, func(n *html.Node, str string) bool {
				return !unicode.IsPunct(getFirstRune(s))
			})(n)) {
			if err = printIndent(w, n, level); err != nil {
				return
			}
		}
		if isChildOfSpecialContentElement(n) {
			scanner := bufio.NewScanner(strings.NewReader(s))
			for scanner.Scan() {
				t := scanner.Text()
				if _, err = fmt.Fprintln(w); err != nil {
					return
				}
				if err = printIndent(w, n, level); err != nil {
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
			if !isSingleTextChild(n) {
				if err = printNewLine(w, n, level); err != nil {
					return
				}
			}
		}
	}
	return
}

func printOpeningTag(w io.Writer, n *html.Node, _ int) (err error) {
	if _, err = fmt.Fprintf(w, "<%s", n.Data); err != nil {
		return
	}

	for _, a := range n.Attr {
		val := html.EscapeString(a.Val)
		if _, err = fmt.Fprintf(w, ` %s="%s"`, a.Key, val); err != nil {
			return
		}
	}

	_, err = fmt.Fprint(w, ">")

	return
}

func printClosingTag(w io.Writer, n *html.Node, _ int) (err error) {
	_, err = fmt.Fprintf(w, "</%s>", n.Data)
	return
}

func printNewLine(w io.Writer, _ *html.Node, _ int) (err error) {
	_, err = fmt.Fprint(w, "\n")
	return
}

func printData(w io.Writer, n *html.Node, _ int) (err error) {
	_, err = fmt.Fprint(w, n.Data)
	return
}

func printDelegateChildren(childPrinter NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int) (err error) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = childPrinter(w, c, level); err != nil {
				return
			}
		}

		return
	}
}

func runPrinters(printers ...NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int) (err error) {
		for _, printer := range printers {
			if err = printer(w, n, level); err != nil {
				return
			}
		}

		return
	}
}

func incrementLevel(addLevel int, printer NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int) (err error) {
		return printer(w, n, level+addLevel)
	}
}

func printIf(cf Conditional, printer NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int) (err error) {
		if cf(n) {
			return printer(w, n, level)
		}

		return
	}
}

func not(cf Conditional) Conditional {
	return func(n *html.Node) bool {
		return !cf(n)
	}
}

func any(cfs ...Conditional) Conditional {
	return func(n *html.Node) bool {
		for _, cf := range cfs {
			if cf(n) {
				return true
			}
		}

		return false
	}
}

func printIfElse(cf Conditional, printerIfTrue, printerIfFalse NodePrinter) NodePrinter {
	return func(w io.Writer, n *html.Node, level int) (err error) {
		if cf(n) {
			return printerIfTrue(w, n, level)
		}

		return printerIfFalse(w, n, level)
	}
}

func printElementNode(w io.Writer, n *html.Node, level int) (err error) {
	switch {
	case isPre(n):
		return runPrinters(
			printIndent,
			printOpeningTag,
			printDelegateChildren(printPreChild),
			printClosingTag,
			printNewLine,
		)(w, n, level)

	case isParagraphLike(n):
		return runPrinters(
			printIndent,
			printOpeningTag,
			printNewLine,
			incrementLevel(1, printParagraphChildren),
			printNewLine,
			printIndent,
			printClosingTag,
			printNewLine,
		)(w, n, level)

	case isEmptyElement(n):
		return runPrinters(
			printIndent,
			printOpeningTag,
			printNewLine,
		)(w, n, level)

	default:
		return runPrinters(
			printIndent,
			printOpeningTag,
			printIf(not(hasSingleTextChild), printNewLine),
			printIfElse(
				isHtmlElement, printChildren, incrementLevel(1, printChildren),
			),
			printIf(
				any(isSpecialContentElement, not(hasSingleTextChild)),
				printIndent,
			),
			printClosingTag,
			printIf(
				any(noNextSibling, nextSiblingIsNotPunctuation, nextSiblingIsElementNode),
				printNewLine,
			),
		)(w, n, level)
	}

}

func printParagraphChildren(w io.Writer, n *html.Node, level int) (err error) {
	child := n.FirstChild
	var col uint = 0
	for child != nil {
		if col, err = printParagraphNode(w, child, level, col); err != nil {
			return
		}
		child = child.NextSibling
	}
	return
}

func printParagraphNode(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	switch n.Type {
	case html.TextNode:
		return printParagraphTextNode(w, n, level, col)
	case html.ElementNode:
		err = printElementNode(w, n, level)
	case html.CommentNode:
		err = printCommentNode(w, n, level)
	case html.DoctypeNode:
		err = printDoctypeNode(w, n, level)
	case html.DocumentNode:
		err = printChildren(w, n, level)
	}

	return
}

func printParagraphTextNode(w io.Writer, n *html.Node, level int, col uint) (colAfter uint, err error) {
	s := n.Data
	s = strings.TrimSpace(s)
	if s != "" {
		WrapString(s, w, WrapOptions{
			Limit:       paragraphLength,
			StartsAt:    col,
			Indentation: indentAtLevel(level),
		})
		if !hasSingleTextChild(n.Parent) {
			if _, err = fmt.Fprint(w, "\n"); err != nil {
				return
			}
		}
	}

	return
}

func printChildren(w io.Writer, n *html.Node, level int) (err error) {
	child := n.FirstChild
	for child != nil {
		if err = printNode(w, child, level); err != nil {
			return
		}
		child = child.NextSibling
	}
	return
}

func indentAtLevel(level int) string {
	return strings.Repeat(indentString, level)
}

func printIndent(w io.Writer, _ *html.Node, level int) (err error) {
	_, err = fmt.Fprint(w, indentAtLevel(level))
	return err
}
