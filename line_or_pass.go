package formathtml

import (
	"bytes"
	"io"
	"unicode"
)

type LineOrPassWriter struct {
	// The writer to write to.
	writer io.Writer

	leadingSpaceBuffer bytes.Buffer
	lineBuffer         bytes.Buffer

	lineBufferStart       bool
	endOfFirstLineReached bool
}

// NewLineOrPassWriter creates a new LineOrPassWriter.
func NewLineOrPassWriter(writer io.Writer) *LineOrPassWriter {
	return &LineOrPassWriter{
		writer: writer,
	}
}

func (l *LineOrPassWriter) IsEndOfFirstLineReached() bool {
	return l.endOfFirstLineReached
}

// Write writes the given bytes to the writer.
func (l *LineOrPassWriter) Write(bytes []byte) (n int, err error) {
	if l.endOfFirstLineReached {
		return l.writer.Write(bytes)
	}

	for _, b := range string(bytes) {
		if l.endOfFirstLineReached {
			l.lineBuffer.WriteRune(b)
			continue
		}

		if !l.lineBufferStart && !unicode.IsSpace(b) {
			l.lineBufferStart = true
		}

		if l.lineBufferStart {
			if b == '\n' {
				l.endOfFirstLineReached = true
			}
			l.lineBuffer.WriteRune(b)
		} else if unicode.IsSpace(b) {
			l.leadingSpaceBuffer.WriteRune(b)
		}
	}

	if l.endOfFirstLineReached {
		return l.Drain()
	}

	return 0, nil
}

// Drain signals that no new data will be written and flushes the buffers.
// Leading whitespace is discarded if the string is a single line.
func (l *LineOrPassWriter) Drain() (n int, err error) {
	var n64 int64

	if l.endOfFirstLineReached {
		n64, err = l.leadingSpaceBuffer.WriteTo(l.writer)
		if err != nil {
			return int(n64), err
		}
	}

	n64b, err := l.lineBuffer.WriteTo(l.writer)
	n64 += n64b

	return int(n64), err
}
