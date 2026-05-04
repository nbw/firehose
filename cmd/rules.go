package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nbw/firehose/internal/client"
	"github.com/spf13/cobra"
)

func newRulesCmd() *cobra.Command {
	rules := &cobra.Command{
		Use:   "rules",
		Short: "Manage rules (requires tap token)",
	}
	rules.AddCommand(
		newRulesListCmd(),
		newRulesCreateCmd(),
		newRulesGetCmd(),
		newRulesUpdateCmd(),
		newRulesDeleteCmd(),
	)
	return rules
}

func readValueArg(value string) (string, error) {
	if value != "-" {
		return value, nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", fmt.Errorf("read --value from stdin: %w", err)
	}
	return strings.TrimRight(string(b), "\r\n"), nil
}

func newRulesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List rules in this tap",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rs, raw, err := apiClient.ListRules(cmd.Context())
			if err != nil {
				return err
			}
			printer.Rules(rs, raw)
			return nil
		},
	}
}

func newRulesCreateCmd() *cobra.Command {
	var (
		value, tag    string
		nsfw, quality bool
	)
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a rule (Lucene query)",
		Args:  cobra.NoArgs,
		Long: `Create a rule. The --value flag takes a Lucene query string.
Pass "--value -" to read the query from stdin (useful for complex queries
that would be awkward to escape on a shell command line).`,
		Example: `  firehose rules create --value 'title:tesla' --tag market-news
  firehose rules create --value 'ahrefs OR semrush' --tag seo
  firehose rules create --value - --tag complex < query.txt`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			v, err := readValueArg(value)
			if err != nil {
				return err
			}
			req := client.RuleCreate{Value: v, Tag: tag}
			if cmd.Flags().Changed("nsfw") {
				req.NSFW = &nsfw
			}
			if cmd.Flags().Changed("quality") {
				req.Quality = &quality
			}
			r, raw, err := apiClient.CreateRule(cmd.Context(), req)
			if err != nil {
				return err
			}
			printer.Rule(*r, raw)
			return nil
		},
	}
	c.Flags().StringVar(&value, "value", "", `Lucene query (required; use "-" to read from stdin)`)
	c.Flags().StringVar(&tag, "tag", "", "optional tag (max 255 chars)")
	c.Flags().BoolVar(&nsfw, "nsfw", false, "include NSFW results")
	c.Flags().BoolVar(&quality, "quality", true, "apply quality filters")
	_ = c.MarkFlagRequired("value")
	return c
}

func newRulesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a rule by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, raw, err := apiClient.GetRule(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printer.Rule(*r, raw)
			return nil
		},
	}
}

func newRulesUpdateCmd() *cobra.Command {
	var (
		value, tag    string
		nsfw, quality bool
	)
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a rule (partial — only changed flags are sent)",
		Args:  cobra.ExactArgs(1),
		Long: `Update a rule. Only flags you explicitly set are sent in the request.

To disable a boolean field, use the explicit form: --nsfw=false / --quality=false.`,
		Example: `  firehose rules update <id> --tag new-tag
  firehose rules update <id> --quality=false
  firehose rules update <id> --value 'updated query'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			upd := client.RuleUpdate{}
			if cmd.Flags().Changed("value") {
				v, err := readValueArg(value)
				if err != nil {
					return err
				}
				upd.Value = &v
			}
			if cmd.Flags().Changed("tag") {
				upd.Tag = &tag
			}
			if cmd.Flags().Changed("nsfw") {
				upd.NSFW = &nsfw
			}
			if cmd.Flags().Changed("quality") {
				upd.Quality = &quality
			}
			r, raw, err := apiClient.UpdateRule(cmd.Context(), args[0], upd)
			if err != nil {
				return err
			}
			printer.Rule(*r, raw)
			return nil
		},
	}
	c.Flags().StringVar(&value, "value", "", `new Lucene query (use "-" for stdin)`)
	c.Flags().StringVar(&tag, "tag", "", "new tag")
	c.Flags().BoolVar(&nsfw, "nsfw", false, "set NSFW flag")
	c.Flags().BoolVar(&quality, "quality", true, "set quality flag")
	return c
}

func newRulesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a rule",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := apiClient.DeleteRule(cmd.Context(), args[0]); err != nil {
				return err
			}
			printer.Deleted("rule", args[0])
			return nil
		},
	}
}
