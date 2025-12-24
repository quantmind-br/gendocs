package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/user/gendocs/internal/config"
	"github.com/user/gendocs/internal/errors"
	"github.com/user/gendocs/internal/export"
	"github.com/user/gendocs/internal/handlers"
	"github.com/user/gendocs/internal/logging"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate documentation from analysis results",
	Long:  `Generate documentation files (README.md, AI rules) from existing analysis results.`,
}

var (
	readmeRepoPath  string
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
	// Build configuration
	cfg := config.DocumenterConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: readmeRepoPath,
			Debug:    debugFlag,
		},
		LLM: config.LLMConfig{
			Provider:    os.Getenv("DOCUMENTER_LLM_PROVIDER"),
			Model:       os.Getenv("DOCUMENTER_LLM_MODEL"),
			APIKey:      os.Getenv("DOCUMENTER_LLM_API_KEY"),
			BaseURL:     os.Getenv("DOCUMENTER_LLM_BASE_URL"),
			Retries:     2,
			Timeout:     180,
			MaxTokens:   8192,
			Temperature: 0.0,
		},
	}

	// Set defaults from environment if not set
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = os.Getenv("ANALYZER_LLM_PROVIDER")
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = os.Getenv("ANALYZER_LLM_MODEL")
	}
	if cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = os.Getenv("ANALYZER_LLM_API_KEY")
	}

	// Initialize logger
	logDir := ".ai/logs"
	if readmeRepoPath != "." {
		logDir = readmeRepoPath + "/.ai/logs"
	}
	logCfg := &logging.Config{
		LogDir:       logDir,
		FileLevel:    logging.LevelFromString("info"),
		ConsoleLevel: logging.LevelFromString("debug"),
		EnableCaller: debugFlag,
	}

	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Starting README generation",
		logging.String("repo_path", readmeRepoPath),
	)

	// Create and run ReadmeHandler
	handler := handlers.NewReadmeHandler(cfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		if docErr, ok := err.(*errors.AIDocGenError); ok {
			fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
			return docErr
		}
		return err
	}

	logger.Info("README.md generation complete")

	// Auto-export to HTML if requested
	if autoExportHTML {
		readmePath := filepath.Join(readmeRepoPath, "README.md")
		htmlPath := filepath.Join(readmeRepoPath, "README.html")

		fmt.Println("\nExporting to HTML...")
		if err := exportToHTML(readmePath, htmlPath); err != nil {
			// Don't fail the whole command, just warn
			fmt.Fprintf(os.Stderr, "Warning: HTML export failed: %v\n", err)
		}
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
	// Build configuration
	cfg := config.AIRulesConfig{
		BaseConfig: config.BaseConfig{
			RepoPath: readmeRepoPath,
			Debug:    debugFlag,
		},
		LLM: config.LLMConfig{
			Provider:    os.Getenv("AI_RULES_LLM_PROVIDER"),
			Model:       os.Getenv("AI_RULES_LLM_MODEL"),
			APIKey:      os.Getenv("AI_RULES_LLM_API_KEY"),
			BaseURL:     os.Getenv("AI_RULES_LLM_BASE_URL"),
			Retries:     2,
			Timeout:     240,
			MaxTokens:   8192,
			Temperature: 0.0,
		},
	}

	// Set defaults from environment if not set
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = os.Getenv("ANALYZER_LLM_PROVIDER")
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = os.Getenv("ANALYZER_LLM_MODEL")
	}
	if cfg.LLM.APIKey == "" {
		cfg.LLM.APIKey = os.Getenv("ANALYZER_LLM_API_KEY")
	}

	// Initialize logger
	logDir := ".ai/logs"
	if readmeRepoPath != "." {
		logDir = readmeRepoPath + "/.ai/logs"
	}
	logCfg := &logging.Config{
		LogDir:       logDir,
		FileLevel:    logging.LevelFromString("info"),
		ConsoleLevel: logging.LevelFromString("debug"),
		EnableCaller: debugFlag,
	}

	logger, err := logging.NewLogger(logCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	logger.Info("Starting AI rules generation",
		logging.String("repo_path", readmeRepoPath),
	)

	// Create and run AIRulesHandler
	handler := handlers.NewAIRulesHandler(cfg, logger)

	if err := handler.Handle(cmd.Context()); err != nil {
		if docErr, ok := err.(*errors.AIDocGenError); ok {
			fmt.Fprintf(os.Stderr, "%s\n", docErr.GetUserMessage())
			return docErr
		}
		return err
	}

	logger.Info("AI rules generation complete")
	return nil
}

// exportCmd represents the generate export command
var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export documentation to different formats",
	Long: `Export generated documentation to formats like HTML for easier sharing.

Supported formats:
  - html: Standalone HTML file with embedded CSS and syntax highlighting

Examples:
  # Export README.md to HTML
  gendocs generate export --repo-path . --format html --output docs.html

  # Export specific file
  gendocs generate export --repo-path . --input .ai/docs/code_structure.md --format html

  # Export with default output (README.md → README.html)
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

	exportCmd.Flags().StringVar(&exportFormat, "format", "html", "Export format (html)")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "Output file path (default: input.html)")
	exportCmd.Flags().StringVar(&readmeRepoPath, "repo-path", ".", "Path to repository")
	exportCmd.Flags().StringVar(&exportInput, "input", "README.md", "Input markdown file")
}

func runExport(cmd *cobra.Command, args []string) error {
	// Determine input file path
	inputPath := exportInput
	if !filepath.IsAbs(inputPath) {
		inputPath = filepath.Join(readmeRepoPath, inputPath)
	}

	// Check if input file exists
	if _, err := os.Stat(inputPath); err != nil {
		return fmt.Errorf("input file not found: %s", inputPath)
	}

	// Determine output file path
	outputPath := exportOutput
	if outputPath == "" {
		// Default: replace extension with .html
		ext := filepath.Ext(inputPath)
		outputPath = strings.TrimSuffix(inputPath, ext) + ".html"
	}

	// Ensure output path is absolute
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(readmeRepoPath, outputPath)
	}

	// Export based on format
	switch exportFormat {
	case "html":
		return exportToHTML(inputPath, outputPath)
	default:
		return fmt.Errorf("unsupported format: %s (supported: html)", exportFormat)
	}
}

func exportToHTML(inputPath, outputPath string) error {
	fmt.Printf("Exporting %s to %s...\n", inputPath, outputPath)

	exporter, err := export.NewHTMLExporter()
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	if err := exporter.ExportToHTML(inputPath, outputPath); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	fmt.Printf("✓ HTML exported to %s\n", outputPath)
	return nil
}
