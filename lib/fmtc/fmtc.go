package fmtc

import "fmt"

//-----------------------------------------------------------------------------

// Define the Color type
type Color string

// Define color constants using iota
const (
	Red     Color = "\033[31m" // Red color code
	Green   Color = "\033[32m" // Green color code
	Yellow  Color = "\033[33m" // Yellow color code
	Blue    Color = "\033[34m" // Blue color code
	Reset   Color = "\033[0m"  // Reset color
	NoColor Color = ""         // NoColor
)

//-----------------------------------------------------------------------------

func Printf(color Color, format string, a ...any) {
	if color == NoColor {
		fmt.Printf(format, a...)
	} else {
		fmt.Printf(string(color)+format+string(Reset), a...)
	}
}

//-----------------------------------------------------------------------------

func Println(color Color, s ...any) {
	if color == NoColor {
		fmt.Println(s...)
	} else {
		fmt.Println(string(color) + fmt.Sprint(s...) + string(Reset))
	}
}

func Colorise(color Color, s string) string {
	return fmt.Sprintf("%s%s%s", color, s, Reset)
}

//-----------------------------------------------------------------------------

func Sprintf(color Color, format string, a ...any) string {
	if color == NoColor {
		return fmt.Sprintf(format, a...)
	}
	return fmt.Sprintf(string(color)+format+string(Reset), a...)
}
