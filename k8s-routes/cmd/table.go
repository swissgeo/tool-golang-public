package cmd

import (
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

// UseColor reports whether the output to w should be colorized: only if it is a terminal.
func UseColor(w io.Writer) bool {
	f, ok := w.(*os.File)
	return !NoColor && ok && term.IsTerminal(int(f.Fd()))
}

// RenderTable writes the rows as a table with aligned columns, the hosts separated by an empty line.
// The rows are expected to be sorted by host (see BuildRows). Computed priorities are shown in
// parentheses.
func RenderTable(w io.Writer, rows []Row, color bool) error {
	lines := make([][]cell, 0, len(rows)+1)
	header := make([]cell, 0, len(tableHeader))
	for _, h := range tableHeader {
		header = append(header, plain(h))
	}
	lines = append(lines, header)
	for i, r := range rows {
		if i > 0 && r.Host != rows[i-1].Host {
			lines = append(lines, nil) // host separator
		}
		lines = append(lines, rowCells(r))
	}

	widths := make([]int, len(tableHeader))
	for _, line := range lines {
		for i, c := range line {
			widths[i] = max(widths[i], c.width())
		}
	}

	var sb strings.Builder
	for _, line := range lines {
		for i, c := range line {
			sb.WriteString(c.render(color))
			// The last column is not padded to avoid trailing spaces
			if i < len(line)-1 {
				sb.WriteString(strings.Repeat(" ", widths[i]-c.width()) + columnSeparator)
			}
		}
		sb.WriteString("\n")
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

func rowCells(r Row) []cell {
	prio := cell{{text: strconv.Itoa(r.Priority)}}
	if !r.ExplicitPriority {
		prio = cell{{text: "(" + strconv.Itoa(r.Priority) + ")", color: fmtc.Blue}}
	}
	name := plain(r.Name)
	if r.Marker != NoMarker {
		name = append(name, part{text: " "}, part{text: string(r.Marker), color: markerColors[r.Marker]})
	}
	return []cell{
		plain(r.Match),
		prio,
		plain(strings.Join(r.Services, ",")),
		plain(strings.Join(r.Middlewares, ",")),
		name,
	}
}
