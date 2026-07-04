CREATE TABLE consultations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    appointment_id UUID NOT NULL UNIQUE REFERENCES appointments(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'scheduled',
    notes TEXT,
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for searching status and appointments
CREATE INDEX idx_consultations_appointment ON consultations(appointment_id);
CREATE INDEX idx_consultations_status ON consultations(status);
