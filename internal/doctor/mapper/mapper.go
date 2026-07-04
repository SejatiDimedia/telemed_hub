package mapper

import (
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
)

// ToResponse maps a Doctor domain entity to a DoctorResponse DTO.
func ToResponse(doctor *model.Doctor) *dto.DoctorResponse {
	if doctor == nil {
		return nil
	}

	return &dto.DoctorResponse{
		ID:                   doctor.ID.String(),
		UserID:               doctor.UserID.String(),
		Email:                doctor.Email,
		FullName:             doctor.FullName,
		PhoneNumber:          doctor.PhoneNumber,
		Specialty:            doctor.Specialty,
		LicenseNumber:        doctor.LicenseNumber,
		IsCredentialVerified: doctor.IsCredentialVerified,
		ConsultationFee:      doctor.ConsultationFee,
	}
}

// ToResponseList maps a list of Doctor entities to response DTOs.
func ToResponseList(doctors []*model.Doctor) []*dto.DoctorResponse {
	resp := make([]*dto.DoctorResponse, len(doctors))
	for i, d := range doctors {
		resp[i] = ToResponse(d)
	}
	return resp
}
