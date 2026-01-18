-- Anomaly Detection Tables

-- Access Baselines: stores normal access patterns for principals
CREATE TABLE IF NOT EXISTS access_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id),
    principal_id VARCHAR(255) NOT NULL,
    principal_type VARCHAR(50) NOT NULL,
    principal_name VARCHAR(255),

    -- Time configuration
    time_window VARCHAR(20) DEFAULT 'DAILY',
    baseline_period_start TIMESTAMP WITH TIME ZONE,
    baseline_period_end TIMESTAMP WITH TIME ZONE,

    -- Statistical measures
    avg_daily_access_count DOUBLE PRECISION DEFAULT 0,
    std_dev_access_count DOUBLE PRECISION DEFAULT 0,
    avg_data_volume_bytes DOUBLE PRECISION DEFAULT 0,
    std_dev_data_volume DOUBLE PRECISION DEFAULT 0,

    -- Pattern data (JSON arrays)
    normal_access_hours JSONB DEFAULT '[]',
    normal_access_days JSONB DEFAULT '[]',
    common_assets JSONB DEFAULT '[]',
    common_operations JSONB DEFAULT '[]',
    common_source_ips JSONB DEFAULT '[]',
    common_geo_locations JSONB DEFAULT '[]',

    -- Thresholds
    access_count_threshold DOUBLE PRECISION DEFAULT 0,
    data_volume_threshold DOUBLE PRECISION DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(account_id, principal_id)
);

CREATE INDEX IF NOT EXISTS idx_baselines_account ON access_baselines(account_id);
CREATE INDEX IF NOT EXISTS idx_baselines_principal ON access_baselines(principal_id);

-- Anomalies: detected security anomalies
CREATE TABLE IF NOT EXISTS anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id),
    asset_id UUID REFERENCES data_assets(id),

    -- Principal information
    principal_id VARCHAR(255),
    principal_type VARCHAR(50),
    principal_name VARCHAR(255),

    -- Anomaly details
    anomaly_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'NEW',
    severity VARCHAR(20) NOT NULL DEFAULT 'LOW',
    title VARCHAR(500) NOT NULL,
    description TEXT,
    details JSONB DEFAULT '{}',

    -- Statistical values
    baseline_value DOUBLE PRECISION DEFAULT 0,
    observed_value DOUBLE PRECISION DEFAULT 0,
    deviation_factor DOUBLE PRECISION DEFAULT 0,

    -- Timing
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolved_by VARCHAR(255),
    resolution TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_anomalies_account ON anomalies(account_id);
CREATE INDEX IF NOT EXISTS idx_anomalies_principal ON anomalies(principal_id);
CREATE INDEX IF NOT EXISTS idx_anomalies_status ON anomalies(status);
CREATE INDEX IF NOT EXISTS idx_anomalies_type ON anomalies(anomaly_type);
CREATE INDEX IF NOT EXISTS idx_anomalies_severity ON anomalies(severity);
CREATE INDEX IF NOT EXISTS idx_anomalies_detected ON anomalies(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_anomalies_asset ON anomalies(asset_id);

-- Anomaly type constraint
ALTER TABLE anomalies
    DROP CONSTRAINT IF EXISTS anomalies_type_check;
ALTER TABLE anomalies
    ADD CONSTRAINT anomalies_type_check
    CHECK (anomaly_type IN (
        'VOLUME_SPIKE',
        'FREQUENCY_SPIKE',
        'NEW_DESTINATION',
        'OFF_HOURS_ACCESS',
        'BULK_DOWNLOAD',
        'UNUSUAL_PATTERN',
        'GEO_ANOMALY',
        'PRIVILEGE_ESCALATION'
    ));

-- Anomaly status constraint
ALTER TABLE anomalies
    DROP CONSTRAINT IF EXISTS anomalies_status_check;
ALTER TABLE anomalies
    ADD CONSTRAINT anomalies_status_check
    CHECK (status IN (
        'NEW',
        'INVESTIGATING',
        'CONFIRMED',
        'FALSE_POSITIVE',
        'RESOLVED'
    ));

-- Severity constraint
ALTER TABLE anomalies
    DROP CONSTRAINT IF EXISTS anomalies_severity_check;
ALTER TABLE anomalies
    ADD CONSTRAINT anomalies_severity_check
    CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL'));

-- Threat Scores: insider threat scoring for principals
CREATE TABLE IF NOT EXISTS threat_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id),
    principal_id VARCHAR(255) NOT NULL,
    principal_type VARCHAR(50) NOT NULL,
    principal_name VARCHAR(255),

    -- Score data
    score DOUBLE PRECISION DEFAULT 0,
    risk_level VARCHAR(20) DEFAULT 'LOW',
    factors JSONB DEFAULT '[]',
    recent_anomalies INTEGER DEFAULT 0,
    trend_direction VARCHAR(10) DEFAULT 'STABLE',
    details JSONB DEFAULT '{}',

    -- Timing
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(account_id, principal_id)
);

