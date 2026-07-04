-- Create medicines table (referenced by prescription_items)
CREATE TABLE medicines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    unit_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    requires_prescription BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_medicines_name ON medicines(name);

-- Create prescriptions table (with soft delete)
CREATE TABLE prescriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consultation_id UUID NOT NULL REFERENCES consultations(id),
    patient_id UUID NOT NULL REFERENCES patients(id),
    doctor_id UUID NOT NULL REFERENCES doctors(id),
    issued_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_by UUID,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID
);

CREATE INDEX idx_prescriptions_consultation ON prescriptions(consultation_id);
CREATE INDEX idx_prescriptions_patient ON prescriptions(patient_id);
CREATE INDEX idx_prescriptions_doctor ON prescriptions(doctor_id);
CREATE INDEX idx_prescriptions_status ON prescriptions(status) WHERE deleted_at IS NULL;

-- Create prescription_items table (no soft delete, structural sub-table)
CREATE TABLE prescription_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prescription_id UUID NOT NULL REFERENCES prescriptions(id) ON DELETE CASCADE,
    medicine_id UUID NOT NULL REFERENCES medicines(id),
    dosage TEXT NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    instructions TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_prescription_items_prescription ON prescription_items(prescription_id);

-- Seed sample medicines for development
INSERT INTO medicines (id, name, unit_price, stock_quantity, requires_prescription) VALUES
    (gen_random_uuid(), 'Amoxicillin 500mg',    15000.00, 500, true),
    (gen_random_uuid(), 'Paracetamol 500mg',     5000.00, 1000, false),
    (gen_random_uuid(), 'Ibuprofen 400mg',        8000.00, 800, false),
    (gen_random_uuid(), 'Omeprazole 20mg',        12000.00, 300, true),
    (gen_random_uuid(), 'Cetirizine 10mg',        7500.00, 600, false);
