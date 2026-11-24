package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/SeanMcP/ste-check/ste"
)

type palette struct{ red, green, yellow, dim, bold, mark, reset string }

func colors(on bool) palette {
	if !on {
		return palette{}
	}
	return palette{
		red: "\x1b[31m", green: "\x1b[32m", yellow: "\x1b[33m", dim: "\x1b[2m",
		bold: "\x1b[1m", mark: "\x1b[1;4;31m", reset: "\x1b[0m",
	}
}

// report prints each sentence that has issues, then a summary. It returns the issue count.
func report(w io.Writer, sentences []ste.Sentence, t ste.TextType, maxWords int, color bool) int {
	c := colors(color)
	issues, flagged := printSentences(w, sentences, color)

	checked := fmt.Sprintf("Checked as a %s (max %d words per sentence).", t, maxWords)
	if issues == 0 {
		fmt.Fprintf(w, "%s✓ %s with no issues.%s %s%s%s\n",
			c.green, plural(len(sentences), "sentence"), c.reset, c.dim, checked, c.reset)
	} else {
		fmt.Fprintf(w, "%s%s✗ %s in %d of %s.%s %s%s%s\n",
			c.bold, c.red, plural(issues, "issue"), flagged, plural(len(sentences), "sentence"), c.reset,
			c.dim, checked, c.reset)
	}
	return issues
}

// printSentences prints each sentence that has issues. It returns the issue
// count and the number of sentences with issues.
func printSentences(w io.Writer, sentences []ste.Sentence, color bool) (issues, flagged int) {
	c := colors(color)
	for i, s := range sentences {
		if len(s.Issues) == 0 {
			continue
		}
		issues += len(s.Issues)
		flagged++
		fmt.Fprintf(w, "%s%3d%s  %s\n", c.dim, i+1, c.reset, highlight(s, c, color))
		for _, issue := range s.Issues {
			fmt.Fprintf(w, "     %s✗%s %s%-19s%s %s\n", c.red, c.reset, c.yellow, issue.Rule, c.reset, issue.Message)
		}
		fmt.Fprintln(w)
	}
	return issues, flagged
}

// highlight marks flagged words with color, or with [brackets] when color is off.
func highlight(s ste.Sentence, c palette, color bool) string {
	open, close := "[", "]"
	if color {
		open, close = c.mark, c.reset
	}

	var b strings.Builder
	pos := 0
	for _, issue := range s.Issues {
		if issue.End == 0 || issue.Start < pos {
			continue // whole-sentence issue, or overlaps a span already marked
		}
		b.WriteString(s.Text[pos:issue.Start])
		b.WriteString(open + s.Text[issue.Start:issue.End] + close)
		pos = issue.End
	}
	b.WriteString(s.Text[pos:])
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
