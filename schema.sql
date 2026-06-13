CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS team_core (
    team_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_code VARCHAR(128) NOT NULL,
    team_name VARCHAR(256) NOT NULL,
    parent_team_id UUID REFERENCES team_core (team_id),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    source_system VARCHAR(64) NOT NULL DEFAULT 'manual',
    external_id VARCHAR(256),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uk_team_core_code UNIQUE (team_code)
);

CREATE TABLE IF NOT EXISTS person_core (
    person_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_code VARCHAR(128) NOT NULL,
    person_name VARCHAR(256) NOT NULL,
    email VARCHAR(320),
    phone VARCHAR(64),
    team_id UUID REFERENCES team_core (team_id),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    source_system VARCHAR(64) NOT NULL DEFAULT 'manual',
    external_id VARCHAR(256),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uk_person_core_code UNIQUE (person_code),
    CONSTRAINT uk_person_core_email UNIQUE (email)
);

CREATE TABLE IF NOT EXISTS ci_core (
    ci_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ci_type VARCHAR(64) NOT NULL,
    ci_code VARCHAR(128) NOT NULL,
    ci_name VARCHAR(256) NOT NULL,
    display_name VARCHAR(256),
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    lifecycle VARCHAR(32) NOT NULL DEFAULT 'managed',
    environment VARCHAR(16) NOT NULL,
    region VARCHAR(64),
    zone VARCHAR(64),
    owner_team_id UUID REFERENCES team_core (team_id),
    source_system VARCHAR(64) NOT NULL DEFAULT 'manual',
    external_id VARCHAR(256),
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT ck_ci_core_environment CHECK (environment IN ('dev', 'uat', 'pre', 'prod')),
    CONSTRAINT uk_ci_core_code_env UNIQUE (ci_type, ci_code, environment)
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_ci_core_source_external
    ON ci_core (ci_type, source_system, external_id)
    WHERE external_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_ci_core_type_status
    ON ci_core (ci_type, status);

CREATE INDEX IF NOT EXISTS idx_ci_core_owner_team
    ON ci_core (owner_team_id);

CREATE INDEX IF NOT EXISTS idx_ci_core_labels_gin
    ON ci_core USING GIN (labels);

CREATE TABLE IF NOT EXISTS ci_attribute (
    attribute_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ci_id UUID NOT NULL REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    attribute_key VARCHAR(128) NOT NULL,
    attribute_value JSONB NOT NULL,
    value_type VARCHAR(32) NOT NULL DEFAULT 'json',
    source_system VARCHAR(64) NOT NULL DEFAULT 'manual',
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uk_ci_attribute_key UNIQUE (ci_id, attribute_key)
);

CREATE INDEX IF NOT EXISTS idx_ci_attribute_key
    ON ci_attribute (attribute_key);

CREATE INDEX IF NOT EXISTS idx_ci_attribute_value_gin
    ON ci_attribute USING GIN (attribute_value);

CREATE TABLE IF NOT EXISTS ci_change_event (
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    ci_id UUID REFERENCES ci_core (ci_id),
    source_system VARCHAR(64) NOT NULL,
    external_id VARCHAR(256) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    sequence BIGINT,
    resource_version VARCHAR(128),
    idempotency_key VARCHAR(512) NOT NULL,
    payload_hash VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, event_time)
) PARTITION BY RANGE (event_time);

CREATE TABLE IF NOT EXISTS ci_change_event_default
    PARTITION OF ci_change_event DEFAULT;

