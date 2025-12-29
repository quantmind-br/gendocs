package errors

import (
	"fmt"
)

// ConfigurationError is raised when configuration is invalid or missing
type ConfigurationError struct {
	*AIDocGenError
}

// NewConfigurationError creates a new configuration error
func NewConfigurationError(message string) *ConfigurationError {
	return &ConfigurationError{
		AIDocGenError: &AIDocGenError{
			Message:  message,
			ExitCode: ExitConfigError,
		},
	}
}

// MissingEnvVarError is raised when a required environment variable is not set
type MissingEnvVarError struct {
	*AIDocGenError
}

// NewMissingEnvVarError creates a new missing environment variable error
func NewMissingEnvVarError(varName, description string) *MissingEnvVarError {
	// Convert environment variable name to YAML key format for suggestions
	yamlKey := convertEnvToYAMLKey(varName)

	return &MissingEnvVarError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Required environment variable '%s' is not set", varName),
			Context: &ErrorContext{
				Operation: "Loading configuration",
				Component: "Environment",
				Details: map[string]interface{}{
					"variable":    varName,
					"description": description,
				},
				Suggestions: []string{
					"Run 'gendocs config' to set up configuration interactively",
					fmt.Sprintf("Export the variable: export %s='your-value'", varName),
					fmt.Sprintf("Add to .ai/config.yaml under analyzer.llm.%s", yamlKey),
					"Check .env.example for required variables",
				},
				Recoverable: false,
			},
			ExitCode: ExitConfigError,
		},
	}
}

// convertEnvToYAMLKey converts environment variable name to YAML key format
// Example: ANALYZER_LLM_API_KEY -> api_key
func convertEnvToYAMLKey(envVar string) string {
	parts := splitAndLower(envVar, "_")
	if len(parts) > 2 {
		// Take everything after the first two parts (ANALYZER_LLM)
		return joinParts(parts[2:], "_")
	}
	return joinParts(parts, "_")
}

func splitAndLower(s, sep string) []string {
	parts := make([]string, 0)
	current := ""
	for _, char := range s {
		if string(char) == sep {
			if current != "" {
				parts = append(parts, toLower(current))
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, toLower(current))
	}
	return parts
}

func joinParts(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

func toLower(s string) string {
	result := ""
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}
	return result
}

// InvalidEnvVarError is raised when an environment variable has an invalid value
type InvalidEnvVarError struct {
	*AIDocGenError
}

// NewInvalidEnvVarError creates a new invalid environment variable error
func NewInvalidEnvVarError(varName, value, reason string) *InvalidEnvVarError {
	return &InvalidEnvVarError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Environment variable '%s' has an invalid value", varName),
			Context: &ErrorContext{
				Operation: "Validating configuration",
				Component: "Environment",
				Details: map[string]interface{}{
					"variable": varName,
					"value":    value,
					"reason":   reason,
				},
				Suggestions: []string{
					fmt.Sprintf("Check the value of %s in your .env file", varName),
					"Refer to the documentation for valid values",
				},
				Recoverable: false,
			},
			ExitCode: ExitConfigError,
		},
	}
}

// ConfigFileError is raised when a configuration file cannot be read or parsed
type ConfigFileError struct {
	*AIDocGenError
}

// NewConfigFileError creates a new config file error
func NewConfigFileError(filePath string, cause error) *ConfigFileError {
	return &ConfigFileError{
		AIDocGenError: &AIDocGenError{
			Message: fmt.Sprintf("Failed to load configuration file: %s", filePath),
			Cause:   cause,
			Context: &ErrorContext{
				Operation: "Loading configuration",
				Component: "Config File",
				Details: map[string]interface{}{
					"file_path": filePath,
				},
				Suggestions: []string{
					"Check that the file exists and is readable",
					"Validate YAML syntax",
					"Check file permissions",
				},
				Recoverable: false,
			},
			ExitCode: ExitConfigError,
		},
	}
}
