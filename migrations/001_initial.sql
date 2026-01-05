-- DSPM Initial Schema Migration
-- Run with: psql -d dspm -f migrations/001_initial.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Cloud accounts being monitored
CREATE TABLE IF NOT EXISTS cloud_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider VARCHAR(20) NOT NULL,  -- AWS, AZURE, GCP
    external_id VARCHAR(100) NOT NULL,  -- AWS account ID, subscription ID, project ID
    display_name VARCHAR(255),

    -- Connection configuration
    connector_config JSONB NOT NULL DEFAULT '{}',

    status VARCHAR(20) DEFAULT 'active',  -- active, inactive, error
    status_message TEXT,

    -- Scanning state
    last_scan_at TIMESTAMP,
    last_scan_status VARCHAR(20),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(provider, external_id)
);

CREATE INDEX IF NOT EXISTS idx_accounts_provider ON cloud_accounts(provider);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON cloud_accounts(status);

-- Discovered data assets (buckets, databases, functions, etc.)
CREATE TABLE IF NOT EXISTS data_assets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Resource identification
    resource_type VARCHAR(50) NOT NULL,
    resource_arn VARCHAR(500) NOT NULL,
    region VARCHAR(50),
    name VARCHAR(255) NOT NULL,

    -- Security posture
    encryption_status VARCHAR(20),
    encryption_key_arn VARCHAR(500),
    public_access BOOLEAN DEFAULT false,
    public_access_details JSONB,
    versioning_enabled BOOLEAN,
    logging_enabled BOOLEAN,

    -- Size and usage
    size_bytes BIGINT,
    object_count INTEGER,
    last_accessed_at TIMESTAMP,

    -- Metadata
    tags JSONB DEFAULT '{}',
    raw_metadata JSONB DEFAULT '{}',

    -- Classification summary (denormalized for query performance)
    sensitivity_level VARCHAR(20),
    data_categories TEXT[] DEFAULT '{}',
    classification_count INTEGER DEFAULT 0,

    -- Scanning state
    last_scanned_at TIMESTAMP,
    scan_status VARCHAR(20),
    scan_error TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(resource_arn)
);

CREATE INDEX IF NOT EXISTS idx_assets_account ON data_assets(account_id);
CREATE INDEX IF NOT EXISTS idx_assets_type ON data_assets(resource_type);
CREATE INDEX IF NOT EXISTS idx_assets_sensitivity ON data_assets(sensitivity_level);
CREATE INDEX IF NOT EXISTS idx_assets_public ON data_assets(public_access) WHERE public_access = true;
CREATE INDEX IF NOT EXISTS idx_assets_categories ON data_assets USING GIN(data_categories);

-- Data classification findings
CREATE TABLE IF NOT EXISTS classifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,

    -- Location within asset
    object_path VARCHAR(1000),
    object_size BIGINT,

    -- Classification details
    rule_name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    sensitivity VARCHAR(20) NOT NULL,

    -- Match details
    finding_count INTEGER DEFAULT 1,
    sample_matches JSONB,
    match_locations JSONB,

    -- Confidence
    confidence_score DECIMAL(3,2),
    validated BOOLEAN DEFAULT false,

    discovered_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(asset_id, object_path, rule_name)
);

CREATE INDEX IF NOT EXISTS idx_class_asset ON classifications(asset_id);
CREATE INDEX IF NOT EXISTS idx_class_category ON classifications(category);
CREATE INDEX IF NOT EXISTS idx_class_sensitivity ON classifications(sensitivity);
CREATE INDEX IF NOT EXISTS idx_class_rule ON classifications(rule_name);

-- IAM policies and permissions
CREATE TABLE IF NOT EXISTS access_policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Policy identification
    policy_arn VARCHAR(500) NOT NULL,
    policy_name VARCHAR(255),
    policy_type VARCHAR(50),

    -- Policy content
    policy_document JSONB NOT NULL,
    policy_version VARCHAR(50),

    -- Attachments
    attached_to JSONB DEFAULT '[]',

    -- Analysis results
    allows_public_access BOOLEAN DEFAULT false,
    overly_permissive BOOLEAN DEFAULT false,
    analysis_notes TEXT,

    discovered_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(policy_arn)
);

CREATE INDEX IF NOT EXISTS idx_policies_account ON access_policies(account_id);
CREATE INDEX IF NOT EXISTS idx_policies_public ON access_policies(allows_public_access) WHERE allows_public_access = true;

-- Access relationships between principals and assets
CREATE TABLE IF NOT EXISTS access_edges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Source (who/what has access)
    source_type VARCHAR(50) NOT NULL,
    source_arn VARCHAR(500) NOT NULL,
    source_name VARCHAR(255),

    -- Target (what they have access to)
    target_asset_id UUID REFERENCES data_assets(id) ON DELETE CASCADE,
    target_arn VARCHAR(500),

    -- Access details
    permission_level VARCHAR(20),
    permissions TEXT[],

    -- How access is granted
    policy_id UUID REFERENCES access_policies(id),
    grant_type VARCHAR(50),

    -- Flags
    is_direct BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false,
    is_cross_account BOOLEAN DEFAULT false,

    -- Conditions
    conditions JSONB,

    discovered_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_edges_target ON access_edges(target_asset_id);
