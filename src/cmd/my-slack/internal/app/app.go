package app

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/end-2/my-cli/src/cmd/my-slack/internal/slack"
	"github.com/end-2/my-cli/src/pkg/cliutil"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	LoadConfig func(slack.Request) (slack.ClientConfig, error)
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
		Use:   "my-slack '<json>'",
		Short: "Call Slack Web API methods for create, read, update, delete, and list flows",
		Long: strings.Join([]string{
			"Call Slack Web API methods with one JSON request.",
			"my-slack supports create, read, update, delete, and list flows with a method-centric request shape.",
			"Provide the JSON as a single argument or pipe it through stdin.",
			"Configure base URL, timeout, user agent, token, and workspace aliases with my-slack.yaml via src/pkg/config.",
			"For list requests, my-slack follows response_metadata.next_cursor automatically until the requested limit is collected.",
		}, "\n"),
		Example: strings.Join([]string{
			`my-slack '{"kind":"create","method":"conversations.create","args":{"name":"eng-bot-playground"}}'`,
			`my-slack '{"kind":"read","method":"conversations.info","args":{"channel":"C12345678"}}'`,
			`my-slack '{"kind":"update","method":"conversations.rename","args":{"channel":"C12345678","name":"eng-platform"}}'`,
			`my-slack '{"kind":"delete","method":"conversations.archive","args":{"channel":"C12345678"}}'`,
			`my-slack '{"kind":"list","method":"conversations.list","limit":50,"args":{"types":"public_channel,private_channel"}}'`,
			`my-slack '{"kind":"list","method":"users.list","limit":200,"alias":"workspace-prod"}'`,
			`echo '{"kind":"create","method":"chat.postMessage","args":{"channel":"C12345678","text":"hello from my-slack"}}' | my-slack`,
			`my-slack --dry-run '{"kind":"list","method":"conversations.history","limit":20,"list_field":"messages","args":{"channel":"C12345678"}}'`,
			`my-slack --version`,
			`my-slack --help`,
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

			request, err := slack.ParseRequest(rawInput)
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
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview the Slack API request without running it")

	return cmd
}

func (e *executor) newClient(request slack.Request) (*slack.Client, error) {
	config, err := e.deps.LoadConfig(request)
	if err != nil {
		return nil, err
	}

	client, err := slack.NewClient(config, e.deps.HTTPClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type dryRunOutput struct {
	Mode    string        `json:"mode"`
	HTTP    dryRunHTTP    `json:"http"`
	Request slack.Request `json:"request"`
}

type dryRunHTTP struct {
	Method      string `json:"method"`
	URL         string `json:"url"`
	Auth        string `json:"auth"`
	ContentType string `json:"content_type,omitempty"`
}
