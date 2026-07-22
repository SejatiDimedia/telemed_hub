-- Re-add specialty string column
ALTER TABLE doctors ADD COLUMN specialty VARCHAR(100);

-- Drop the foreign key and specialty_id column
ALTER TABLE doctors DROP CONSTRAINT IF EXISTS fk_doctors_specialty;
ALTER TABLE doctors DROP COLUMN specialty_id;

-- Re-create the index on the string specialty
CREATE INDEX idx_doctors_specialty ON doctors(specialty) WHERE deleted_at IS NULL;

-- Drop specialties table
DROP TABLE IF EXISTS specialties;
