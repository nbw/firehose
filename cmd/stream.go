package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"

	"github.com/nbw/firehose/internal/client"
	"github.com/spf13/cobra"
)

func newStreamCmd() *cobra.Command {
	var opts client.StreamOptions
	c := &cobra.Command{
		Use:   "stream",
		Short: "Open the SSE stream and emit events as JSONL",
		Long: `Open the Firehose SSE stream (requires a tap token) and emit one JSON
object per event to stdout. Each line is {"event":"…","id":"…","data":…}.

Output is always JSONL regardless of --json — pretty-printing a live
stream breaks pipeability. Pipe to jq for filtering:

  firehose stream | jq 'select(.event=="update") | .data.document.url'

Press Ctrl-C to stop (exit 130).`,
		Example: `  firehose stream
  firehose stream --timeout 60 --since 5m
  firehose stream --limit 100 --reconnect`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.TimeoutSet = cmd.Flags().Changed("timeout")
			opts.OffsetSet = cmd.Flags().Changed("offset")
			opts.LimitSet = cmd.Flags().Changed("limit")

			out := bufio.NewWriter(cmd.OutOrStdout())
			defer out.Flush()
			enc := json.NewEncoder(out)

			handler := func(ev client.StreamEvent) error {
				record := streamRecord{Event: ev.Event, ID: ev.ID}
				if len(ev.Data) > 0 {
					record.Data = json.RawMessage(ev.Data)
				}
				if err := enc.Encode(record); err != nil {
					return err
				}
				return out.Flush()
			}
			err := apiClient.Stream(cmd.Context(), opts, handler)
			if err != nil {
				return fmt.Errorf("stream: %w", err)
			}
			return nil
		},
	}
	c.Flags().IntVar(&opts.Timeout, "timeout", 300, "connection duration in seconds (1-300)")
	c.Flags().StringVar(&opts.Since, "since", "", "replay window (e.g. 5m, 1h, 24h)")
	c.Flags().Int64Var(&opts.Offset, "offset", 0, "exact Kafka offset to start from")
	c.Flags().IntVar(&opts.Limit, "limit", 0, "close stream after N events (1-10000)")
	c.Flags().StringVar(&opts.ResumeFrom, "resume-from", "", "resume from a Last-Event-ID value")
	c.Flags().BoolVar(&opts.Reconnect, "reconnect", false, "auto-reconnect with exponential backoff")
	return c
}

type streamRecord struct {
	Event string          `json:"event"`
	ID    string          `json:"id,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}
