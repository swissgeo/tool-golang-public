package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/swissgeo/tool-golang-public/lib/fmtc"
	"github.com/swissgeo/tool-golang-public/lib/version"
)

// Shared flags
var NoColor = false
var Host = ""

// Kustomize rendering flags, used by `local` and `live --local`
var Root = "."
var Glob = "services/*/overlays/{env}"
var EnableHelm = false

// Live flags
var KubeContext = ""
var WithLocal = false

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-routes",
	Short: "List Traefik IngressRoute routes grouped by host and sorted by priority",
	Long: `List Traefik IngressRoute routes grouped by host and sorted by priority
(highest first) within each host, the hosts separated by an empty line. Only
routes of the same host can conflict. A route matching several hosts is listed
under each of them, the routes without Host matcher are listed first, as they
compete with the routes of all hosts.

When no explicit priority is set, Traefik uses the length of the match rule as
priority. Such computed priorities are shown in parentheses.`,
}

var localCmd = &cobra.Command{
	Use:   "local <env>",
	Short: "List the routes rendered from the local kustomize overlays",
	Long: `List the routes of all IngressRoutes rendered by 'kustomize build' of the
overlays matching --glob, where {env} is replaced by the given env.

IngressRoutes without a namespace get the path segment containing the first
wildcard of the glob as namespace (e.g. the service name), like the ArgoCD
application would do.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLocal(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), args[0])
	},
}

var liveCmd = &cobra.Command{
	Use:   "live",
	Short: "List the routes deployed in a cluster (all namespaces)",
	Long: `List the routes of all IngressRoutes deployed in a cluster. Names are shown as
<namespace>/<name>.

With --local, the IngressRoutes rendered from the local kustomize overlays
replace the live ones with the same namespace/name, to preview the effect of
local changes not yet applied. The env (dev|int|prod) is taken from the kube
context name. Such routes are marked:
  (*)  the IngressRoute differs from the live one
  (+)  the IngressRoute does not exist live yet
IngressRoutes removed or renamed locally are still shown with their live version.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runLive(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), version.GetVersion())
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Cancel the running kubectl and kustomize commands on Ctrl-C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := rootCmd.ExecuteContext(ctx)
	stop()
	if err != nil {
		msg := "Error: " + err.Error()
		if UseColor(os.Stderr) {
			msg = fmtc.Colorise(fmtc.Red, msg)
		}
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
}

func addKustomizeFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&Root, "root", Root, "Root directory the glob is relative to.")
	cmd.Flags().StringVar(&Glob, "glob", Glob, "Glob of the kustomize overlays, {env} is replaced by the env.")
	cmd.Flags().BoolVar(&EnableHelm, "enable-helm", false, "Pass --enable-helm to kustomize build.")
}

func init() {
	// Suppress cobra's default error and usage printing so Execute() can
	// print a single, consistently formatted and colourised error line.
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.AddCommand(localCmd, liveCmd, versionCmd)

	rootCmd.PersistentFlags().BoolVar(&NoColor, "no-color", false, "Disable colorized output.")
	rootCmd.PersistentFlags().StringVar(&Host, "host", "", "Only show routes whose host contains this string.")

	addKustomizeFlags(localCmd)

	liveCmd.Flags().StringVar(&KubeContext, "context", "", "Kube context to use (default: current context).")
	liveCmd.Flags().BoolVarP(
		&WithLocal, "local", "l", false, "Overlay the IngressRoutes rendered from the local kustomize overlays.",
	)
	addKustomizeFlags(liveCmd)
}
