package app

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/end-2/my-cli/src/cmd/my-github/internal/github"
	"github.com/end-2/my-cli/src/pkg/cliutil"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	LoadConfig func(github.Request) (github.ClientConfig, error)
	HTTPClient *http.Client
}

type executor struct {
	stdin   io.Reader
	version string
	deps    Dependencies
}

func Execute(stdin io.Reader, stdout, stderr io.Writer, args []string, version string) error {
	return ExecuteWithDependencies(stdin, stdout, stderr, args, version, Dependencies{})
}

func ExecuteWithDependencies(stdin io.Reader, stdout, stderr io.Writer, args []string, version string, deps Dependencies) error {
	if deps.LoadConfig == nil {
		deps.LoadConfig = LoadConfigForRequest
	}

	cmd := newExecutor(stdin, version, deps).newRootCmd(stdout, stderr)
	cmd.SetArgs(cliutil.NormalizeLongFlags(args, "version", "help", "dry-run"))

	return cmd.Execute()
}

func newExecutor(stdin io.Reader, version string, deps Dependencies) *executor {
	return &executor{
		stdin:   stdin,
		version: version,
		deps:    deps,
	}
}

func (e *executor) newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	var showVersion bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "my-github '<json>'",
		Short: "Query GitHub issues, pull requests, lists, commits, and commit history",
		Long: strings.Join([]string{
			"Query GitHub issues, pull requests, issue lists, pull request lists, commits, and commit history.",
			"my-github queries the GitHub REST API with one JSON request.",
			"Provide the JSON as a single argument or pipe it through stdin.",
			"Configure base URL, per-base-url overrides, timeout, user agent, and token with my-github.yaml via src/pkg/config.",
			"Request JSON may also select a configured instance with base_url or alias.",
		}, "\n"),
		Example: strings.Join([]string{
			`my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123}'`,
			`my-github '{"kind":"issue_list","owner":"cli","repo":"cli","limit":10}'`,
			`my-github '{"kind":"pull_request","owner":"cli","repo":"cli","number":456}'`,
			`my-github '{"kind":"pull_request_list","owner":"cli","repo":"cli","limit":10}'`,
			`my-github '{"kind":"issue","owner":"cli","repo":"cli","number":123,"alias":"example-ghe"}'`,
			`echo '{"kind":"commit","owner":"cli","repo":"cli","ref":"trunk"}' | my-github`,
			`my-github '{"kind":"commit_history","owner":"cli","repo":"cli","ref":"trunk","limit":10}'`,
			`my-github --dry-run '{"kind":"issue","owner":"cli","repo":"cli","number":123}'`,
			`my-github --version`,
			`my-github --help`,
		}, "\n"),
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), e.version)
				return err
			}

			rawInput, err := cliutil.ReadSingleInput(args, e.stdin)
			if err != nil {
				return err
			}

			request, err := github.ParseRequest(rawInput)
			if err != nil {
				return err
			}

			client, err := e.newClient(request)
			if err != nil {
				return err
			}

			plan, err := client.BuildRequest(request)
			if err != nil {
				return err
			}

			if dryRun {
				return cliutil.WriteJSON(cmd.OutOrStdout(), dryRunOutput{
					Mode: "dry-run",
					HTTP: dryRunHTTP{
						Method: "GET",
						URL:    plan.URL.String(),
						Auth:   client.AuthMode(),
					},
					Request: request,
				})
			}

			output, err := client.Execute(plan, request)
			if err != nil {
				return err
			}

			return cliutil.WriteJSON(cmd.OutOrStdout(), output)
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print binary version")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview the GitHub API request without running it")

	return cmd
}

func (e *executor) newClient(request github.Request) (*github.Client, error) {
	config, err := e.deps.LoadConfig(request)
	if err != nil {
		return nil, err
	}

	client, err := github.NewClient(config, e.deps.HTTPClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type dryRunOutput struct {
	Mode    string         `json:"mode"`
	HTTP    dryRunHTTP     `json:"http"`
	Request github.Request `json:"request"`
}

type dryRunHTTP struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Auth   string `json:"auth"`
}
