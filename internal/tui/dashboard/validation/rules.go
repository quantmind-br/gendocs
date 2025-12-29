package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
)

func ValidateNonEmpty(fieldName string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
		return nil
	}
}

func ValidateIntRange(min, max int) func(string) error {
	return func(s string) error {
		if s == "" {
			return nil
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		if v < min || v > max {
			return fmt.Errorf("must be between %d and %d", min, max)
		}
		return nil
	}
}

func ValidateFloatRange(min, max float64) func(string) error {
	return func(s string) error {
		if s == "" {
			return nil
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		if v < min || v > max {
			return fmt.Errorf("must be between %.2f and %.2f", min, max)
		}
		return nil
	}
}

func ValidateURL() func(string) error {
	return func(s string) error {
		if s == "" {
			return nil
		}
		u, err := url.Parse(s)
		if err != nil {
			return fmt.Errorf("invalid URL format")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("URL must start with http:// or https://")
		}
		return nil
	}
}

func ValidateEnum(allowed ...string) func(string) error {
	return func(s string) error {
		if s == "" {
			return nil
		}
		for _, a := range allowed {
			if s == a {
				return nil
			}
		}
		return fmt.Errorf("must be one of: %v", allowed)
	}
}

func ValidatePath() func(string) error {
	validPath := regexp.MustCompile(`^[a-zA-Z0-9_./-]+$`)
	return func(s string) error {
		if s == "" {
			return nil
		}
		if !validPath.MatchString(s) {
			return fmt.Errorf("invalid path characters")
		}
		return nil
	}
}

func ValidatePositiveInt() func(string) error {
	return ValidateIntRange(1, 999999)
}

func ValidateEmail() func(string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return func(s string) error {
		if s == "" {
			return nil
		}
		if !emailRegex.MatchString(s) {
			return fmt.Errorf("invalid email format")
		}
		return nil
	}
}
