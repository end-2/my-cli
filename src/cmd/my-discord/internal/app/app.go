package app

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/end-2/my-cli/src/cmd/my-discord/internal/discord"
	"github.com/end-2/my-cli/src/pkg/cliutil"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	LoadConfig func(discord.Request) (discord.ClientConfig, error)
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
		Use:   "my-discord '<json>'",
		Short: "Call Discord REST API routes for create, read, update, delete, and list flows",
		Long: strings.Join([]string{
			"Call Discord REST API routes with one JSON request.",
			"my-discord supports create, read, update, delete, and list flows with a route-centric request shape.",
			"Provide the JSON as a single argument or pipe it through stdin.",
			"Configure base URL, timeout, user agent, token, token type, and bot aliases with my-discord.yaml via src/pkg/config.",
			"For list requests, my-discord can keep following before/after pagination using cursor_field.",
		}, "\n"),
		Example: strings.Join([]string{
			`my-discord '{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from my-discord"}}'`,
			`my-discord '{"kind":"read","path":"/channels/123"}'`,
			`my-discord '{"kind":"update","path":"/channels/123","body":{"name":"eng-platform"},"reason":"rename channel"}'`,
			`my-discord '{"kind":"delete","path":"/channels/123/messages/456"}'`,
			`my-discord '{"kind":"list","path":"/channels/123/messages","limit":150,"before":"145000000000000002"}'`,
			`my-discord '{"kind":"list","path":"/guilds/123/members","limit":200,"page_limit":100,"after":"0","cursor_field":"user.id"}'`,
			`my-discord '{"kind":"list","path":"/guilds/123/audit-logs","limit":50,"list_field":"audit_log_entries","query":{"action_type":10}}'`,
			`echo '{"kind":"create","path":"/channels/123/messages","body":{"content":"hello from stdin"}}' | my-discord`,
			`my-discord --dry-run '{"kind":"read","path":"/channels/123","alias":"bot-prod"}'`,
			`my-discord --version`,
			`my-discord --help`,
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

			request, err := discord.ParseRequest(rawInput)
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
						Method:      plan.HTTPMethod,
						URL:         plan.URL.String(),
						Auth:        client.AuthMode(),
						ContentType: plan.ContentType,
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
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview the Discord API request without running it")

	return cmd
}

func (e *executor) newClient(request discord.Request) (*discord.Client, error) {
	config, err := e.deps.LoadConfig(request)
	if err != nil {
		return nil, err
	}

	client, err := discord.NewClient(config, e.deps.HTTPClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type dryRunOutput struct {
	Mode    string          `json:"mode"`
	HTTP    dryRunHTTP      `json:"http"`
	Request discord.Request `json:"request"`
}

type dryRunHTTP struct {
	Method      string `json:"method"`
	URL         string `json:"url"`
	Auth        string `json:"auth"`
	ContentType string `json:"content_type,omitempty"`
}
