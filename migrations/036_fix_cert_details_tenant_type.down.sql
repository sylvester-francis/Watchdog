ALTER TABLE cert_details
    ALTER COLUMN tenant_id TYPE UUID USING tenant_id::uuid,
    ALTER COLUMN tenant_id DROP DEFAULT;
