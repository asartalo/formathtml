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
func printPreChild(w io.Writer, n *html.Node) (err error) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		if _, err = fmt.Fprint(w, s); err != nil {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = printPreChild(w, c); err != nil {
				return
			}
		}
	case html.ElementNode:
		if _, err = fmt.Fprintf(w, "<%s", n.Data); err != nil {
			return
		}
		for _, a := range n.Attr {
			val := html.EscapeString(a.Val)
			if _, err = fmt.Fprintf(w, ` %s="%s"`, a.Key, val); err != nil {
				return
			}
		}
		if _, err = fmt.Fprint(w, ">"); err != nil {
			return
		}
		if !isEmptyElement(n) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if err = printPreChild(w, c); err != nil {
					return
				}
			}
			if _, err = fmt.Fprintf(w, "</%s>", n.Data); err != nil {
				return
			}
		}
	case html.CommentNode:
		data := n.Data
		if _, err = fmt.Fprintf(w, "<!--%s-->\n", data); err != nil {
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = printPreChild(w, c); err != nil {
				return
			}
		}
	case html.DoctypeNode, html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = printPreChild(w, c); err != nil {
				return
			}
		}
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

func printDoctypeNode(w io.Writer, n *html.Node, level int) (err error) {
	if err = html.Render(w, n); err != nil {
		return
	}

	_, err = fmt.Fprint(w, "\n")
	return
}

func printCommentNode(w io.Writer, n *html.Node, level int) (err error) {
	if err = printIndent(w, level); err != nil {
		return
	}
	if _, err = fmt.Fprintf(w, "<!--%s-->\n", n.Data); err != nil {
		return
	}

	return printChildren(w, n, level)

}

func printTextNode(w io.Writer, n *html.Node, level int) (err error) {
	s := n.Data
	s = strings.TrimSpace(s)
	if s != "" {
		if !isSpecialContentElement(n.Parent) && !hasSingleTextChild(n.Parent) &&
			(n.PrevSibling == nil || !unicode.IsPunct(getFirstRune(s))) {
			if err = printIndent(w, level); err != nil {
				return
			}
		}
		if isSpecialContentElement(n.Parent) {
			scanner := bufio.NewScanner(strings.NewReader(s))
			for scanner.Scan() {
				t := scanner.Text()
				if _, err = fmt.Fprintln(w); err != nil {
					return
				}
				if err = printIndent(w, level); err != nil {
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
			if !hasSingleTextChild(n.Parent) {
				if _, err = fmt.Fprint(w, "\n"); err != nil {
					return
				}
			}
		}
	}
	return
}

func printOpeningTag(w io.Writer, n *html.Node) (err error) {
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

func printElementNode(w io.Writer, n *html.Node, level int) (err error) {
	switch {
	case isPre(n):
		if err = printIndent(w, level); err != nil {
			return
		}

		if err = printOpeningTag(w, n); err != nil {
			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err = printPreChild(w, c); err != nil {
				return
			}
		}
		if _, err = fmt.Fprintf(w, "</%s>", n.Data); err != nil {
			return
		}
		if _, err = fmt.Fprint(w, "\n"); err != nil {
			return
		}
	case isParagraphLike(n):
		if err = printIndent(w, level); err != nil {
			return
		}

		if err = printOpeningTag(w, n); err != nil {
			return
		}
		if _, err = fmt.Fprint(w, "\n"); err != nil {
			return
		}
		childLevel := level + 1
		if err = printParagraphChildren(w, n, childLevel); err != nil {
			return
		}
		if _, err = fmt.Fprint(w, "\n"); err != nil {
			return
		}
		if err = printIndent(w, level); err != nil {
			return
		}
		if _, err = fmt.Fprintf(w, "</%s>", n.Data); err != nil {
			return
		}
		if _, err = fmt.Fprint(w, "\n"); err != nil {
			return
		}
	case isEmptyElement(n):
		if err = printIndent(w, level); err != nil {
			return
		}

		if err = printOpeningTag(w, n); err != nil {
			return
		}

		if _, err = fmt.Fprint(w, "\n"); err != nil {
			return
		}
	default:
		childLevel := level + 1
		// Do not indent children of HTML elements
		if n.DataAtom == atom.Html {
			childLevel = level
		}
		if err = printIndent(w, level); err != nil {
			return
		}

		if err = printOpeningTag(w, n); err != nil {
			return
		}

		if !hasSingleTextChild(n) {
			if _, err = fmt.Fprint(w, "\n"); err != nil {
				return
			}
		}
		if err = printChildren(w, n, childLevel); err != nil {
			return
		}
		if isSpecialContentElement(n) || !hasSingleTextChild(n) {
			if err = printIndent(w, level); err != nil {
				return
			}
		}
		if _, err = fmt.Fprintf(w, "</%s>", n.Data); err != nil {
			return
		}

		if n.NextSibling == nil ||
			(!unicode.IsPunct(getFirstRune(n.NextSibling.Data)) || n.NextSibling.Type == html.ElementNode) {
			if _, err = fmt.Fprint(w, "\n"); err != nil {
				return
			}
		}
	}

	return
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

const paragraphLength = 100

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
	return strings.Repeat("  ", level)
}

func printIndent(w io.Writer, level int) (err error) {
	_, err = fmt.Fprint(w, indentAtLevel(level))
	return err
}
