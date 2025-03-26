package str

import "fmt"

// -----------------------------------------------------------------------------
// Returns a pointer to the string passed as argument
func Ptr(s string) *string {
	return &s
}

// -----------------------------------------------------------------------------
// Returns a string with green color terminal marker
func Green(s string) string {
	return "\033[32m" + s + "\033[0m"
}

// -----------------------------------------------------------------------------
// Returns a string with red color terminal marker
func Greenf(f string, a ...any) string {
	return "\033[32m" + fmt.Sprintf(f, a...) + "\033[0m"
}

// -----------------------------------------------------------------------------
// Returns a string with red color terminal marker
func Red(s string) string {
	return "\033[31m" + s + "\033[0m"
}

// -----------------------------------------------------------------------------
// Returns a string with red color terminal marker
func Redf(f string, a ...any) string {
	return "\033[31m" + fmt.Sprintf(f, a...) + "\033[0m"
}

// -----------------------------------------------------------------------------
// Returns a string with yellow color terminal marker
func Yellow(s string) string {
	return "\033[33m" + s + "\033[0m"
}

// -----------------------------------------------------------------------------
// Returns a string with red color terminal marker
func Yellowf(f string, a ...any) string {
	return "\033[33m" + fmt.Sprintf(f, a...) + "\033[0m"
}

// -----------------------------------------------------------------------------
// Returns a string with blue color terminal marker
func Blue(s string) string {
	return "\033[34m" + s + "\033[0m"
}

// -----------------------------------------------------------------------------
// Returns a string with red color terminal marker
func Bluef(f string, a ...any) string {
	return "\033[34m" + fmt.Sprintf(f, a...) + "\033[0m"
}
