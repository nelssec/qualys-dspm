-- Migration: Phase 2 Expansion - Data Lineage, ML Classification, AI Tracking, Encryption Visibility

-- =====================================================
-- SECTION 1: DATA LINEAGE TRACKING
-- =====================================================

-- Lineage events tracking data flow between resources
CREATE TABLE IF NOT EXISTS lineage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Source resource
    source_resource_arn VARCHAR(500) NOT NULL,
    source_resource_type VARCHAR(50) NOT NULL,
    source_resource_name VARCHAR(255),

    -- Target resource
    target_resource_arn VARCHAR(500) NOT NULL,
    target_resource_type VARCHAR(50) NOT NULL,
    target_resource_name VARCHAR(255),

    -- Lineage details
    flow_type VARCHAR(50) NOT NULL,  -- READS_FROM, WRITES_TO, EXPORTS_TO, REPLICATES_TO
    access_method VARCHAR(100),       -- SDK, API, Console, Event-triggered
    frequency VARCHAR(50),            -- CONTINUOUS, SCHEDULED, ON_DEMAND
    data_volume_bytes BIGINT,

    -- Inference metadata
    inferred_from VARCHAR(50),        -- IAM_POLICY, ENV_VARIABLE, EVENT_SOURCE, CLOUDTRAIL
    confidence_score DECIMAL(3,2) DEFAULT 1.0,

    -- Evidence
    evidence JSONB DEFAULT '{}',

    -- Timestamps
    first_observed_at TIMESTAMPTZ DEFAULT NOW(),
    last_observed_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(source_resource_arn, target_resource_arn, flow_type)
);

CREATE INDEX IF NOT EXISTS idx_lineage_account ON lineage_events(account_id);
CREATE INDEX IF NOT EXISTS idx_lineage_source ON lineage_events(source_resource_arn);
CREATE INDEX IF NOT EXISTS idx_lineage_target ON lineage_events(target_resource_arn);
CREATE INDEX IF NOT EXISTS idx_lineage_flow_type ON lineage_events(flow_type);
CREATE INDEX IF NOT EXISTS idx_lineage_inferred ON lineage_events(inferred_from);

-- Computed lineage paths (materialized for query performance)
CREATE TABLE IF NOT EXISTS lineage_paths (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Path endpoints
    origin_arn VARCHAR(500) NOT NULL,
    origin_type VARCHAR(50),
    destination_arn VARCHAR(500) NOT NULL,
    destination_type VARCHAR(50),

    -- Path details
    path_hops INTEGER NOT NULL,
    path_arns TEXT[] NOT NULL,  -- Ordered list of ARNs in path
    flow_types TEXT[] NOT NULL,  -- Flow types at each hop

    -- Risk assessment
    contains_sensitive_data BOOLEAN DEFAULT false,
    sensitivity_level VARCHAR(20),
    data_categories TEXT[],
    risk_score INTEGER DEFAULT 0,

    computed_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lineage_paths_account ON lineage_paths(account_id);
CREATE INDEX IF NOT EXISTS idx_lineage_paths_origin ON lineage_paths(origin_arn);
CREATE INDEX IF NOT EXISTS idx_lineage_paths_destination ON lineage_paths(destination_arn);
CREATE INDEX IF NOT EXISTS idx_lineage_paths_sensitive ON lineage_paths(contains_sensitive_data) WHERE contains_sensitive_data = true;

-- =====================================================
-- SECTION 2: ML-ENHANCED CLASSIFICATION
-- =====================================================

-- ML model registry
CREATE TABLE IF NOT EXISTS ml_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    model_type VARCHAR(50) NOT NULL,  -- NER, DOCUMENT_CLASSIFIER, CONFIDENCE_SCORER
    version VARCHAR(50) NOT NULL,

    -- Model details
    description TEXT,
    framework VARCHAR(50),           -- ONNX, TensorFlow, PyTorch
    model_path VARCHAR(500),         -- S3/GCS path or local path
    config JSONB DEFAULT '{}',

    -- Performance metrics
    accuracy DECIMAL(5,4),
    precision_score DECIMAL(5,4),
    recall_score DECIMAL(5,4),
    f1_score DECIMAL(5,4),

    -- Status
    status VARCHAR(20) DEFAULT 'active',  -- active, training, deprecated
    is_default BOOLEAN DEFAULT false,

    -- Training info
    trained_on_samples INTEGER,
    training_data_version VARCHAR(50),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(name, version)
);

