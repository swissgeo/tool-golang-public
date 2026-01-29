package log_test

import (
	"testing"

	"github.com/geoadmin/tool-golang-bgdi/lib/log"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestDefineFlags(t *testing.T) {
	flags := pflag.NewFlagSet("foo", pflag.ContinueOnError)
	log.DefineFlags(flags)
	require.NotNil(t, flags.Lookup("log-level"))
	require.NotNil(t, flags.ShorthandLookup("v"))
}

func TestParseFlags(t *testing.T) {
	flags := pflag.NewFlagSet("foo", pflag.ContinueOnError)
	log.DefineFlags(flags)

	for _, l := range []string{"DEBUG", "INFO", "WARN", "ERROR"} {
		require.NoError(t, flags.Set("log-level", l))
		require.NoError(t, log.ParseFlags(*flags))
	}
}
