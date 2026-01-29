package log

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/pflag"
)

var parsedFlags = false

func init() {
	DefineFlags(pflag.CommandLine)
}

func DefineFlags(flags *pflag.FlagSet) {
	flags.StringP("log-level", "v", "INFO", "Minimum log level. E.g. DEBUG, INFO, WARN, ERROR.")
}

func ParseFlags(flags pflag.FlagSet) error {
	strLevel, err := flags.GetString("log-level")
	if err != nil {
		return fmt.Errorf(`unable to determine "log-level" flag value: %w`, err)
	}
	var level slog.Level
	err = level.UnmarshalText([]byte(strLevel))
	if err != nil {
		return fmt.Errorf(`invalid "log-level" flag value: %s %w`, strLevel, err)
	}
	slog.SetLogLoggerLevel(level)
	parsedFlags = true
	return nil
}

func Log(level slog.Level, format string, a ...any) {
	if !parsedFlags {
		_ = ParseFlags(*pflag.CommandLine)
	}
	msg := fmt.Sprintf(format, a...)
	slog.Default().Log(context.Background(), level, msg)
}

func Debug(format string, a ...any) {
	Log(slog.LevelDebug, format, a...)
}

func Info(format string, a ...any) {
	Log(slog.LevelInfo, format, a...)
}

func Warn(format string, a ...any) {
	Log(slog.LevelWarn, format, a...)
}

func Error(format string, a ...any) {
	Log(slog.LevelError, format, a...)
}
