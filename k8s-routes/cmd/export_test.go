package cmd

// RootCmd exposes the root command to the tests of the cmd_test package.
var RootCmd = rootCmd

// ResetFlags resets the flag variables to their defaults, as cobra doesn't between executions.
func ResetFlags() {
	NoColor = false
	Host = ""
	Root = "."
	Glob = "services/*/overlays/{env}"
	EnableHelm = false
	KubeContext = ""
	WithLocal = false
}
