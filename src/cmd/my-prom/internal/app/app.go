package app

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/end-2/my-cli/src/cmd/my-prom/internal/prom"
	"github.com/end-2/my-cli/src/pkg/cliutil"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	LoadConfig func(prom.Request) (prom.ClientConfig, error)
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
		Use:   "my-prom '<json>'",
		Short: "Query Prometheus metrics, series, and label metadata with one JSON request",
		Long: strings.Join([]string{
			"Query Prometheus HTTP API endpoints with one JSON request.",
			"my-prom supports instant queries, range queries, series lookup, label name lookup, and label value lookup.",
			"Provide the JSON as a single argument or pipe it through stdin.",
			"Configure base URL, timeout, user agent, token, and instance aliases with my-prom.yaml via src/pkg/config.",
			"Responses are normalized for AI agents with explicit counts, normalized timestamps, warnings, and infos.",
		}, "\n"),
		Example: strings.Join([]string{
			`my-prom '{"kind":"query","query":"up"}'`,
			`my-prom '{"kind":"query_range","query":"rate(http_requests_total[5m])","start":"2026-03-13T00:00:00Z","end":"2026-03-13T01:00:00Z","step":"5m"}'`,
			`my-prom '{"kind":"series","matchers":["up{job=\"node\"}"],"start":"2026-03-13T00:00:00Z","end":"2026-03-13T01:00:00Z"}'`,
			`my-prom '{"kind":"label_values","label":"__name__","limit":100}'`,
			`my-prom '{"kind":"label_names","matchers":["{job=\"api\"}"],"alias":"prod-prom"}'`,
			`echo '{"kind":"query","query":"sum(rate(container_cpu_usage_seconds_total[5m])) by (pod)","http_method":"POST"}' | my-prom`,
			`my-prom --dry-run '{"kind":"query","query":"up","time":"2026-03-13T12:00:00Z"}'`,
			`my-prom --version`,
			`my-prom --help`,
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

			request, err := prom.ParseRequest(rawInput)
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
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "preview the Prometheus API request without running it")

	return cmd
}

func (e *executor) newClient(request prom.Request) (*prom.Client, error) {
	config, err := e.deps.LoadConfig(request)
	if err != nil {
		return nil, err
	}

	client, err := prom.NewClient(config, e.deps.HTTPClient)
	if err != nil {
		return nil, err
	}

	return client, nil
}

type dryRunOutput struct {
	Mode    string       `json:"mode"`
	HTTP    dryRunHTTP   `json:"http"`
	Request prom.Request `json:"request"`
}

type dryRunHTTP struct {
	Method      string `json:"method"`
	URL         string `json:"url"`
	Auth        string `json:"auth"`
	ContentType string `json:"content_type,omitempty"`
}
