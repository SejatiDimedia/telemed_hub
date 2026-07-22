package mapper

import (
	"time"

	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/dto"
	"github.com/timurdianradhasejati/telemed_hub/internal/doctor/model"
)

// ToResponse maps a Doctor domain entity to a DoctorResponse DTO.
func ToResponse(doctor *model.Doctor) *dto.DoctorResponse {
	if doctor == nil {
		return nil
	}

	var specialtyResp *dto.SpecialtyResponse
	var specialtyIDStr *string

	if doctor.SpecialtyID != nil {
		idStr := doctor.SpecialtyID.String()
		specialtyIDStr = &idStr
	}

	if doctor.Specialty != nil {
		specialtyResp = &dto.SpecialtyResponse{
			ID:          doctor.Specialty.ID.String(),
			Name:        doctor.Specialty.Name,
			ImageIcon:   doctor.Specialty.ImageIcon,
			Description: doctor.Specialty.Description,
		}
	}

	return &dto.DoctorResponse{
		ID:                   doctor.ID.String(),
		UserID:               doctor.UserID.String(),
		Email:                doctor.Email,
		FullName:             doctor.FullName,
		PhoneNumber:          doctor.PhoneNumber,
		SpecialtyID:          specialtyIDStr,
		Specialty:            specialtyResp,
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

// ToAvailabilityResponse maps an Availability model to AvailabilityResponse DTO.
func ToAvailabilityResponse(avail *model.Availability) *dto.AvailabilityResponse {
	if avail == nil {
		return nil
	}

	return &dto.AvailabilityResponse{
		ID:        avail.ID.String(),
		DoctorID:  avail.DoctorID.String(),
		StartTime: avail.StartTime.Format(time.RFC3339),
		EndTime:   avail.EndTime.Format(time.RFC3339),
		IsBooked:  avail.IsBooked,
	}
}

// ToAvailabilityResponseList maps a slice of Availability models to a slice of AvailabilityResponse DTOs.
func ToAvailabilityResponseList(avails []*model.Availability) []*dto.AvailabilityResponse {
	resp := make([]*dto.AvailabilityResponse, len(avails))
	for i, a := range avails {
		resp[i] = ToAvailabilityResponse(a)
	}
	return resp
}
