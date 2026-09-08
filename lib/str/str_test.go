package str_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swissgeo/tool-golang-public/lib/str"
)

func TestPtr(t *testing.T) {
	for _, value := range []string{"", "hello", "Grüezi 🌍", "first\nsecond"} {
		t.Run(value, func(t *testing.T) {
			pointer := str.Ptr(value)
			require.NotNil(t, pointer)
			assert.Equal(t, value, *pointer)
		})
	}
}

func TestPtrReturnsIndependentValues(t *testing.T) {
	value := "original"
	first := str.Ptr(value)
	second := str.Ptr(value)
	*first = "changed"
	assert.Equal(t, "original", value)
	assert.Equal(t, "original", *second)
}

func TestColors(t *testing.T) {
	for _, color := range []struct {
		name   string
		format func(string) string
		prefix string
	}{
		{"green", str.Green, "\x1b[32m"},
		{"red", str.Red, "\x1b[31m"},
		{"yellow", str.Yellow, "\x1b[33m"},
		{"blue", str.Blue, "\x1b[34m"},
	} {
		t.Run(color.name, func(t *testing.T) {
			for _, value := range []string{"", "hello", "Grüezi 🌍", "first\nsecond", "100% %s"} {
				assert.Equal(t, color.prefix+value+"\x1b[0m", color.format(value))
			}
		})
	}
}

func TestFormattedColors(t *testing.T) {
	for _, color := range []struct {
		name   string
		format func(string, ...any) string
		prefix string
	}{
		{"green", str.Greenf, "\x1b[32m"},
		{"red", str.Redf, "\x1b[31m"},
		{"yellow", str.Yellowf, "\x1b[33m"},
		{"blue", str.Bluef, "\x1b[34m"},
	} {
		t.Run(color.name, func(t *testing.T) {
			for _, tt := range []struct {
				format string
				args   []any
				want   string
			}{
				{format: "", want: ""},
				{format: "plain text", want: "plain text"},
				{format: "%s: %d (%t)", args: []any{"tests", 3, true}, want: "tests: 3 (true)"},
				{format: "100%% complete", want: "100% complete"},
				{format: "%s\n%s", args: []any{"Grüezi", "🌍"}, want: "Grüezi\n🌍"},
			} {
				assert.Equal(t, color.prefix+tt.want+"\x1b[0m", color.format(tt.format, tt.args...))
			}
		})
	}
}
