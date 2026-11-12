package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

type setCmdOptions struct {
	profile string
	name    string
	create  bool
	typeStr string
	verbose bool
}

// putParameterInput is passed to the aws cli via --cli-input-json, so that the value never
// appears on the command line (visible in `ps`) and is not interpreted by the aws cli
// (e.g. `file://` prefixes).
type putParameterInput struct {
	Name      string        `json:"Name"`
	Value     string        `json:"Value"`
	Type      string        `json:"Type,omitempty"`
	KeyID     string        `json:"KeyId,omitempty"`
	DataType  string        `json:"DataType,omitempty"`
	Overwrite bool          `json:"Overwrite"`
	Tags      []ssmTagInput `json:"Tags,omitempty"`
}

type ssmTagInput struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// ssmParameterMetadata is the existing parameter metadata (without its value) returned by
// describe-parameters.
type ssmParameterMetadata struct {
	Type     string `json:"Type"`
	KeyID    string `json:"KeyId"`
	DataType string `json:"DataType"`
}

var ssmParameterTypes = []string{"String", "StringList", "SecureString"}

type putParameterOutput struct {
	Version int    `json:"Version"`
	Tier    string `json:"Tier"`
}

func newSetCmd() *cobra.Command {
	options := setCmdOptions{}

	setCmd := &cobra.Command{
		Use:   "set --profile PROFILE --name NAME [--create --type TYPE]",
		Short: "Update/create a single SSM parameter",
		Long: `Update/create the SSM parameter NAME in the account PROFILE.

The value is asked interactively (hidden input), it is never passed on the command line.
The value must be a single line: multi-line values (e.g. PEM certificates or keys) are not
supported, and values longer than about 4 KB are truncated by the terminal.
Without --create the parameter must already exist. With --create the parameter must not
exist yet, --type is mandatory and the tags CreatedBy and Created are added.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			options.verbose, err = cmd.Flags().GetBool("verbose")
			if err != nil {
				return err
			}

			return runSetCmd(options)
		},
	}
	setCmd.Flags().StringVarP(&options.profile, "profile", "p", "", "MANDATORY: Profile/Account name")
	setCmd.Flags().StringVarP(&options.name, "name", "n", "", "MANDATORY: Name of the ssm parameter")
	setCmd.Flags().StringVarP(
		&options.typeStr, "type", "t", "",
		"Type of the ssm parameter (String, StringList or SecureString), mandatory with --create, not allowed without",
	)
	setCmd.Flags().BoolVar(&options.create, "create", false, "WARNING: create the parameter, it must not exist yet")

	nameCompletionCache, err := nameCompletionCached()
	if err != nil {
		panic(err)
	}
	// These calls only fail on programming errors (unknown flag), therefore panic
	for _, e := range []error{
		setCmd.MarkFlagRequired("profile"),
		setCmd.MarkFlagRequired("name"),
		setCmd.RegisterFlagCompletionFunc("profile", profileCompletion),
		setCmd.RegisterFlagCompletionFunc("name", nameCompletionCache),
		setCmd.RegisterFlagCompletionFunc(
			"type", cobra.FixedCompletions(ssmParameterTypes, cobra.ShellCompDirectiveNoFileComp),
		),
	} {
		if e != nil {
			panic(e)
		}
	}
	return setCmd
}

func runSetCmd(opts setCmdOptions) error {
	if opts.create && opts.typeStr == "" {
		return errors.New("--type is mandatory with --create")
	}
	if !opts.create && opts.typeStr != "" {
		return errors.New("--type is only allowed with --create")
	}
	if opts.create && !slices.Contains(ssmParameterTypes, opts.typeStr) {
		return fmt.Errorf("invalid --type %q, must be one of %s", opts.typeStr, strings.Join(ssmParameterTypes, ", "))
	}
	if !term.IsTerminal(stdinFd()) {
		return errors.New("the value must be entered interactively, stdin is not a terminal")
	}

	// Check the existence before asking the value, so that the user does not type it for nothing.
	// put-parameter with overwrite also creates the parameter when it does not exist, therefore
	// this check is mandatory for updates. For creation AWS also refuses to overwrite an
	// existing parameter.
	existing, err := describeSSMParameter(opts.profile, opts.name, opts.verbose)
	if err != nil {
		return err
	}
	if opts.create && existing != nil {
		return fmt.Errorf("parameter %s already exists, remove --create to update it", opts.name)
	}
	if !opts.create && existing == nil {
		return fmt.Errorf("parameter %s does not exist, use --create to create it", opts.name)
	}

	value, err := promptValue(opts.name)
	if err != nil {
		return err
	}

	res, err := setSSMParameter(opts, existing, value)
	if err != nil {
		return err
	}

	fmt.Println(res)
	return nil
}

func stdinFd() int {
	return int(os.Stdin.Fd())
}

// promptValue asks the value interactively twice without echo and checks that both match.
func promptValue(name string) (string, error) {
	fd := stdinFd()

	// term.ReadPassword disables the echo, make sure it is restored when the user hits Ctrl+C
	oldState, err := term.GetState(fd)
	if err != nil {
		return "", err
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-sigCh:
			_ = term.Restore(fd, oldState)
			fmt.Fprintln(os.Stderr)
			os.Exit(130) //nolint:mnd // 128 + SIGINT
		case <-done:
		}
	}()

	read := func(prompt string) (string, error) {
		fmt.Fprint(os.Stderr, prompt)
		b, readErr := term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		return string(b), readErr
	}
	value, err := read(fmt.Sprintf("Value for %s: ", name))
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("the value must not be empty")
	}
	confirm, err := read("Confirm value: ")
	if err != nil {
		return "", err
	}
	if value != confirm {
		return "", errors.New("the values do not match")
	}
	return value, nil
}

// describeSSMParameter returns the metadata of the parameter without decrypting its value, or
// nil if the parameter does not exist.
func describeSSMParameter(profile, name string, verbose bool) (*ssmParameterMetadata, error) {
	cmdArgs := []string{
		awsCLISSMSubCmd, "describe-parameters", //nolint:goconst // aws cli arguments are kept as literals for readability
		"--profile", profile,
		"--parameter-filters", "Key=Name,Option=Equals,Values=" + name,
		"--query", "Parameters[0]",
		"--output", "json",
	}
	stdout, stderr, err := runAWSCmd(context.Background(), cmdArgs, verbose)
	if err != nil {
		return nil, awsCmdError("failed to check if parameter exists", err, stderr)
	}
	// Parameters[0] returns null when the parameter does not exist
	var param *ssmParameterMetadata
	if err = json.Unmarshal([]byte(stdout), &param); err != nil {
		return nil, fmt.Errorf("failed to parse parameter description %q: %w", stdout, err)
	}
	return param, nil
}

// newPutParameterInput returns the put-parameter input. On creation it uses the given type and
// adds the CreatedBy and Created tags, on update it keeps the settings of the existing parameter.
func newPutParameterInput(
	opts setCmdOptions, existing *ssmParameterMetadata, value string,
) (putParameterInput, error) {
	input := putParameterInput{
		Name:      opts.name,
		Value:     value,
		Overwrite: !opts.create,
	}
	if opts.create {
		createdBy, err := currentUsername()
		if err != nil {
			return putParameterInput{}, err
		}
		input.Type = opts.typeStr
		// Tags are only set on creation, existing parameters are tagged with AddTagsToResource
		input.Tags = []ssmTagInput{
			{Key: "CreatedBy", Value: createdBy},
			{Key: "Created", Value: time.Now().Format(time.RFC3339)},
		}
		return input, nil
	}
	if existing != nil {
		// Without KeyId AWS may encrypt the value with the account default key and without
		// DataType it defaults to "text", so keep the existing ones. The type is also sent so
		// that the KeyId is always accompanied by its type and so that, if the parameter has
		// been deleted meanwhile, it is recreated with the same type.
		input.Type = existing.Type
		input.DataType = existing.DataType
		if existing.Type == "SecureString" {
			input.KeyID = existing.KeyID
		}
	}
	return input, nil
}

func setSSMParameter(opts setCmdOptions, existing *ssmParameterMetadata, value string) (string, error) {
	// On SIGINT/SIGTERM/SIGHUP cancel the context instead of exiting, so that the aws cli is killed,
	// the function returns and the temporary file containing the value is removed. This must be
	// set up before the temporary file is created.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer stop()

	input, err := newPutParameterInput(opts, existing, value)
	if err != nil {
		return "", err
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	inputPath, cleanup, err := writeSecretTempFile(inputJSON)
	if err != nil {
		return "", err
	}
	defer cleanup()

	cmdArgs := []string{
		awsCLISSMSubCmd, "put-parameter",
		"--profile", opts.profile,
		"--cli-input-json", "file://" + inputPath,
		"--output", "json",
	}
	if opts.verbose {
		redacted := input
		redacted.Value = "***"
		redactedJSON, _ := json.Marshal(redacted)
		fmt.Printf("cli-input-json: %s\n", redactedJSON)
	}

	stdout, stderr, err := runAWSCmd(ctx, cmdArgs, opts.verbose)
	if err != nil {
		if ctx.Err() != nil {
			return "", errors.New("interrupted, the parameter may or may not have been set")
		}
		if strings.Contains(stderr, "ParameterAlreadyExists") {
			return "", fmt.Errorf("parameter %s already exists, remove --create to update it", opts.name)
		}
		return "", awsCmdError("failed to set parameter", err, stderr)
	}

	// SSM has no "update only if it exists" option. If the parameter has been deleted between
	// the existence check and the update, it has been recreated as a new parameter (version 1)
	// with the same type but without tags (and without description, policies, ...).
	if !opts.create {
		var output putParameterOutput
		if err = json.Unmarshal([]byte(stdout), &output); err != nil {
			return "", fmt.Errorf("failed to parse put-parameter output %q: %w", stdout, err)
		}
		if output.Version == 1 {
			return "", fmt.Errorf(
				"parameter %s has been deleted during the update and has been recreated as a new "+
					"parameter of type %s without tags, please check it", opts.name, existing.Type,
			)
		}
	}
	return stdout, nil
}

// writeSecretTempFile writes data to a temporary file only readable by the current user and
// returns its path and a cleanup function removing it, to be deferred by the caller. The aws cli
// cannot read --cli-input-json from stdin or a named pipe, therefore a file is needed.
func writeSecretTempFile(data []byte) (string, func(), error) {
	f, err := createSecretTempFile()
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.Remove(f.Name()) }
	_, err = f.Write(data)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		cleanup()
		return "", nil, err
	}
	return f.Name(), cleanup, nil
}

// createSecretTempFile creates a temporary file only readable by the current user. The
// $XDG_RUNTIME_DIR is preferred because it is private to the user and usually in memory
// (tmpfs), so the content is not written to disk. If it is not set or not usable (e.g. after
// `su` it may still point to the directory of another user), it falls back to $TMPDIR or /tmp.
func createSecretTempFile() (*os.File, error) {
	const pattern = "aws-ssm-set-*.json"
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		if f, err := os.CreateTemp(dir, pattern); err == nil {
			return f, nil
		}
	}
	return os.CreateTemp("", pattern)
}

// currentUsername returns the name of the user running the command, used for the
// CreatedBy tag. The USER environment variable is often not set in containers, CI or cron.
func currentUsername() (string, error) {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username, nil
	}
	if name := os.Getenv("USER"); name != "" {
		return name, nil
	}
	return "", errors.New("cannot determine the current user for the CreatedBy tag")
}

func init() {
	rootCmd.AddCommand(newSetCmd())
}