CREATE INDEX IF NOT EXISTS idx_edges_source ON access_edges(source_arn);
CREATE INDEX IF NOT EXISTS idx_edges_public ON access_edges(is_public) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_edges_level ON access_edges(permission_level);

-- Security findings and risks
CREATE TABLE IF NOT EXISTS findings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Scope
    account_id UUID REFERENCES cloud_accounts(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES data_assets(id) ON DELETE SET NULL,

    -- Finding details
    finding_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,

    title VARCHAR(500) NOT NULL,
    description TEXT,
    remediation TEXT,

    -- Status
    status VARCHAR(20) DEFAULT 'open',
    status_reason TEXT,
    assigned_to VARCHAR(255),

    -- Compliance
    compliance_frameworks TEXT[],

    -- Evidence
    evidence JSONB,
    resource_snapshot JSONB,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP,
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_seen_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_findings_account ON findings(account_id);
CREATE INDEX IF NOT EXISTS idx_findings_asset ON findings(asset_id);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);
CREATE INDEX IF NOT EXISTS idx_findings_status ON findings(status);
CREATE INDEX IF NOT EXISTS idx_findings_type ON findings(finding_type);

-- Scan job tracking
CREATE TABLE IF NOT EXISTS scan_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Job configuration
    scan_type VARCHAR(50) NOT NULL,
    scan_scope JSONB,

    -- Status
    status VARCHAR(20) DEFAULT 'pending',

    -- Progress
    total_assets INTEGER DEFAULT 0,
    scanned_assets INTEGER DEFAULT 0,

    -- Results summary
    findings_count INTEGER DEFAULT 0,
    classifications_count INTEGER DEFAULT 0,
    errors JSONB DEFAULT '[]',

    -- Timing
    scheduled_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Metadata
    triggered_by VARCHAR(100),
    worker_id VARCHAR(100)
);

CREATE INDEX IF NOT EXISTS idx_jobs_account ON scan_jobs(account_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON scan_jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON scan_jobs(scan_type);

-- Compliance control definitions
CREATE TABLE IF NOT EXISTS compliance_controls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Framework
    framework VARCHAR(50) NOT NULL,
    control_id VARCHAR(50) NOT NULL,
    control_name VARCHAR(255),
    control_description TEXT,

    -- Mapping to DSPM
    finding_types TEXT[],
    data_categories TEXT[],

    -- Requirements
    requirements JSONB,

    UNIQUE(framework, control_id)
);

-- Asset-level compliance status
CREATE TABLE IF NOT EXISTS compliance_status (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    asset_id UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,
    control_id UUID NOT NULL REFERENCES compliance_controls(id),

    -- Status
    status VARCHAR(20) NOT NULL,

    -- Evidence
    evidence JSONB,
    finding_ids UUID[],

    -- Evaluation
    evaluated_at TIMESTAMP DEFAULT NOW(),
    evaluated_by VARCHAR(100),

    notes TEXT,

    UNIQUE(asset_id, control_id)
);

CREATE INDEX IF NOT EXISTS idx_compliance_asset ON compliance_status(asset_id);
CREATE INDEX IF NOT EXISTS idx_compliance_status ON compliance_status(status);

-- Insert default compliance controls
INSERT INTO compliance_controls (framework, control_id, control_name, finding_types, data_categories) VALUES
    ('GDPR', 'Art.32', 'Security of Processing', ARRAY['PUBLIC_BUCKET', 'UNENCRYPTED_STORAGE', 'OVERPRIVILEGED_ACCESS'], ARRAY['PII']),
    ('HIPAA', '164.312(a)(2)(iv)', 'Encryption and Decryption', ARRAY['UNENCRYPTED_STORAGE'], ARRAY['PHI']),
    ('HIPAA', '164.312(c)(1)', 'Integrity Controls', ARRAY['VERSIONING_DISABLED'], ARRAY['PHI']),
    ('PCI-DSS', '3.4', 'Render PAN Unreadable', ARRAY['UNENCRYPTED_STORAGE'], ARRAY['PCI']),
    ('PCI-DSS', '1.3', 'Prohibit Direct Public Access', ARRAY['PUBLIC_BUCKET'], ARRAY['PCI']),
    ('PCI-DSS', '10.2', 'Audit Trail', ARRAY['LOGGING_DISABLED'], ARRAY['PCI']),
    ('SOC2', 'CC6.1', 'Logical Access Controls', ARRAY['PUBLIC_BUCKET', 'OVERPRIVILEGED_ACCESS'], NULL),
    ('SOC2', 'CC7.2', 'System Monitoring', ARRAY['LOGGING_DISABLED'], NULL),
    ('SOC2', 'A1.2', 'Recovery Testing', ARRAY['VERSIONING_DISABLED'], NULL)
ON CONFLICT (framework, control_id) DO NOTHING;