CREATE INDEX IF NOT EXISTS idx_ml_models_type ON ml_models(model_type);
CREATE INDEX IF NOT EXISTS idx_ml_models_status ON ml_models(status);
CREATE INDEX IF NOT EXISTS idx_ml_models_default ON ml_models(is_default) WHERE is_default = true;

-- ML predictions with confidence scores
CREATE TABLE IF NOT EXISTS ml_predictions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    classification_id UUID REFERENCES classifications(id) ON DELETE CASCADE,
    model_id UUID REFERENCES ml_models(id) ON DELETE SET NULL,

    -- Prediction details
    prediction_type VARCHAR(50) NOT NULL,  -- ENTITY, DOCUMENT_TYPE, CONFIDENCE_ADJUSTMENT
    predicted_label VARCHAR(100) NOT NULL,
    confidence_score DECIMAL(5,4) NOT NULL,

    -- For NER predictions
    entity_text VARCHAR(1000),
    entity_start_offset INTEGER,
    entity_end_offset INTEGER,

    -- Raw model output
    raw_output JSONB,

    -- Review status
    review_status VARCHAR(20) DEFAULT 'pending',  -- pending, approved, rejected
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_predictions_classification ON ml_predictions(classification_id);
CREATE INDEX IF NOT EXISTS idx_predictions_model ON ml_predictions(model_id);
CREATE INDEX IF NOT EXISTS idx_predictions_review_status ON ml_predictions(review_status);
CREATE INDEX IF NOT EXISTS idx_predictions_confidence ON ml_predictions(confidence_score);
CREATE INDEX IF NOT EXISTS idx_predictions_type ON ml_predictions(prediction_type);

-- Human review queue for low-confidence predictions
CREATE TABLE IF NOT EXISTS classification_review_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    classification_id UUID NOT NULL REFERENCES classifications(id) ON DELETE CASCADE,
    prediction_id UUID REFERENCES ml_predictions(id) ON DELETE SET NULL,

    -- Review metadata
    priority INTEGER DEFAULT 0,
    reason VARCHAR(100),  -- LOW_CONFIDENCE, CONFLICTING_PREDICTIONS, SENSITIVE_DATA
    original_confidence DECIMAL(5,4),

    -- Assignment
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    assigned_at TIMESTAMPTZ,
    due_by TIMESTAMPTZ,

    -- Resolution
    status VARCHAR(20) DEFAULT 'pending',  -- pending, in_review, resolved, skipped
    resolved_at TIMESTAMPTZ,
    resolution VARCHAR(50),  -- CONFIRMED, REJECTED, MODIFIED
    final_label VARCHAR(100),
    final_confidence DECIMAL(5,4),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_review_queue_status ON classification_review_queue(status);
CREATE INDEX IF NOT EXISTS idx_review_queue_assigned ON classification_review_queue(assigned_to);
CREATE INDEX IF NOT EXISTS idx_review_queue_priority ON classification_review_queue(priority DESC);
CREATE INDEX IF NOT EXISTS idx_review_queue_classification ON classification_review_queue(classification_id);

-- Training feedback for model improvement
CREATE TABLE IF NOT EXISTS training_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID REFERENCES ml_models(id) ON DELETE SET NULL,
    prediction_id UUID REFERENCES ml_predictions(id) ON DELETE SET NULL,

    -- Feedback data
    original_prediction VARCHAR(100),
    corrected_label VARCHAR(100),
    feedback_type VARCHAR(50),  -- CORRECTION, CONFIRMATION, FALSE_POSITIVE, FALSE_NEGATIVE

    -- Sample data (anonymized)
    sample_content TEXT,
    sample_hash VARCHAR(64),  -- SHA-256 hash for deduplication
    context_window TEXT,

    -- Status
    incorporated_in_training BOOLEAN DEFAULT false,
    training_run_id VARCHAR(100),

    submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    submitted_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feedback_model ON training_feedback(model_id);
CREATE INDEX IF NOT EXISTS idx_feedback_incorporated ON training_feedback(incorporated_in_training);
CREATE INDEX IF NOT EXISTS idx_feedback_type ON training_feedback(feedback_type);

-- =====================================================
-- SECTION 3: AI SOURCE TRACKING
-- =====================================================

