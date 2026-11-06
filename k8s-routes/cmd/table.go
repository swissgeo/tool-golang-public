package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/swissgeo/tool-golang-public/lib/fmtc"
	"golang.org/x/term"
)

const columnSeparator = "  "

var tableHeader = []string{"MATCH", "PRIORITY", "SERVICES", "MIDDLEWARES", "NAME"}

var markerColors = map[Marker]fmtc.Color{
	NoMarker:      fmtc.NoColor,
	MarkerNew:     fmtc.Green,
	MarkerChanged: fmtc.Yellow,
}

// cell is the text of a table cell with optional colored parts. The colors are applied after the
// column widths are computed, so that the escape codes don't break the alignment.
type cell []part

type part struct {
	text  string
	color fmtc.Color
}

func (c cell) width() int {
	n := 0
	for _, p := range c {
		n += utf8.RuneCountInString(p.text)
	}
	return n
}

func (c cell) render(color bool) string {
	var sb strings.Builder
	for _, p := range c {
		if color && p.color != fmtc.NoColor {
			sb.WriteString(fmtc.Colorise(p.color, p.text))
		} else {
			sb.WriteString(p.text)
		}
	}
	return sb.String()
}

func plain(text string) cell {
	return cell{{text: text}}
}

// IsTerminal reports whether w is a terminal.
func IsTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// UseColor reports whether the output to w should be colorized: only if it is a terminal.
func UseColor(w io.Writer) bool {
	return !NoColor && IsTerminal(w)
}

// warnf writes a warning line to w, in yellow if w is a terminal.
func warnf(w io.Writer, format string, args ...any) {
	msg := "Warning: " + fmt.Sprintf(format, args...)
	if UseColor(w) {
		msg = fmtc.Colorise(fmtc.Yellow, msg)
	}
	fmt.Fprintln(w, msg)
}

// TableOptions controls the rendering of the table.
type TableOptions struct {
	// Color colorizes the computed priorities and the markers.
	Color bool
	// SplitMatch puts each top-level && clause of the match rules longer than splitMatchLength on
	// its own line, to keep the MATCH column narrow. Off when the output is piped, so that a route
	// stays on one greppable line.
	SplitMatch bool
}

// splitMatchLength is the length above which a match rule is split when SplitMatch is set.
const splitMatchLength = 80

// matchContinuation prefixes the continuation lines of a split match rule.
const matchContinuation = "  && "

// RenderTable writes the rows as a table with aligned columns, the hosts separated by an empty line.
// The rows are expected to be sorted by host (see BuildRows). Computed priorities are shown in
// parentheses.
func RenderTable(w io.Writer, rows []Row, opts TableOptions) error {
	lines := tableLines(rows, opts.SplitMatch)
	widths := columnWidths(lines)
	var sb strings.Builder
	for _, line := range lines {
		for k := range line.height() {
			writePhysicalLine(&sb, line, k, widths, opts.Color)
		}
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

// tableLine is a table row with one []cell per column, a cell per physical line of the column.
// A nil tableLine is an empty line.
type tableLine [][]cell

// height returns the number of physical lines of the table line, at least 1.
func (l tableLine) height() int {
	h := 1
	for _, col := range l {
		h = max(h, len(col))
	}
	return h
}

// at returns the cell of column i on the physical line k, empty if the column has fewer lines.
func (l tableLine) at(i, k int) cell {
	if k < len(l[i]) {
		return l[i][k]
	}
	return nil
}

// tableLines returns the header and the rows, with an empty line between the hosts.
func tableLines(rows []Row, splitMatch bool) []tableLine {
	lines := make([]tableLine, 0, len(rows)+1)
	header := make(tableLine, 0, len(tableHeader))
	for _, h := range tableHeader {
		header = append(header, []cell{plain(h)})
	}
	lines = append(lines, header)
	for i, r := range rows {
		if i > 0 && r.Host != rows[i-1].Host {
			lines = append(lines, nil) // host separator
		}
		lines = append(lines, rowCells(r, splitMatch))
	}
	return lines
}

func columnWidths(lines []tableLine) []int {
	widths := make([]int, len(tableHeader))
	for _, line := range lines {
		for i, col := range line {
			for _, c := range col {
				widths[i] = max(widths[i], c.width())
			}
		}
	}
	return widths
}

// writePhysicalLine writes the physical line k of the table line, with the columns padded to widths.
func writePhysicalLine(sb *strings.Builder, line tableLine, k int, widths []int, color bool) {
	// The columns after the last non-empty one are skipped to avoid trailing spaces
	last := -1
	for i := range line {
		if line.at(i, k).width() > 0 {
			last = i
		}
	}
	for i := 0; i <= last; i++ {
		c := line.at(i, k)
		sb.WriteString(c.render(color))
		if i < last {
			sb.WriteString(strings.Repeat(" ", widths[i]-c.width()) + columnSeparator)
		}
	}
	sb.WriteString("\n")
}

func rowCells(r Row, splitMatch bool) tableLine {
	prio := cell{{text: strconv.Itoa(r.Priority)}}
	if !r.ExplicitPriority {
		prio = cell{{text: "(" + strconv.Itoa(r.Priority) + ")", color: fmtc.Blue}}
	}
	name := plain(r.Name)
	if r.Marker != NoMarker {
		name = append(name, part{text: " "}, part{text: string(r.Marker), color: markerColors[r.Marker]})
	}
	match := []cell{plain(r.Match)}
	if splitMatch && utf8.RuneCountInString(r.Match) > splitMatchLength {
		clauses := SplitAnd(r.Match)
		match = make([]cell, len(clauses))
		for i, c := range clauses {
			if i > 0 {
				c = matchContinuation + c
			}
			match[i] = plain(c)
		}
	}
	return tableLine{
		match,
		{prio},
		{plain(strings.Join(r.Services, ","))},
		{plain(strings.Join(r.Middlewares, ","))},
		{name},
	}
}

// SplitAnd splits a match rule on its top-level && operators, i.e. not inside parentheses or
// quoted strings, and trims the clauses. A rule without top-level && is returned as is.
func SplitAnd(match string) []string {
	var clauses []string
	depth, start := 0, 0
	var quote byte // the open quote character, 0 outside quoted strings
	for i := 0; i < len(match); i++ {
		ch := match[i]
		switch {
		case quote != 0:
			if ch == '\\' && quote == '"' {
				i++ // skip the escaped character
			} else if ch == quote {
				quote = 0
			}
		case ch == '`' || ch == '"':
			quote = ch
		case ch == '(':
			depth++
		case ch == ')':
			depth--
		case ch == '&' && depth == 0 && strings.HasPrefix(match[i:], "&&"):
			clauses = append(clauses, strings.TrimSpace(match[start:i]))
			i++
			start = i + 1
		}
	}
	if clauses == nil {
		return []string{match}
	}
	return append(clauses, strings.TrimSpace(match[start:]))
}
