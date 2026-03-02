-- Fix cert_details.tenant_id type: UUID -> VARCHAR(255) to match all other tables.
-- The CE tenant resolver returns "default" (a string), not a UUID.
ALTER TABLE cert_details
    ALTER COLUMN tenant_id TYPE VARCHAR(255) USING tenant_id::text,
    ALTER COLUMN tenant_id SET DEFAULT 'default';
