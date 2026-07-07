package dto

type CreateMedicalRecordRequest struct {
	PatientID      string  `json:"patient_id"`
	ConsultationID *string `json:"consultation_id,omitempty"`
	RecordType     string  `json:"record_type"` // 'diagnosis', 'allergy', 'lab_result', 'note'
	Content        string  `json:"content"`
	FileID         *string `json:"file_id,omitempty"`
}

type UpdateMedicalRecordRequest struct {
	RecordType string  `json:"record_type"`
	Content    string  `json:"content"`
	FileID     *string `json:"file_id,omitempty"`
}

type MedicalRecordResponse struct {
	ID             string  `json:"id"`
	PatientID      string  `json:"patient_id"`
	ConsultationID *string `json:"consultation_id,omitempty"`
	RecordType     string  `json:"record_type"`
	Content        string  `json:"content"`
	FileID         *string `json:"file_id,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type ListMedicalRecordsFilter struct {
	PatientID  *string `json:"patient_id,omitempty"`
	RecordType *string `json:"record_type,omitempty"`
}
