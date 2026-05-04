package cmd

import (
	"github.com/nbw/firehose/internal/client"
	"github.com/spf13/cobra"
)

func newTapsCmd() *cobra.Command {
	taps := &cobra.Command{
		Use:   "taps",
		Short: "Manage taps (requires management key)",
	}
	taps.AddCommand(
		newTapsListCmd(),
		newTapsCreateCmd(),
		newTapsGetCmd(),
		newTapsUpdateCmd(),
		newTapsDeleteCmd(),
	)
	return taps
}

func newTapsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all taps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ts, raw, err := apiClient.ListTaps(cmd.Context())
			if err != nil {
				return err
			}
			printer.Taps(ts, raw)
			return nil
		},
	}
}

func newTapsCreateCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a new tap",
		Args:  cobra.NoArgs,
		Example: `  firehose taps create --name "Brand Mentions"`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			res, raw, err := apiClient.CreateTap(cmd.Context(), name)
			if err != nil {
				return err
			}
			printer.TapCreated(*res, raw)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "tap name (required)")
	_ = c.MarkFlagRequired("name")
	return c
}

func newTapsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a tap by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, raw, err := apiClient.GetTap(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			printer.Tap(*t, raw)
			return nil
		},
	}
}

func newTapsUpdateCmd() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a tap",
		Args:  cobra.ExactArgs(1),
		Example: `  firehose taps update <id> --name "New Name"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			upd := client.TapUpdate{}
			if cmd.Flags().Changed("name") {
				upd.Name = &name
			}
			t, raw, err := apiClient.UpdateTap(cmd.Context(), args[0], upd)
			if err != nil {
				return err
			}
			printer.Tap(*t, raw)
			return nil
		},
	}
	c.Flags().StringVar(&name, "name", "", "new tap name")
	return c
}

func newTapsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Revoke (delete) a tap",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := apiClient.DeleteTap(cmd.Context(), args[0]); err != nil {
				return err
			}
			printer.Deleted("tap", args[0])
			return nil
		},
	}
}
