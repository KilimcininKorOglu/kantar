-- Add unique constraint on package_dependencies for idempotent upserts
ALTER TABLE package_dependencies
    ADD CONSTRAINT uq_package_dependencies UNIQUE (version_id, dep_name);
