CREATE TABLE specialties (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    image_icon VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_specialties_name ON specialties(name) WHERE deleted_at IS NULL;

-- Add specialty_id to doctors table
ALTER TABLE doctors ADD COLUMN specialty_id UUID;

-- We can't immediately add NOT NULL to specialty_id if there are existing rows, 
-- but since we are running this with seeding in mind, we'll leave it nullable 
-- during transition, or we assume it's clean. Let's make it reference the specialties table.
ALTER TABLE doctors ADD CONSTRAINT fk_doctors_specialty FOREIGN KEY (specialty_id) REFERENCES specialties(id);

-- Drop the old specialty column
ALTER TABLE doctors DROP COLUMN specialty;
