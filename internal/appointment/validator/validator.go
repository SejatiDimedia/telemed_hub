package validator

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/timurdianradhasejati/telemed_hub/internal/appointment/dto"
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

func ValidateCreateAppointment(req dto.CreateAppointmentRequest) (uuid.UUID, uuid.UUID, error) {
	var errs ValidationErrors
	var docID, availID uuid.UUID
	var err error

	if req.DoctorID == "" {
		errs = append(errs, ValidationError{Field: "doctor_id", Issue: "must not be empty"})
	} else {
		docID, err = uuid.Parse(req.DoctorID)
		if err != nil {
			errs = append(errs, ValidationError{Field: "doctor_id", Issue: "invalid UUID format"})
		}
	}

	if req.AvailabilityID == "" {
		errs = append(errs, ValidationError{Field: "availability_id", Issue: "must not be empty"})
	} else {
		availID, err = uuid.Parse(req.AvailabilityID)
		if err != nil {
			errs = append(errs, ValidationError{Field: "availability_id", Issue: "invalid UUID format"})
		}
	}

	if len(errs) > 0 {
		return uuid.Nil, uuid.Nil, errs
	}

	return docID, availID, nil
}

func ValidateRescheduleAppointment(req dto.RescheduleAppointmentRequest) (uuid.UUID, error) {
	var errs ValidationErrors
	var availID uuid.UUID
	var err error

	if req.NewAvailabilityID == "" {
		errs = append(errs, ValidationError{Field: "new_availability_id", Issue: "must not be empty"})
	} else {
		availID, err = uuid.Parse(req.NewAvailabilityID)
		if err != nil {
			errs = append(errs, ValidationError{Field: "new_availability_id", Issue: "invalid UUID format"})
		}
	}

	if len(errs) > 0 {
		return uuid.Nil, errs
	}

	return availID, nil
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