-- AI/ML service registry
CREATE TABLE IF NOT EXISTS ai_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Service identification
    service_type VARCHAR(50) NOT NULL,  -- SAGEMAKER, BEDROCK, VERTEX_AI, AZURE_ML
    service_arn VARCHAR(500) NOT NULL,
    service_name VARCHAR(255),

    -- Service details
    region VARCHAR(50),
    service_config JSONB DEFAULT '{}',

    -- Status
    status VARCHAR(20) DEFAULT 'active',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(service_arn)
);

CREATE INDEX IF NOT EXISTS idx_ai_services_account ON ai_services(account_id);
CREATE INDEX IF NOT EXISTS idx_ai_services_type ON ai_services(service_type);
CREATE INDEX IF NOT EXISTS idx_ai_services_status ON ai_services(status);

-- AI model inventory
CREATE TABLE IF NOT EXISTS ai_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID REFERENCES ai_services(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Model identification
    model_arn VARCHAR(500) NOT NULL,
    model_name VARCHAR(255),
    model_type VARCHAR(50),  -- TRAINING, INFERENCE, FOUNDATION

    -- Model details
    framework VARCHAR(50),
    version VARCHAR(50),
    description TEXT,

    -- Status
    status VARCHAR(20),  -- CREATING, READY, DELETING, FAILED
    endpoint_arn VARCHAR(500),

    -- Metadata
    tags JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(model_arn)
);

CREATE INDEX IF NOT EXISTS idx_ai_models_service ON ai_models(service_id);
CREATE INDEX IF NOT EXISTS idx_ai_models_account ON ai_models(account_id);
CREATE INDEX IF NOT EXISTS idx_ai_models_type ON ai_models(model_type);
CREATE INDEX IF NOT EXISTS idx_ai_models_status ON ai_models(status);

-- Training data lineage (which datasets trained which models)
CREATE TABLE IF NOT EXISTS ai_training_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id UUID NOT NULL REFERENCES ai_models(id) ON DELETE CASCADE,

    -- Data source
    data_source_arn VARCHAR(500) NOT NULL,  -- S3 bucket, database, etc.
    data_source_type VARCHAR(50) NOT NULL,

    -- Data details
    data_path VARCHAR(1000),
    data_format VARCHAR(50),
    sample_count BIGINT,
    data_size_bytes BIGINT,

    -- Sensitivity tracking
    contains_sensitive_data BOOLEAN DEFAULT false,
    sensitivity_categories TEXT[],
    sensitivity_level VARCHAR(20),

    -- Timestamps
    used_at TIMESTAMPTZ DEFAULT NOW(),
    discovered_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_training_data_model ON ai_training_data(model_id);
CREATE INDEX IF NOT EXISTS idx_training_data_source ON ai_training_data(data_source_arn);
CREATE INDEX IF NOT EXISTS idx_training_data_sensitive ON ai_training_data(contains_sensitive_data) WHERE contains_sensitive_data = true;

-- AI data access events (inference access to sensitive data)
CREATE TABLE IF NOT EXISTS ai_processing_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- AI service details
    service_id UUID REFERENCES ai_services(id) ON DELETE SET NULL,
    model_id UUID REFERENCES ai_models(id) ON DELETE SET NULL,

    -- Access details
    event_type VARCHAR(50) NOT NULL,  -- TRAINING_JOB, INFERENCE_REQUEST, DATA_FETCH, BATCH_TRANSFORM
    event_time TIMESTAMPTZ NOT NULL,

    -- Data accessed
    data_source_arn VARCHAR(500),
    data_asset_id UUID REFERENCES data_assets(id) ON DELETE SET NULL,

    -- Data sensitivity
    accessed_sensitivity_level VARCHAR(20),
    accessed_categories TEXT[],
    data_volume_bytes BIGINT,
    record_count INTEGER,

    -- Principal info
    principal_arn VARCHAR(500),
    principal_type VARCHAR(50),

    -- Event details
    event_details JSONB DEFAULT '{}',

    -- Risk assessment
    risk_score INTEGER DEFAULT 0,
    risk_factors TEXT[],

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_events_account ON ai_processing_events(account_id);
CREATE INDEX IF NOT EXISTS idx_ai_events_service ON ai_processing_events(service_id);
CREATE INDEX IF NOT EXISTS idx_ai_events_model ON ai_processing_events(model_id);
CREATE INDEX IF NOT EXISTS idx_ai_events_time ON ai_processing_events(event_time);
CREATE INDEX IF NOT EXISTS idx_ai_events_sensitivity ON ai_processing_events(accessed_sensitivity_level);
CREATE INDEX IF NOT EXISTS idx_ai_events_type ON ai_processing_events(event_type);

