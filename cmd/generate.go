package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/export"
	"github.com/user/gendocs/internal/handlers"
	"github.com/user/gendocs/internal/logging"
	"github.com/user/gendocs/internal/tui"
)

type readmeOptions struct {
	repoPath       string
	autoExportHTML bool
}

type aiRulesOptions struct {
	repoPath string
}

type exportOptions struct {
	repoPath string
	format   string
	output   string
	input    string
}

func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate documentation from analysis results",
		Long:  `Generate documentation files (README.md, AI rules) from existing analysis results.`,
	}

	cmd.AddCommand(newReadmeCmd())
	cmd.AddCommand(newAIRulesCmd())
	cmd.AddCommand(newExportCmd())

	return cmd
}

func newReadmeCmd() *cobra.Command {
	opts := &readmeOptions{}

	cmd := &cobra.Command{
		Use:   "readme",
		Short: "Generate README.md from analysis results",
		Long: `Generate a comprehensive README.md file based on existing analysis documents
in .ai/docs/. This synthesizes information from structure, dependency, data flow,
request flow, and API analyses into a user-friendly README.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReadme(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.repoPath, "repo-path", ".", "Path to repository")
	cmd.Flags().BoolVar(&opts.autoExportHTML, "export-html", false, "Also export to HTML after generation")

	return cmd
}

func newAIRulesCmd() *cobra.Command {
	opts := &aiRulesOptions{}

	cmd := &cobra.Command{
		Use:   "ai-rules",
		Short: "Generate AI assistant configuration files",
		Long: `Generate AI assistant configuration files (CLAUDE.md, AGENTS.md, .cursor/rules/)
from existing analysis results. These files help AI coding assistants understand the project.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAIRules(cmd, opts)
		},
	}

	cmd.Flags().StringVar(&opts.repoPath, "repo-path", ".", "Path to repository")

	return cmd
}

func newExportCmd() *cobra.Command {
	opts := &exportOptions{}

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export documentation to different formats",
		Long: `Export generated documentation to formats like HTML or JSON for easier sharing.

Supported formats:
  - html: Standalone HTML file with embedded CSS and syntax highlighting
  - json: Structured JSON with metadata and hierarchical content

Examples:
  # Export README.md to HTML
  gendocs generate export --repo-path . --format html --output docs.html

  # Export README.md to JSON
  gendocs generate export --repo-path . --format json --output docs.json

  # Export specific file
  gendocs generate export --repo-path . --input .ai/docs/code_structure.md --format html

  # Export with default output (README.md → README.html or README.json)
  gendocs generate export --repo-path .
`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runExport(opts)
		},
	}

	cmd.Flags().StringVar(&opts.format, "format", "html", "Export format (html, json)")
	cmd.Flags().StringVar(&opts.output, "output", "", "Output file path (default: input.html or input.json)")
	cmd.Flags().StringVar(&opts.repoPath, "repo-path", ".", "Path to repository")
	cmd.Flags().StringVar(&opts.input, "input", "README.md", "Input markdown file")

	return cmd
}

func init() {
	rootCmd.AddCommand(newGenerateCmd())
}

