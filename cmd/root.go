package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/nbw/firehose/internal/client"
	"github.com/nbw/firehose/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagMgmtKey  string
	flagTapToken string
	flagBaseURL  string
	flagJSON     bool

	apiClient *client.Client
	printer   output.Printer

	version = "dev"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "firehose",
		Short: "Firehose CLI — manage taps, rules, and consume the SSE stream",
		Long: `firehose is a CLI for the Firehose API (https://api.firehose.com).

Authentication uses two tokens:
  - FIREHOSE_MANAGEMENT_KEY (fhm_…) for tap management
  - FIREHOSE_TAP_TOKEN     (fh_…)  for rules and streaming

For streaming, see examples/stream.sh and examples/stream.go.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			resolveAuth()
			apiClient = client.New(client.Options{
				BaseURL:  flagBaseURL,
				MgmtKey:  flagMgmtKey,
				TapToken: flagTapToken,
				Version:  version,
			})
			printer = output.New(cmd.OutOrStdout(), cmd.ErrOrStderr(), flagJSON)
			return nil
		},
	}

	root.PersistentFlags().StringVar(&flagMgmtKey, "management-key", "", "management key (env FIREHOSE_MANAGEMENT_KEY)")
	root.PersistentFlags().StringVar(&flagTapToken, "tap-token", "", "tap token (env FIREHOSE_TAP_TOKEN)")
	root.PersistentFlags().StringVar(&flagBaseURL, "base-url", "", "API base URL (env FIREHOSE_BASE_URL, default https://api.firehose.com)")
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "emit raw JSON output")

	root.AddCommand(newTapsCmd())
	root.AddCommand(newRulesCmd())
	root.AddCommand(newStreamCmd())
	root.AddCommand(newVersionCmd())
	return root
}

func resolveAuth() {
	if flagMgmtKey == "" {
		flagMgmtKey = os.Getenv("FIREHOSE_MANAGEMENT_KEY")
	}
	if flagTapToken == "" {
		flagTapToken = os.Getenv("FIREHOSE_TAP_TOKEN")
	}
	if flagBaseURL == "" {
		flagBaseURL = os.Getenv("FIREHOSE_BASE_URL")
	}
}

func Execute() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root := newRootCmd()
	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return 130
	}

	renderError(os.Stderr, err)
	return client.ExitCode(err)
}

func renderError(w io.Writer, err error) {
	var apiErr *client.APIError
	var missing *client.ErrMissingAuth
	switch {
	case errors.As(err, &missing):
		fmt.Fprintf(w, "Error: %s\n", missing.Error())
	case errors.As(err, &apiErr):
		if flagJSON && len(apiErr.Body) > 0 {
			w.Write(apiErr.Body)
			if apiErr.Body[len(apiErr.Body)-1] != '\n' {
				w.Write([]byte{'\n'})
			}
			return
		}
		msg := apiErr.Msg
		if msg == "" {
			msg = http_StatusName(apiErr.Status)
		}
		fmt.Fprintf(w, "Error: %d %s\n", apiErr.Status, msg)
		if hint := hintFor(apiErr.Status); hint != "" {
			fmt.Fprintf(w, "Hint:  %s\n", hint)
		}
	default:
		fmt.Fprintf(w, "Error: %s\n", err.Error())
	}
}

func hintFor(status int) string {
	switch status {
	case 401:
		return "check your token (env FIREHOSE_MANAGEMENT_KEY / FIREHOSE_TAP_TOKEN, or --management-key / --tap-token)"
	case 403:
		return "this endpoint may require a different token type — taps need fhm_, rules and stream need fh_"
	case 404:
		return "the resource was not found, or your token doesn't have access"
	case 422:
		return "validation failed — see message; rule limit is 25 per tap"
	case 429:
		return "rate limited — slow down and retry"
	}
	return ""
}

func http_StatusName(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 422:
		return "Unprocessable Entity"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	}
	return "error"
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}
