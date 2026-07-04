package validator

import (
	"errors"
	"regexp"
	"strings"

	"github.com/timurdianradhasejati/telemed_hub/internal/auth/dto"
)

var (
	EmailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

type ValidationError struct {
	Field string `json:"field"`
	Issue string `json:"issue"`
}

type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	var sb strings.Builder
	for i, err := range ve {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Field + ": " + err.Issue)
	}
	return sb.String()
}

// ValidateRegister checks registration DTO inputs.
func ValidateRegister(req dto.RegisterRequest) error {
	var errs ValidationErrors

	// FullName
	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		errs = append(errs, ValidationError{Field: "full_name", Issue: "must not be empty"})
	}

	// Email
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		errs = append(errs, ValidationError{Field: "email", Issue: "must not be empty"})
	} else if !EmailRegex.MatchString(req.Email) {
		errs = append(errs, ValidationError{Field: "email", Issue: "must be a valid email address"})
	}

	// Password complexity check: min 8 chars, 1 uppercase, 1 digit, 1 special character
	if req.Password == "" {
		errs = append(errs, ValidationError{Field: "password", Issue: "must not be empty"})
	} else {
		if len(req.Password) < 8 {
			errs = append(errs, ValidationError{Field: "password", Issue: "must be at least 8 characters long"})
		}
		var hasUpper, hasDigit, hasSpecial bool
		for _, c := range req.Password {
			switch {
			case c >= 'A' && c <= 'Z':
				hasUpper = true
			case c >= '0' && c <= '9':
				hasDigit = true
			case (c >= '!' && c <= '/') || (c >= ':' && c <= '@') || (c >= '[' && c <= '`') || (c >= '{' && c <= '~'):
				hasSpecial = true
			}
		}
		if !hasUpper {
			errs = append(errs, ValidationError{Field: "password", Issue: "must contain at least one uppercase letter"})
		}
		if !hasDigit {
			errs = append(errs, ValidationError{Field: "password", Issue: "must contain at least one digit"})
		}
		if !hasSpecial {
			errs = append(errs, ValidationError{Field: "password", Issue: "must contain at least one special character"})
		}
	}

	// Role
	if req.Role != "patient" && req.Role != "doctor" {
		errs = append(errs, ValidationError{Field: "role", Issue: "must be either 'patient' or 'doctor'"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateLogin checks credentials DTO.
func ValidateLogin(req dto.LoginRequest) error {
	var errs ValidationErrors

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" {
		errs = append(errs, ValidationError{Field: "email", Issue: "must not be empty"})
	}

	if req.Password == "" {
		errs = append(errs, ValidationError{Field: "password", Issue: "must not be empty"})
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// ValidateRefresh checks token renewal input.
func ValidateRefresh(req dto.RefreshRequest) error {
	req.RefreshToken = strings.TrimSpace(req.RefreshToken)
	if req.RefreshToken == "" {
		return ValidationErrors{ValidationError{Field: "refresh_token", Issue: "must not be empty"}}
	}
	return nil
}

// ExtractValidationDetails helper to extract standard issue lists
func ExtractValidationDetails(err error) []map[string]string {
	var valErrs ValidationErrors
	if errors.As(err, &valErrs) {
		details := make([]map[string]string, len(valErrs))
		for i, ve := range valErrs {
			details[i] = map[string]string{
				"field": ve.Field,
				"issue": ve.Issue,
			}
		}
		return details
	}
	return []map[string]string{
		{
			"issue": err.Error(),
		},
	}
}