CREATE INDEX IF NOT EXISTS idx_threat_scores_account ON threat_scores(account_id);
CREATE INDEX IF NOT EXISTS idx_threat_scores_score ON threat_scores(score DESC);
CREATE INDEX IF NOT EXISTS idx_threat_scores_risk ON threat_scores(risk_level);

-- Risk level constraint
ALTER TABLE threat_scores
    DROP CONSTRAINT IF EXISTS threat_scores_risk_check;
ALTER TABLE threat_scores
    ADD CONSTRAINT threat_scores_risk_check
    CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL'));

-- Trend direction constraint
ALTER TABLE threat_scores
    DROP CONSTRAINT IF EXISTS threat_scores_trend_check;
ALTER TABLE threat_scores
    ADD CONSTRAINT threat_scores_trend_check
    CHECK (trend_direction IN ('UP', 'DOWN', 'STABLE'));

-- Detection Rules: configurable anomaly detection rules
CREATE TABLE IF NOT EXISTS detection_rules (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    anomaly_type VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    severity VARCHAR(20) DEFAULT 'MEDIUM',
    threshold DOUBLE PRECISION DEFAULT 3.0,
    conditions JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default detection rules
INSERT INTO detection_rules (id, name, description, anomaly_type, enabled, severity, threshold, conditions)
VALUES
    ('volume-spike', 'Data Volume Spike', 'Detects unusual data access volume compared to baseline', 'VOLUME_SPIKE', true, 'HIGH', 3.0, '{"min_volume_bytes": 10485760}'),
    ('frequency-spike', 'Access Frequency Spike', 'Detects unusual access frequency compared to baseline', 'FREQUENCY_SPIKE', true, 'MEDIUM', 3.0, '{"min_access_count": 10}'),
    ('new-destination', 'New Data Destination', 'Detects data flow to a new or unusual destination', 'NEW_DESTINATION', true, 'MEDIUM', 0, '{}'),
    ('off-hours-access', 'Off-Hours Data Access', 'Detects data access outside normal working hours', 'OFF_HOURS_ACCESS', true, 'LOW', 0, '{"off_hours_start": 22, "off_hours_end": 6}'),
    ('bulk-download', 'Bulk Data Download', 'Detects large-scale data extraction events', 'BULK_DOWNLOAD', true, 'CRITICAL', 5.0, '{"min_volume_bytes": 104857600, "time_window_minutes": 60}'),
    ('geo-anomaly', 'Geographic Anomaly', 'Detects access from unusual geographic locations', 'GEO_ANOMALY', true, 'HIGH', 0, '{}')
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    anomaly_type = EXCLUDED.anomaly_type,
    severity = EXCLUDED.severity,
    threshold = EXCLUDED.threshold,
    conditions = EXCLUDED.conditions,
    updated_at = NOW();

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_anomaly_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for auto-updating timestamps
DROP TRIGGER IF EXISTS trigger_baselines_updated ON access_baselines;
CREATE TRIGGER trigger_baselines_updated
    BEFORE UPDATE ON access_baselines
    FOR EACH ROW
    EXECUTE FUNCTION update_anomaly_timestamp();

DROP TRIGGER IF EXISTS trigger_anomalies_updated ON anomalies;
CREATE TRIGGER trigger_anomalies_updated
    BEFORE UPDATE ON anomalies
    FOR EACH ROW
    EXECUTE FUNCTION update_anomaly_timestamp();

DROP TRIGGER IF EXISTS trigger_detection_rules_updated ON detection_rules;
CREATE TRIGGER trigger_detection_rules_updated
    BEFORE UPDATE ON detection_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_anomaly_timestamp();
