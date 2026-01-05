package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/handlers"
	"github.com/user/gendocs/internal/tui"
)

type checkOptions struct {
	repoPath     string
	outputFormat string
	verbose      bool
	exitCode     bool
}

func newCheckCmd() *cobra.Command {
	opts := &checkOptions{}

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check for documentation drift",
		Long: `Compare the current codebase against the latest analysis in .ai/docs/
and report inconsistencies.

This command detects when your code has changed since the last documentation
analysis, helping you keep documentation synchronized with code changes.

The check command will report:
  - Files that have been added, modified, or deleted
  - Which analysis agents need to be re-run
  - The severity of the documentation drift
  - A recommendation for next steps

Exit codes (when --exit-code is used):
  0: No drift detected, documentation is up to date
  1: Minor drift detected
  2: Moderate drift detected  
  3: Major drift or no previous analysis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheck(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.repoPath, "repo-path", ".", "Path to repository")
	cmd.Flags().StringVarP(&opts.outputFormat, "output", "o", "text", "Output format (text, json)")
	cmd.Flags().BoolVarP(&opts.verbose, "verbose", "V", false, "Show detailed file lists")
	cmd.Flags().BoolVar(&opts.exitCode, "exit-code", false, "Use exit code to indicate drift severity")

	return cmd
}

func init() {
	rootCmd.AddCommand(newCheckCmd())
}

func runCheck(cmd *cobra.Command, opts *checkOptions) error {
	cliOverrides := map[string]interface{}{
		"repo_path":     opts.repoPath,
		"debug":         debugFlag,
		"output_format": opts.outputFormat,
		"verbose":       opts.verbose,
	}

	cfg, err := config.LoadCheckConfig(opts.repoPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger, err := InitLogger(cfg.RepoPath, debugFlag, verboseFlag)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	handler := handlers.NewCheckHandler(*cfg, logger)

	showProgress := opts.outputFormat != "json" && !verboseFlag
	var progress *tui.SimpleProgress
	if showProgress {
		progress = tui.NewSimpleProgress("Gendocs Check")
		progress.Start()
		progress.Info(fmt.Sprintf("Repository: %s", cfg.RepoPath))
		progress.Step("Scanning for changes...")
	}

	report, err := handler.Handle(cmd.Context())
	if err != nil {
		if showProgress {
			progress.Error(err.Error())
		}
		if docErr, ok := err.(*errors.AIDocGenError); ok {
			if !showProgress {
				fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
			}
			return docErr
		}
		return err
	}

	if showProgress {
		progress.Done()
	}

	switch opts.outputFormat {
	case "json":
		output, err := handler.FormatJSONReport(report)
		if err != nil {
			return err
		}
		fmt.Println(output)
	default:
		output := handler.FormatTextReport(report)
		fmt.Print(output)
	}

	if opts.exitCode && report.HasDrift {
		switch report.Severity {
		case handlers.DriftSeverityMinor:
			os.Exit(1)
		case handlers.DriftSeverityModerate:
			os.Exit(2)
		case handlers.DriftSeverityMajor:
			os.Exit(3)
		}
	}

	return nil
}