-- =====================================================
-- SECTION 4: ENHANCED ENCRYPTION VISIBILITY
-- =====================================================

-- KMS keys registry
CREATE TABLE IF NOT EXISTS encryption_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Key identification
    key_id VARCHAR(100) NOT NULL,
    key_arn VARCHAR(500) NOT NULL,
    alias VARCHAR(255),
    description TEXT,

    -- Key details
    key_type VARCHAR(50),           -- SYMMETRIC, ASYMMETRIC
    key_usage VARCHAR(50),          -- ENCRYPT_DECRYPT, SIGN_VERIFY
    key_spec VARCHAR(50),           -- SYMMETRIC_DEFAULT, RSA_2048, etc.
    key_manager VARCHAR(50),        -- AWS, CUSTOMER
    origin VARCHAR(50),             -- AWS_KMS, EXTERNAL, AWS_CLOUDHSM

    -- Key state
    key_state VARCHAR(50) NOT NULL, -- Enabled, Disabled, PendingDeletion, etc.
    enabled BOOLEAN DEFAULT true,

    -- Rotation
    rotation_enabled BOOLEAN DEFAULT false,
    last_rotated_at TIMESTAMPTZ,
    next_rotation_at TIMESTAMPTZ,
    rotation_period_days INTEGER,

    -- Deletion
    deletion_date TIMESTAMPTZ,
    pending_deletion_days INTEGER,

    -- Policy
    key_policy JSONB,
    allows_public_access BOOLEAN DEFAULT false,
    allows_cross_account BOOLEAN DEFAULT false,
    cross_account_principals TEXT[],

    -- Metadata
    tags JSONB DEFAULT '{}',
    region VARCHAR(50),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    discovered_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(key_arn)
);

CREATE INDEX IF NOT EXISTS idx_encryption_keys_account ON encryption_keys(account_id);
CREATE INDEX IF NOT EXISTS idx_encryption_keys_state ON encryption_keys(key_state);
CREATE INDEX IF NOT EXISTS idx_encryption_keys_rotation ON encryption_keys(rotation_enabled);
CREATE INDEX IF NOT EXISTS idx_encryption_keys_type ON encryption_keys(key_type);

-- Key usage tracking (which assets use which keys)
CREATE TABLE IF NOT EXISTS encryption_key_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id UUID NOT NULL REFERENCES encryption_keys(id) ON DELETE CASCADE,

    -- Asset using the key
    asset_id UUID REFERENCES data_assets(id) ON DELETE SET NULL,
    asset_arn VARCHAR(500) NOT NULL,
    asset_type VARCHAR(50) NOT NULL,

    -- Usage details
    usage_type VARCHAR(50) NOT NULL,  -- BUCKET_ENCRYPTION, EBS_ENCRYPTION, RDS_ENCRYPTION, LAMBDA_ENV
    encryption_context JSONB,

    -- Timestamps
    first_seen_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_key_usage_key ON encryption_key_usage(key_id);
CREATE INDEX IF NOT EXISTS idx_key_usage_asset ON encryption_key_usage(asset_id);
CREATE INDEX IF NOT EXISTS idx_key_usage_type ON encryption_key_usage(usage_type);

