package mapper

import (
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/patient/model"
)

// ToResponse maps a Patient domain entity to a PatientResponse DTO.
func ToResponse(patient *model.Patient) *dto.PatientResponse {
	if patient == nil {
		return nil
	}

	var dobStr *string
	if patient.DateOfBirth != nil {
		s := patient.DateOfBirth.Format("2006-01-02")
		dobStr = &s
	}

	return &dto.PatientResponse{
		ID:          patient.ID.String(),
		UserID:      patient.UserID.String(),
		Email:       patient.Email,
		FullName:    patient.FullName,
		PhoneNumber: patient.PhoneNumber,
		DateOfBirth: dobStr,
		Gender:      patient.Gender,
		BloodType:   patient.BloodType,
	}
}
