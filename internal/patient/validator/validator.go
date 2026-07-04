package validator

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
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

func ValidateUpdatePatient(req dto.UpdatePatientRequest) error {
	var errs ValidationErrors

	// Phone Number validation (E.164)
	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	if req.PhoneNumber == "" {
		errs = append(errs, ValidationError{Field: "phone_number", Issue: "must not be empty"})
	} else if !E164Regex.MatchString(req.PhoneNumber) {
		errs = append(errs, ValidationError{Field: "phone_number", Issue: "must be in valid E.164 format (e.g. +6281234567890)"})
	}

	// Date of Birth validation
	req.DateOfBirth = strings.TrimSpace(req.DateOfBirth)
	if req.DateOfBirth == "" {
		errs = append(errs, ValidationError{Field: "date_of_birth", Issue: "must not be empty"})
	} else {
		dob, err := time.Parse("2006-01-02", req.DateOfBirth)
		if err != nil {
			errs = append(errs, ValidationError{Field: "date_of_birth", Issue: "must be in YYYY-MM-DD format"})
		} else if dob.After(time.Now()) {
			errs = append(errs, ValidationError{Field: "date_of_birth", Issue: "must be a date in the past"})
		}
	}

	// Gender validation
	req.Gender = strings.ToLower(strings.TrimSpace(req.Gender))
	if req.Gender == "" {
		errs = append(errs, ValidationError{Field: "gender", Issue: "must not be empty"})
	} else if req.Gender != "male" && req.Gender != "female" {
		errs = append(errs, ValidationError{Field: "gender", Issue: "must be either 'male' or 'female'"})
	}

	// Blood Type validation
	if req.BloodType != nil {
		bt := strings.ToUpper(strings.TrimSpace(*req.BloodType))
		if bt != "" {
			validTypes := map[string]bool{
				"A+": true, "A-": true,
				"B+": true, "B-": true,
				"AB+": true, "AB-": true,
				"O+": true, "O-": true,
			}
			if !validTypes[bt] {
				errs = append(errs, ValidationError{Field: "blood_type", Issue: "must be a valid blood type (A+, A-, B+, B-, AB+, AB-, O+, O-)"})
			}
		}
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
