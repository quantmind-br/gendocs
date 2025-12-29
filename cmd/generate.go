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

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate documentation from analysis results",
	Long:  `Generate documentation files (README.md, AI rules) from existing analysis results.`,
}

var (
	readmeRepoPath string
	autoExportHTML bool
)

// readmeCmd represents the generate readme command
var readmeCmd = &cobra.Command{
	Use:   "readme",
	Short: "Generate README.md from analysis results",
	Long: `Generate a comprehensive README.md file based on existing analysis documents
in .ai/docs/. This synthesizes information from structure, dependency, data flow,
request flow, and API analyses into a user-friendly README.`,
	RunE: runReadme,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(readmeCmd)

	readmeCmd.Flags().StringVar(&readmeRepoPath, "repo-path", ".", "Path to repository")
	readmeCmd.Flags().BoolVar(&autoExportHTML, "export-html", false, "Also export to HTML after generation")
}

func runReadme(cmd *cobra.Command, args []string) error {
	cliOverrides := map[string]interface{}{
		"repo_path": readmeRepoPath,
		"debug":     debugFlag,
	}

	cfg, err := config.LoadDocumenterConfig(readmeRepoPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger using helper
	logger, err := InitLogger(readmeRepoPath, debugFlag, verboseFlag)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	showProgress := !verboseFlag
	logger.Info("Starting README generation",
		logging.String("repo_path", readmeRepoPath),
	)

	var progress *tui.SimpleProgress
	if showProgress {
		progress = tui.NewSimpleProgress("Gendocs Generate README")
		progress.Start()
		progress.Step("Loading analysis documents...")
		progress.Info(fmt.Sprintf("Repository: %s", readmeRepoPath))
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

	if autoExportHTML {
		readmePath := filepath.Join(readmeRepoPath, "README.md")
		htmlPath := filepath.Join(readmeRepoPath, "README.html")

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

// aiRulesCmd represents the generate ai-rules command
var aiRulesCmd = &cobra.Command{
	Use:   "ai-rules",
	Short: "Generate AI assistant configuration files",
	Long: `Generate AI assistant configuration files (CLAUDE.md, AGENTS.md, .cursor/rules/)
from existing analysis results. These files help AI coding assistants understand the project.`,
	RunE: runAIRules,
}

func init() {
	generateCmd.AddCommand(aiRulesCmd)
	aiRulesCmd.Flags().StringVar(&readmeRepoPath, "repo-path", ".", "Path to repository")
}

func runAIRules(cmd *cobra.Command, args []string) error {
	cliOverrides := map[string]interface{}{
		"repo_path": readmeRepoPath,
		"debug":     debugFlag,
	}

	cfg, err := config.LoadAIRulesConfig(readmeRepoPath, cliOverrides)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger using helper
	logger, err := InitLogger(readmeRepoPath, debugFlag, verboseFlag)
	if err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	showProgress := !verboseFlag
	logger.Info("Starting AI rules generation",
		logging.String("repo_path", readmeRepoPath),
	)

	var progress *tui.SimpleProgress
	if showProgress {
		progress = tui.NewSimpleProgress("Gendocs Generate AI Rules")
		progress.Start()
		progress.Step("Loading analysis documents...")
		progress.Info(fmt.Sprintf("Repository: %s", readmeRepoPath))
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

// exportCmd represents the generate export command
var exportCmd = &cobra.Command{
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
	RunE: runExport,
}

var (
	exportFormat string
	exportOutput string
	exportInput  string
)

func init() {
	generateCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportFormat, "format", "html", "Export format (html, json)")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: input.html or input.json)")
	exportCmd.Flags().StringVar(&readmeRepoPath, "repo-path", ".", "Path to repository")
	exportCmd.Flags().StringVar(&exportInput, "input", "README.md", "Input markdown file")
}

func runExport(cmd *cobra.Command, args []string) error {
	inputPath := exportInput
	if !filepath.IsAbs(inputPath) {
		inputPath = filepath.Join(readmeRepoPath, inputPath)
	}

	if _, err := os.Stat(inputPath); err != nil {
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	outputPath := exportOutput
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		baseName := strings.TrimSuffix(inputPath, ext)
		switch exportFormat {
		case "json":
			outputPath = baseName + ".json"
		case "html":
			outputPath = baseName + ".html"
		default:
			outputPath = baseName + ".html"
		}
	}

	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(readmeRepoPath, outputPath)
	}

	showProgress := !verboseFlag

	switch exportFormat {
	case "html":
		return exportToHTMLWithProgress(inputPath, outputPath, showProgress)
	case "json":
		return exportToJSONWithProgress(inputPath, outputPath, showProgress)
	default:
		return fmt.Errorf("unsupported format: %s (supported: html, json)", exportFormat)
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
