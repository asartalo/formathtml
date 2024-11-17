package formathtml

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineOrPass(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected string
	}{
		{
			name:     "string with no newline passes through",
			inputs:   []string{`foo bar baz`},
			expected: `foo bar baz`,
		},
		{
			name:     "strings written per word with no newline passes through",
			inputs:   []string{`foo`, ` bar`, ` baz`},
			expected: `foo bar baz`,
		},
		{
			name:     "strings written per word with newline in the middle passes through unchanged",
			inputs:   []string{`foo`, ` bar`, "\nbaz"},
			expected: "foo bar\nbaz",
		},
		{
			name:     "string with no newline passes through but with leading space discards leading space",
			inputs:   []string{"\t  foo bar baz"},
			expected: `foo bar baz`,
		},
		{
			name:     "multiple strings but with leading spaces discards leading spaces with no newline",
			inputs:   []string{"\t  ", `foo bar `, `baz`},
			expected: `foo bar baz`,
		},
		{
			name:     "multiple strings but with leading spaces retains leadings spaces with newline encountered",
			inputs:   []string{"\t  ", `foo bar `, "\nbaz"},
			expected: "\t  foo bar \nbaz",
		},
		{
			name:     "multiple strings but with leading spaces retains leadings spaces with newline encountered with extra writes",
			inputs:   []string{"\t  ", `foo bar `, "\nbaz", " qux", " quux\n", "corge"},
			expected: "\t  foo bar \nbaz qux quux\ncorge",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w := new(strings.Builder)
			lopWriter := NewLineOrPassWriter(w)
			for _, input := range test.inputs {
				lopWriter.Write([]byte(input))
			}
			lopWriter.Drain()
			assert.Equal(t, test.expected, w.String())
		})
	}
}
