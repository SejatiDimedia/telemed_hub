CREATE TABLE IF NOT EXISTS medical_records (
    id UUID PRIMARY KEY,
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE RESTRICT,
    consultation_id UUID NULL REFERENCES consultations(id) ON DELETE SET NULL,
    record_type VARCHAR(50) NOT NULL CHECK (record_type IN ('diagnosis', 'allergy', 'lab_result', 'note')),
    content TEXT NOT NULL,
    file_id UUID NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,
    created_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    deleted_by UUID NULL REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_medical_records_patient_record_type ON medical_records(patient_id, record_type);
CREATE INDEX IF NOT EXISTS idx_medical_records_deleted_at ON medical_records(deleted_at);