CREATE INDEX IF NOT EXISTS idx_ci_change_event_ci_time
    ON ci_change_event (ci_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_ci_change_event_source_external
    ON ci_change_event (source_system, external_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_ci_change_event_payload_gin
    ON ci_change_event USING GIN (payload);

CREATE TABLE IF NOT EXISTS ci_event_idempotency (
    idempotency_key VARCHAR(512) PRIMARY KEY,
    event_id UUID NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    source_system VARCHAR(64) NOT NULL,
    external_id VARCHAR(256) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ci_relation_type (
    relation_type_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    relation_code VARCHAR(128) NOT NULL,
    relation_name VARCHAR(256) NOT NULL,
    source_ci_type VARCHAR(64),
    target_ci_type VARCHAR(64),
    direction VARCHAR(32) NOT NULL DEFAULT 'directed',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uk_ci_relation_type_code UNIQUE (relation_code),
    CONSTRAINT ck_ci_relation_type_direction CHECK (direction IN ('directed', 'undirected'))
);

CREATE TABLE IF NOT EXISTS ci_relation (
    relation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_ci_id UUID NOT NULL REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    target_ci_id UUID NOT NULL REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    relation_type_id UUID NOT NULL REFERENCES ci_relation_type (relation_type_id),
    relation_source VARCHAR(64) NOT NULL DEFAULT 'manual',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 1.0000,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    valid_to TIMESTAMPTZ,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    change_event_id UUID,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_ci_relation_confidence CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT ck_ci_relation_valid_time CHECK (valid_to IS NULL OR valid_to > valid_from)
);

CREATE INDEX IF NOT EXISTS idx_ci_relation_source
    ON ci_relation (source_ci_id, relation_type_id, valid_to);

CREATE INDEX IF NOT EXISTS idx_ci_relation_target
    ON ci_relation (target_ci_id, relation_type_id, valid_to);

CREATE INDEX IF NOT EXISTS idx_ci_relation_attributes_gin
    ON ci_relation USING GIN (attributes);

CREATE TABLE IF NOT EXISTS app_person_relation (
    relation_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_ci_id UUID NOT NULL REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    person_id UUID NOT NULL REFERENCES person_core (person_id),
    role VARCHAR(64) NOT NULL,
    environment VARCHAR(16) NOT NULL,
    relation_source VARCHAR(64) NOT NULL DEFAULT 'manual',
    confidence NUMERIC(5, 4) NOT NULL DEFAULT 1.0000,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    valid_to TIMESTAMPTZ,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    change_event_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_app_person_environment CHECK (environment IN ('dev', 'uat', 'pre', 'prod')),
    CONSTRAINT ck_app_person_confidence CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT ck_app_person_valid_time CHECK (valid_to IS NULL OR valid_to > valid_from)
);

CREATE INDEX IF NOT EXISTS idx_app_person_app_role
    ON app_person_relation (app_ci_id, role, valid_to);

CREATE INDEX IF NOT EXISTS idx_app_person_person
    ON app_person_relation (person_id, valid_to);

CREATE TABLE IF NOT EXISTS app_instance_snapshot (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_id UUID NOT NULL REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    instance_ci_id UUID REFERENCES ci_core (ci_id) ON DELETE SET NULL,
    instance_external_id VARCHAR(256) NOT NULL,
    environment VARCHAR(16) NOT NULL,
    region VARCHAR(64),
    zone VARCHAR(64),
    private_ip INET,
    public_ip INET,
    port INTEGER,
    version VARCHAR(128),
    runtime_status VARCHAR(64) NOT NULL,
    resource_version VARCHAR(128),
    last_event_id UUID,
    payload_hash VARCHAR(128),
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_app_instance_environment CHECK (environment IN ('dev', 'uat', 'pre', 'prod')),
    CONSTRAINT uk_app_instance_current UNIQUE (app_id, environment, instance_external_id)
);

CREATE INDEX IF NOT EXISTS idx_app_instance_app_env_status
    ON app_instance_snapshot (app_id, environment, runtime_status);

CREATE INDEX IF NOT EXISTS idx_app_instance_observed
    ON app_instance_snapshot (observed_at DESC);

CREATE INDEX IF NOT EXISTS idx_app_instance_attributes_gin
    ON app_instance_snapshot USING GIN (attributes);

CREATE TABLE IF NOT EXISTS ci_baseline (
    baseline_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ci_id UUID NOT NULL REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    environment VARCHAR(16) NOT NULL,
    baseline_name VARCHAR(256) NOT NULL,
    approved_config JSONB NOT NULL,
    drift_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
    approved_by UUID REFERENCES person_core (person_id),
    approved_at TIMESTAMPTZ,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_ci_baseline_environment CHECK (environment IN ('dev', 'uat', 'pre', 'prod')),
    CONSTRAINT ck_ci_baseline_effective_time CHECK (effective_to IS NULL OR effective_to > effective_from)
);

CREATE INDEX IF NOT EXISTS idx_ci_baseline_ci_env
    ON ci_baseline (ci_id, environment, effective_to);

CREATE INDEX IF NOT EXISTS idx_ci_baseline_config_gin
    ON ci_baseline USING GIN (approved_config);

CREATE TABLE IF NOT EXISTS ci_source_record (
    source_record_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ci_id UUID REFERENCES ci_core (ci_id) ON DELETE SET NULL,
    source_system VARCHAR(64) NOT NULL,
    external_id VARCHAR(256) NOT NULL,
    sync_batch_id VARCHAR(128),
    trust_level INTEGER NOT NULL DEFAULT 50,
    reconcile_status VARCHAR(64) NOT NULL DEFAULT 'pending',
    raw_payload JSONB NOT NULL,
    payload_hash VARCHAR(128) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uk_ci_source_record_external UNIQUE (source_system, external_id),
    CONSTRAINT ck_ci_source_record_trust_level CHECK (trust_level >= 0 AND trust_level <= 100)
);

CREATE INDEX IF NOT EXISTS idx_ci_source_record_ci
    ON ci_source_record (ci_id);

CREATE INDEX IF NOT EXISTS idx_ci_source_record_payload_gin
    ON ci_source_record USING GIN (raw_payload);

CREATE TABLE IF NOT EXISTS app_read_model (
    app_id UUID PRIMARY KEY REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    app_code VARCHAR(128) NOT NULL,
    app_name VARCHAR(256) NOT NULL,
    environment VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL,
    lifecycle VARCHAR(32) NOT NULL,
    owner_team_code VARCHAR(128),
    owner_team_name VARCHAR(256),
    language VARCHAR(64),
    repository_url TEXT,
    instance_count INTEGER NOT NULL DEFAULT 0,
    active_instance_count INTEGER NOT NULL DEFAULT 0,
    cache_version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT ck_app_read_model_environment CHECK (environment IN ('dev', 'uat', 'pre', 'prod'))
);

CREATE INDEX IF NOT EXISTS idx_app_read_model_env_status
    ON app_read_model (environment, status, app_code);

CREATE INDEX IF NOT EXISTS idx_app_read_model_name
    ON app_read_model (app_name);

CREATE TABLE IF NOT EXISTS app_owner_read_model (
    app_id UUID NOT NULL REFERENCES ci_core (ci_id) ON DELETE CASCADE,
    person_id UUID NOT NULL REFERENCES person_core (person_id),
    person_code VARCHAR(128) NOT NULL,
    person_name VARCHAR(256) NOT NULL,
    email VARCHAR(320),
    role VARCHAR(64) NOT NULL,
    relation_source VARCHAR(64) NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ,
    observed_at TIMESTAMPTZ NOT NULL,
    cache_version BIGINT NOT NULL DEFAULT 1,
    PRIMARY KEY (app_id, person_id, role)
);

CREATE INDEX IF NOT EXISTS idx_app_owner_read_model_person
    ON app_owner_read_model (person_id);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_team_core_updated_at ON team_core;

CREATE TRIGGER trg_team_core_updated_at
    BEFORE UPDATE ON team_core
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_person_core_updated_at ON person_core;

CREATE TRIGGER trg_person_core_updated_at
    BEFORE UPDATE ON person_core
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_ci_core_updated_at ON ci_core;

CREATE TRIGGER trg_ci_core_updated_at
    BEFORE UPDATE ON ci_core
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_ci_attribute_updated_at ON ci_attribute;

CREATE TRIGGER trg_ci_attribute_updated_at
    BEFORE UPDATE ON ci_attribute
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_ci_relation_type_updated_at ON ci_relation_type;

CREATE TRIGGER trg_ci_relation_type_updated_at
    BEFORE UPDATE ON ci_relation_type
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_ci_relation_updated_at ON ci_relation;

CREATE TRIGGER trg_ci_relation_updated_at
    BEFORE UPDATE ON ci_relation
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_app_person_relation_updated_at ON app_person_relation;

CREATE TRIGGER trg_app_person_relation_updated_at
    BEFORE UPDATE ON app_person_relation
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_app_instance_snapshot_updated_at ON app_instance_snapshot;

CREATE TRIGGER trg_app_instance_snapshot_updated_at
    BEFORE UPDATE ON app_instance_snapshot
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_ci_baseline_updated_at ON ci_baseline;

CREATE TRIGGER trg_ci_baseline_updated_at
    BEFORE UPDATE ON ci_baseline
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_ci_source_record_updated_at ON ci_source_record;

CREATE TRIGGER trg_ci_source_record_updated_at
    BEFORE UPDATE ON ci_source_record
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO ci_relation_type (relation_code, relation_name, source_ci_type, target_ci_type, direction, description)
VALUES
    ('depends_on', 'Depends On', NULL, NULL, 'directed', 'Source CI depends on target CI.'),
    ('runs_on', 'Runs On', 'application', NULL, 'directed', 'Application or service runs on infrastructure resource.'),
    ('owns', 'Owns', NULL, NULL, 'directed', 'Source CI owns target CI.'),
    ('calls', 'Calls', 'service', 'service', 'directed', 'Source service calls target service.')
ON CONFLICT (relation_code) DO NOTHING;
