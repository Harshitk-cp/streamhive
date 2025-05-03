package util

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate

	// Regular expression for stream ID
	streamIDRegex = regexp.MustCompile(`^str_[a-zA-Z0-9]+$`)

	// Regular expression for stream key
	streamKeyRegex = regexp.MustCompile(`^sk_[a-zA-Z0-9]+$`)
)

func init() {
	validate = validator.New()

	// Register custom validation tags
	validate.RegisterValidation("streamid", validateStreamID)
	validate.RegisterValidation("streamkey", validateStreamKey)
	validate.RegisterValidation("webhookurl", validateWebhookURL)
}

// Validate validates a struct using the validator
func Validate(s interface{}) error {
	return validate.Struct(s)
}

// ValidateVar validates a variable using the validator
func ValidateVar(field interface{}, tag string) error {
	return validate.Var(field, tag)
}

// validateStreamID validates a stream ID
func validateStreamID(fl validator.FieldLevel) bool {
	return streamIDRegex.MatchString(fl.Field().String())
}

// validateStreamKey validates a stream key
func validateStreamKey(fl validator.FieldLevel) bool {
	return streamKeyRegex.MatchString(fl.Field().String())
}

// validateWebhookURL validates a webhook URL
func validateWebhookURL(fl validator.FieldLevel) bool {
	urlStr := fl.Field().String()

	// Parse URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	// Check host
	if parsedURL.Host == "" {
		return false
	}

	return true
}

// ValidateStreamTitle validates a stream title
func ValidateStreamTitle(title string) error {
	if title == "" {
		return fmt.Errorf("title is required")
	}

	if len(title) > 100 {
		return fmt.Errorf("title must be at most 100 characters")
	}

	return nil
}

// ValidateStreamDescription validates a stream description
func ValidateStreamDescription(description string) error {
	if len(description) > 1000 {
		return fmt.Errorf("description must be at most 1000 characters")
	}

	return nil
}

// ValidateStreamTags validates stream tags
func ValidateStreamTags(tags []string) error {
	if len(tags) > 10 {
		return fmt.Errorf("at most 10 tags are allowed")
	}

	for _, tag := range tags {
		if len(tag) > 50 {
			return fmt.Errorf("tag must be at most 50 characters")
		}

		if strings.ContainsAny(tag, " ,;") {
			return fmt.Errorf("tag must not contain spaces or special characters")
		}
	}

	return nil
}

// ValidateWebhookURL validates a webhook URL
func ValidateWebhookURL(url string) error {
	if err := ValidateVar(url, "required,webhookurl"); err != nil {
		return fmt.Errorf("invalid webhook URL: %v", err)
	}

	return nil
}