func runReadme(cmd *cobra.Command, opts *readmeOptions) error {
	cliOverrides := map[string]interface{}{
		"repo_path": opts.repoPath,
		"debug":     debugFlag,
	}

	cfg, err := config.LoadDocumenterConfig(opts.repoPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger, err := InitLogger(opts.repoPath, debugFlag, verboseFlag)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	showProgress := !verboseFlag
	logger.Info("Starting README generation",
		logging.String("repo_path", opts.repoPath),
	)

	var progress *tui.SimpleProgress
	if showProgress {
		progress = tui.NewSimpleProgress("Gendocs Generate README")
		progress.Start()
		progress.Step("Loading analysis documents...")
		progress.Info(fmt.Sprintf("Repository: %s", opts.repoPath))
		progress.Step("Generating README.md...")
	}

	handler := handlers.NewReadmeHandler(*cfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		return HandleCommandError(err, progress, showProgress)
	}

	if showProgress {
		progress.Success("README.md generated successfully")
	} else {
		logger.Info("README.md generation complete")
	}

	if opts.autoExportHTML {
		readmePath := filepath.Join(opts.repoPath, "README.md")
		htmlPath := filepath.Join(opts.repoPath, "README.html")

		if showProgress {
			progress.Step("Exporting to HTML...")
		} else {
			fmt.Println("\nExporting to HTML...")
		}

		if err := exportToHTML(readmePath, htmlPath); err != nil {
			if showProgress {
				progress.Warning(fmt.Sprintf("HTML export failed: %v", err))
			} else {
				fmt.Fprintf(os.Stderr, "Warning: HTML export failed: %v\n", err)
			}
		} else if showProgress {
			progress.Success(fmt.Sprintf("Exported to %s", htmlPath))
		}
	}

	if showProgress {
		progress.Done()
	}

	return nil
}

func runAIRules(cmd *cobra.Command, opts *aiRulesOptions) error {
	cliOverrides := map[string]interface{}{
		"repo_path": opts.repoPath,
		"debug":     debugFlag,
	}

	cfg, err := config.LoadAIRulesConfig(opts.repoPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	logger, err := InitLogger(opts.repoPath, debugFlag, verboseFlag)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	showProgress := !verboseFlag
	logger.Info("Starting AI rules generation",
		logging.String("repo_path", opts.repoPath),
	)

	var progress *tui.SimpleProgress
	if showProgress {
		progress = tui.NewSimpleProgress("Gendocs Generate AI Rules")
		progress.Start()
		progress.Step("Loading analysis documents...")
		progress.Info(fmt.Sprintf("Repository: %s", opts.repoPath))
		progress.Step("Generating CLAUDE.md...")
	}

	handler := handlers.NewAIRulesHandler(*cfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		return HandleCommandError(err, progress, showProgress)
	}

	if showProgress {
		progress.Success("CLAUDE.md generated successfully")
		progress.Done()
	} else {
		logger.Info("AI rules generation complete")
	}

	return nil
}

func runExport(opts *exportOptions) error {
	inputPath := opts.input
	if !filepath.IsAbs(inputPath) {
		inputPath = filepath.Join(opts.repoPath, inputPath)
	}

	if _, err := os.Stat(inputPath); err != nil {
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	outputPath := opts.output
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		baseName := strings.TrimSuffix(inputPath, ext)
		switch opts.format {
		case "json":
			outputPath = baseName + ".json"
		case "html":
			outputPath = baseName + ".html"
		default:
			outputPath = baseName + ".html"
		}
	}

	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(opts.repoPath, outputPath)
	}

	showProgress := !verboseFlag

	switch opts.format {
	case "html":
		return exportToHTMLWithProgress(inputPath, outputPath, showProgress)
	case "json":
		return exportToJSONWithProgress(inputPath, outputPath, showProgress)
	default:
		return fmt.Errorf("unsupported format: %s (supported: html, json)", opts.format)
	}
}

func exportToHTML(inputPath, outputPath string) error {
	return exportToHTMLWithProgress(inputPath, outputPath, false)
}

func exportToHTMLWithProgress(inputPath, outputPath string, showProgress bool) error {
	var progress *tui.SimpleProgress
	if showProgress {
		progress = tui.NewSimpleProgress("Gendocs Export")
		progress.Start()
		progress.Step(fmt.Sprintf("Exporting %s...", filepath.Base(inputPath)))
	} else {
		fmt.Printf("Exporting %s to %s...\n", inputPath, outputPath)
	}

	exporter, err := export.NewHTMLExporter()
	if err != nil {
		if showProgress {
			progress.Failed(err)
		}
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	if err := exporter.ExportToHTML(inputPath, outputPath); err != nil {
		if showProgress {
			progress.Failed(err)
		}
		return fmt.Errorf("export failed: %w", err)
	}

	if showProgress {
		progress.Success(fmt.Sprintf("Exported to %s", outputPath))
		progress.Done()
	} else {
		fmt.Printf("✓ HTML exported to %s\n", outputPath)
	}

	return nil
}

func exportToJSONWithProgress(inputPath, outputPath string, showProgress bool) error {
	var progress *tui.SimpleProgress
	if showProgress {
		progress = tui.NewSimpleProgress("Gendocs Export")
		progress.Start()
		progress.Step(fmt.Sprintf("Exporting %s...", filepath.Base(inputPath)))
	} else {
		fmt.Printf("Exporting %s to %s...\n", inputPath, outputPath)
	}

	exporter, err := export.NewJSONExporter()
	if err != nil {
		if showProgress {
			progress.Failed(err)
		}
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	if err := exporter.ExportToJSON(inputPath, outputPath); err != nil {
		if showProgress {
			progress.Failed(err)
		}
		return fmt.Errorf("export failed: %w", err)
	}

	if showProgress {
		progress.Success(fmt.Sprintf("Exported to %s", outputPath))
		progress.Done()
	} else {
		fmt.Printf("✓ JSON exported to %s\n", outputPath)
	}

	return nil
}
