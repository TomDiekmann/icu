package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tomdiekmann/icu/internal/client"
	"github.com/tomdiekmann/icu/internal/config"
)

var (
	version    string
	cfgOutput  string
	cfgVerbose bool

	cfg *config.Config
	cli *client.Client
)

var rootCmd = &cobra.Command{
	Use:   "icu",
	Short: "A fast, beautiful CLI for Intervals.icu — for humans and AI agents",
	Long: `icu is a dual-mode CLI for Intervals.icu.

In a terminal: rich interactive TUI with tables, charts, and drill-down navigation.
Piped or with --output json: clean structured JSON for scripting and AI agent workflows.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// skip config check for these commands
		skip := map[string]bool{"configure": true, "version": true, "completion": true, "help": true}
		if skip[cmd.Name()] {
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if cfg.APIKey == "" {
			return fmt.Errorf("no API key configured — run `icu configure` to set up")
		}

		athleteID := cfg.AthleteID
		if athleteID == "" {
			athleteID = "0"
		}

		cli = client.New(cfg.APIKey, athleteID, cfgVerbose)
		return nil
	},
}

func Execute(v string) {
	version = v
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		var apiErr *client.APIError
		if errors.As(err, &apiErr) {
			os.Exit(apiErr.ExitCode)
		}
		os.Exit(client.ExitGeneral)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgOutput, "output", "o", "", "output format: auto|table|json|csv")
	rootCmd.PersistentFlags().BoolVarP(&cfgVerbose, "verbose", "v", false, "verbose: log HTTP requests/responses to stderr")

	_ = viper.BindPFlag("default_output", rootCmd.PersistentFlags().Lookup("output"))

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("icu version %s\n", version)
	},
}
