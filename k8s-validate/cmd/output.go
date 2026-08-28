package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

var outputMu sync.Mutex

const (
	colorReset = "\033[0m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"
)

func colorize(color, text string) string {
	if NoColor {
		return text
	}
	return color + text + colorReset
}

const defaultSeparatorWidth = 80

func separatorWidth() int {
	if w, _, err := term.GetSize(int(os.Stderr.Fd())); err == nil && w > 0 {
		return w
	}
	return defaultSeparatorWidth
}

// formatDetails strips the current working directory prefix from paths.
func formatDetails(details []byte) []byte {
	s := string(details)
	if cwd, err := os.Getwd(); err == nil {
		s = strings.ReplaceAll(s, cwd+string(os.PathSeparator), "./")
	}
	return []byte(strings.TrimRight(s, "\n") + "\n")
}

func printResult(operation, folder string, ok bool, details []byte) {
	status := colorize(colorGreen, "OK")
	if !ok {
		status = colorize(colorRed, "ERROR")
	}
	outputMu.Lock()
	defer outputMu.Unlock()
	fmt.Printf("Running %-20s %-60s %s\n", operation+" on:", folder+"...", status)
	if len(details) > 0 {
		os.Stderr.Write(formatDetails(details))
		os.Stderr.WriteString(strings.Repeat("-", separatorWidth()) + "\n")
	}
}
