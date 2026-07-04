package validator

import (
	"errors"
	"regexp"
	"strings"

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
)

var (
	E164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)
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

func ValidateUpdateDoctor(req dto.UpdateDoctorRequest) error {
	var errs ValidationErrors

	// Phone Number validation (E.164)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	if req.PhoneNumber == "" {
		errs = append(errs, ValidationError{Field: "phone_number", Issue: "must not be empty"})
	} else if !E164Regex.MatchString(req.PhoneNumber) {
		errs = append(errs, ValidationError{Field: "phone_number", Issue: "must be in valid E.164 format (e.g. +6281122334455)"})
	}

	// Specialty validation
	req.Specialty = strings.TrimSpace(req.Specialty)
	if req.Specialty == "" {
		errs = append(errs, ValidationError{Field: "specialty", Issue: "must not be empty"})
	}

	// License Number validation
	req.LicenseNumber = strings.TrimSpace(req.LicenseNumber)
	if req.LicenseNumber == "" {
		errs = append(errs, ValidationError{Field: "license_number", Issue: "must not be empty"})
	}

	// Consultation Fee validation
	if req.ConsultationFee <= 0 {
		errs = append(errs, ValidationError{Field: "consultation_fee", Issue: "must be greater than 0"})
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

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
