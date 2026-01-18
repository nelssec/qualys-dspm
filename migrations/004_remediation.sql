-- Remediation Actions Table
CREATE TABLE IF NOT EXISTS remediation_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id),
    asset_id UUID NOT NULL REFERENCES data_assets(id),
    finding_id UUID REFERENCES findings(id),

    -- Action details
    action_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    risk_level VARCHAR(10) NOT NULL DEFAULT 'LOW',
    description TEXT,

    -- Parameters and state
    parameters JSONB DEFAULT '{}',
    previous_state JSONB,
    new_state JSONB,

    -- Approval workflow
    approved_at TIMESTAMP WITH TIME ZONE,
    approved_by VARCHAR(255),

    -- Execution tracking
    executed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_message TEXT,

    -- Rollback capability
    rollback_available BOOLEAN DEFAULT false,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_remediation_actions_account ON remediation_actions(account_id);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_asset ON remediation_actions(asset_id);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_status ON remediation_actions(status);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_type ON remediation_actions(action_type);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_created ON remediation_actions(created_at DESC);

-- Action type enum constraint
ALTER TABLE remediation_actions
    DROP CONSTRAINT IF EXISTS remediation_actions_action_type_check;
ALTER TABLE remediation_actions
    ADD CONSTRAINT remediation_actions_action_type_check
    CHECK (action_type IN (
        'ENABLE_BUCKET_ENCRYPTION',
        'BLOCK_PUBLIC_ACCESS',
        'ENABLE_KMS_ROTATION',
        'UPGRADE_TLS',
        'REVOKE_PUBLIC_ACL',
        'ENABLE_VERSIONING',
        'ENABLE_LOGGING',
        'RESTRICT_IAM_POLICY'
    ));

-- Status enum constraint
ALTER TABLE remediation_actions
    DROP CONSTRAINT IF EXISTS remediation_actions_status_check;
ALTER TABLE remediation_actions
    ADD CONSTRAINT remediation_actions_status_check
    CHECK (status IN (
        'PENDING',
        'APPROVED',
        'EXECUTING',
        'COMPLETED',
        'FAILED',
        'ROLLED_BACK',
        'REJECTED'
    ));

-- Risk level enum constraint
ALTER TABLE remediation_actions
    DROP CONSTRAINT IF EXISTS remediation_actions_risk_level_check;
ALTER TABLE remediation_actions
    ADD CONSTRAINT remediation_actions_risk_level_check
    CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH'));

-- Remediation Playbooks Table (for storing custom playbooks)
CREATE TABLE IF NOT EXISTS remediation_playbooks (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    actions JSONB NOT NULL DEFAULT '[]',
    risk_level VARCHAR(10) NOT NULL DEFAULT 'LOW',
    auto_approve BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Remediation Audit Log
CREATE TABLE IF NOT EXISTS remediation_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_id UUID NOT NULL REFERENCES remediation_actions(id),
    event_type VARCHAR(50) NOT NULL,
    event_data JSONB,
    performed_by VARCHAR(255),
    performed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_remediation_audit_action ON remediation_audit_log(action_id);
CREATE INDEX IF NOT EXISTS idx_remediation_audit_time ON remediation_audit_log(performed_at DESC);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_remediation_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger for auto-updating updated_at
DROP TRIGGER IF EXISTS trigger_remediation_actions_updated ON remediation_actions;
CREATE TRIGGER trigger_remediation_actions_updated
    BEFORE UPDATE ON remediation_actions
    FOR EACH ROW
    EXECUTE FUNCTION update_remediation_timestamp();

-- Insert default playbooks
INSERT INTO remediation_playbooks (id, name, description, actions, risk_level, auto_approve)
VALUES
    ('secure-s3-bucket', 'Secure S3 Bucket', 'Apply standard security controls to an S3 bucket',
     '["ENABLE_BUCKET_ENCRYPTION", "BLOCK_PUBLIC_ACCESS", "ENABLE_VERSIONING", "ENABLE_LOGGING"]',
     'MEDIUM', false),
    ('encryption-compliance', 'Encryption Compliance', 'Ensure encryption best practices are followed',
     '["ENABLE_BUCKET_ENCRYPTION", "ENABLE_KMS_ROTATION", "UPGRADE_TLS"]',
     'LOW', false),
    ('public-exposure-fix', 'Fix Public Exposure', 'Remove public access from exposed resources',
     '["BLOCK_PUBLIC_ACCESS", "REVOKE_PUBLIC_ACL"]',
     'MEDIUM', false)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    actions = EXCLUDED.actions,
    risk_level = EXCLUDED.risk_level,
    updated_at = NOW();
