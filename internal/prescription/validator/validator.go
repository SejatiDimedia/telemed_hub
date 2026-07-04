package validator

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/prescription/dto"
)

// ValidationError represents a single field validation failure.
type ValidationError struct {
	Field string
	Issue string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Issue)
}

// ValidationErrors is a slice of ValidationError.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "validation errors"
	}
	return ve[0].Error()
}

// ValidateCreatePrescription validates the prescription issuance request.
func ValidateCreatePrescription(req dto.CreatePrescriptionRequest) (uuid.UUID, error) {
	var errs ValidationErrors

	consultationID, err := uuid.Parse(req.ConsultationID)
	if err != nil {
		errs = append(errs, ValidationError{Field: "consultation_id", Issue: "must be a valid UUID"})
	}

	if len(req.Items) == 0 {
		errs = append(errs, ValidationError{Field: "items", Issue: "must not be empty"})
	}

	for i, item := range req.Items {
		field := fmt.Sprintf("items[%d]", i)

		if _, err := uuid.Parse(item.MedicineID); err != nil {
			errs = append(errs, ValidationError{Field: field + ".medicine_id", Issue: "must be a valid UUID"})
		}
		if item.Dosage == "" {
			errs = append(errs, ValidationError{Field: field + ".dosage", Issue: "must not be empty"})
		}
		if item.Quantity <= 0 {
			errs = append(errs, ValidationError{Field: field + ".quantity", Issue: "must be greater than 0"})
		}
	}

	if len(errs) > 0 {
		return uuid.Nil, errors.New(errs.Error())
	}

	return consultationID, nil
}