-- In-transit encryption settings
CREATE TABLE IF NOT EXISTS transit_encryption (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,

    -- Endpoint details
    endpoint_type VARCHAR(50) NOT NULL,  -- API, WEBSITE, DATABASE, LOAD_BALANCER
    endpoint_url VARCHAR(500),

    -- TLS configuration
    tls_enabled BOOLEAN DEFAULT false,
    tls_version VARCHAR(20),             -- TLSv1.2, TLSv1.3
    min_tls_version VARCHAR(20),
    certificate_arn VARCHAR(500),
    certificate_expiry TIMESTAMPTZ,

    -- Cipher suites
    cipher_suites TEXT[],
    supports_perfect_forward_secrecy BOOLEAN,

    -- Compliance
    meets_minimum_standards BOOLEAN DEFAULT false,
    compliance_issues TEXT[],

    last_checked_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transit_asset ON transit_encryption(asset_id);
CREATE INDEX IF NOT EXISTS idx_transit_tls ON transit_encryption(tls_enabled);
CREATE INDEX IF NOT EXISTS idx_transit_compliance ON transit_encryption(meets_minimum_standards);

-- Encryption compliance scores
CREATE TABLE IF NOT EXISTS encryption_compliance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID REFERENCES cloud_accounts(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES data_assets(id) ON DELETE CASCADE,

    -- Overall score
    compliance_score INTEGER NOT NULL,  -- 0-100
    grade VARCHAR(5),                    -- A, B, C, D, F

    -- Score breakdown
    at_rest_score INTEGER,
    in_transit_score INTEGER,
    key_management_score INTEGER,

    -- Findings
    findings_count INTEGER DEFAULT 0,
    critical_findings INTEGER DEFAULT 0,

    -- Details
    compliance_details JSONB DEFAULT '{}',
    recommendations TEXT[],

    evaluated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Only one score per asset at a time
    UNIQUE(asset_id)
);

CREATE INDEX IF NOT EXISTS idx_compliance_account ON encryption_compliance(account_id);
CREATE INDEX IF NOT EXISTS idx_compliance_asset ON encryption_compliance(asset_id);
CREATE INDEX IF NOT EXISTS idx_compliance_score ON encryption_compliance(compliance_score);
CREATE INDEX IF NOT EXISTS idx_compliance_grade ON encryption_compliance(grade);

-- =====================================================
-- SECTION 5: CONFIGURATION THRESHOLDS
-- =====================================================

-- ML confidence thresholds configuration
CREATE TABLE IF NOT EXISTS ml_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default ML configuration
INSERT INTO ml_config (config_key, config_value, description) VALUES
    ('confidence_thresholds', '{"auto_approve": 0.85, "require_review": 0.50, "auto_reject": 0.20}', 'Confidence score thresholds for classification review'),
    ('ner_entity_types', '["PERSON", "ORGANIZATION", "LOCATION", "DATE", "MEDICAL_TERM", "FINANCIAL_TERM"]', 'Supported NER entity types'),
    ('document_types', '["MEDICAL_RECORD", "FINANCIAL_STATEMENT", "LEGAL_DOCUMENT", "PII_DOCUMENT", "TECHNICAL_DOCUMENT"]', 'Supported document classification types'),
    ('encryption_scoring_weights', '{"at_rest": 0.40, "in_transit": 0.30, "key_management": 0.30}', 'Weights for encryption compliance scoring')
ON CONFLICT (config_key) DO NOTHING;

-- =====================================================
-- SECTION 6: ADDITIONAL COMPLIANCE CONTROLS
-- =====================================================

-- Add new compliance controls for Phase 2 features
INSERT INTO compliance_controls (framework, control_id, control_name, finding_types, data_categories) VALUES
    ('GDPR', 'Art.30', 'Records of Processing Activities', ARRAY['DATA_LINEAGE_UNKNOWN', 'AI_PROCESSING_SENSITIVE'], ARRAY['PII']),
    ('GDPR', 'Art.22', 'Automated Decision Making', ARRAY['AI_PROCESSING_PII', 'AI_UNMONITORED'], ARRAY['PII']),
    ('HIPAA', '164.312(e)(1)', 'Transmission Security', ARRAY['TRANSIT_ENCRYPTION_WEAK', 'TLS_OUTDATED'], ARRAY['PHI']),
    ('HIPAA', '164.312(d)', 'Person or Entity Authentication', ARRAY['KEY_ROTATION_DISABLED', 'WEAK_ENCRYPTION'], ARRAY['PHI']),
    ('PCI-DSS', '3.5', 'Document Key Management Procedures', ARRAY['KEY_ROTATION_DISABLED', 'KEY_POLICY_OVERPERMISSIVE'], ARRAY['PCI']),
    ('PCI-DSS', '4.1', 'Use Strong Cryptography', ARRAY['TRANSIT_ENCRYPTION_WEAK', 'TLS_OUTDATED'], ARRAY['PCI']),
    ('SOC2', 'CC6.7', 'Restrict Transmission', ARRAY['TRANSIT_ENCRYPTION_DISABLED', 'TLS_OUTDATED'], NULL),
    ('AI-RMF', '1.1', 'AI Risk Mapping', ARRAY['AI_TRAINING_SENSITIVE', 'AI_LINEAGE_UNKNOWN'], NULL),
    ('AI-RMF', '2.3', 'AI Data Governance', ARRAY['AI_PROCESSING_PII', 'AI_TRAINING_UNTRACKED'], NULL)
ON CONFLICT (framework, control_id) DO NOTHING;
