# Data Model

This document describes the database schema and entity relationships for the DSPM solution.

---

## Entity Relationship Diagram

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│  CloudAccount   │       │   DataAsset     │       │  Classification │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id              │──┐    │ id              │──┐    │ id              │
│ provider        │  │    │ account_id (FK) │  │    │ asset_id (FK)   │
│ account_id      │  └───▶│ resource_type   │  └───▶│ rule_name       │
│ name            │       │ resource_arn    │       │ category        │
│ regions[]       │       │ region          │       │ sensitivity     │
│ status          │       │ name            │       │ finding_count   │
│ last_scan       │       │ encryption_type │       │ sample_path     │
└─────────────────┘       │ public_access   │       │ discovered_at   │
                          │ created_at      │       └─────────────────┘
                          │ tags{}          │
                          │ metadata{}      │
                          └─────────────────┘
                                  │
                                  ▼
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│  AccessPolicy   │       │  AccessEdge     │       │  ComplianceMap  │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id              │       │ id              │       │ id              │
│ account_id      │       │ source_type     │       │ asset_id (FK)   │
│ policy_arn      │       │ source_id       │       │ framework       │
│ policy_type     │       │ target_id (FK)  │       │ control_id      │
│ policy_document │       │ permission_level│       │ status          │
│ attached_to[]   │       │ policy_id (FK)  │       │ evidence{}      │
└─────────────────┘       │ is_public       │       │ last_evaluated  │
                          │ conditions{}    │       └─────────────────┘
                          └─────────────────┘

┌─────────────────┐       ┌─────────────────┐
│    Finding      │       │   ScanJob       │
├─────────────────┤       ├─────────────────┤
│ id              │       │ id              │
│ asset_id (FK)   │       │ account_id      │
│ finding_type    │       │ scan_type       │
│ severity        │       │ status          │
│ title           │       │ started_at      │
│ description     │       │ completed_at    │
│ remediation     │       │ assets_scanned  │
│ status          │       │ findings_count  │
│ created_at      │       │ errors[]        │
└─────────────────┘       └─────────────────┘
```

---

## PostgreSQL Schema

### Cloud Accounts

```sql
-- Cloud accounts being monitored
CREATE TABLE cloud_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(20) NOT NULL,  -- AWS, AZURE, GCP
    external_id VARCHAR(100) NOT NULL,  -- AWS account ID, subscription ID, project ID
    display_name VARCHAR(255),

    -- Connection configuration
    connector_config JSONB NOT NULL,
    -- AWS: {"role_arn": "...", "external_id": "..."}
    -- Azure: {"tenant_id": "...", "client_id": "...", "subscription_id": "..."}
    -- GCP: {"project_id": "...", "credentials_path": "..."}

    status VARCHAR(20) DEFAULT 'active',  -- active, inactive, error
    status_message TEXT,

    -- Scanning state
    last_scan_at TIMESTAMP,
    last_scan_status VARCHAR(20),

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(provider, external_id)
);

CREATE INDEX idx_accounts_provider ON cloud_accounts(provider);
CREATE INDEX idx_accounts_status ON cloud_accounts(status);
```

### Data Assets

```sql
-- Discovered data assets (buckets, databases, functions, etc.)
CREATE TABLE data_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Resource identification
    resource_type VARCHAR(50) NOT NULL,  -- s3_bucket, azure_blob_container, gcs_bucket, lambda_function, etc.
    resource_arn VARCHAR(500) NOT NULL,  -- Unique resource identifier
    region VARCHAR(50),
    name VARCHAR(255) NOT NULL,

    -- Security posture
    encryption_status VARCHAR(20),  -- NONE, SSE_S3, SSE_KMS, CMK, AES256
    encryption_key_arn VARCHAR(500),
    public_access BOOLEAN DEFAULT false,
    public_access_details JSONB,  -- Details about how it's public
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
    sensitivity_level VARCHAR(20),  -- CRITICAL, HIGH, MEDIUM, LOW, UNKNOWN
    data_categories TEXT[] DEFAULT '{}',  -- ['PII', 'PCI', 'PHI']
    classification_count INTEGER DEFAULT 0,

    -- Scanning state
    last_scanned_at TIMESTAMP,
    scan_status VARCHAR(20),  -- pending, scanning, completed, error
    scan_error TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(resource_arn)
);

CREATE INDEX idx_assets_account ON data_assets(account_id);
CREATE INDEX idx_assets_type ON data_assets(resource_type);
CREATE INDEX idx_assets_sensitivity ON data_assets(sensitivity_level);
CREATE INDEX idx_assets_public ON data_assets(public_access) WHERE public_access = true;
CREATE INDEX idx_assets_categories ON data_assets USING GIN(data_categories);
```

### Classifications

```sql
-- Data classification findings
CREATE TABLE classifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,

    -- Location within asset
    object_path VARCHAR(1000),  -- e.g., "folder/file.csv" within a bucket
    object_size BIGINT,

    -- Classification details
    rule_name VARCHAR(100) NOT NULL,  -- SSN, EMAIL, CREDIT_CARD, etc.
    category VARCHAR(50) NOT NULL,    -- PII, PHI, PCI, SECRETS, CUSTOM
    sensitivity VARCHAR(20) NOT NULL, -- CRITICAL, HIGH, MEDIUM, LOW

    -- Match details
    finding_count INTEGER DEFAULT 1,
    sample_matches JSONB,  -- Redacted sample matches for evidence
    match_locations JSONB, -- Line numbers, byte offsets

    -- Confidence
    confidence_score DECIMAL(3,2),  -- 0.00 to 1.00
    validated BOOLEAN DEFAULT false,

    discovered_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(asset_id, object_path, rule_name)
);

CREATE INDEX idx_class_asset ON classifications(asset_id);
CREATE INDEX idx_class_category ON classifications(category);
CREATE INDEX idx_class_sensitivity ON classifications(sensitivity);
CREATE INDEX idx_class_rule ON classifications(rule_name);
```

### Access Policies

```sql
-- IAM policies and permissions
CREATE TABLE access_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Policy identification
    policy_arn VARCHAR(500) NOT NULL,
    policy_name VARCHAR(255),
    policy_type VARCHAR(50),  -- MANAGED, INLINE, RESOURCE_BASED, BUCKET_POLICY

    -- Policy content
    policy_document JSONB NOT NULL,
    policy_version VARCHAR(50),

    -- Attachments
    attached_to JSONB DEFAULT '[]',  -- List of principals this is attached to

    -- Analysis results
    allows_public_access BOOLEAN DEFAULT false,
    overly_permissive BOOLEAN DEFAULT false,
    analysis_notes TEXT,

    discovered_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(policy_arn)
);

CREATE INDEX idx_policies_account ON access_policies(account_id);
CREATE INDEX idx_policies_public ON access_policies(allows_public_access) WHERE allows_public_access = true;
```

### Access Edges (Graph)

```sql
-- Access relationships between principals and assets
CREATE TABLE access_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Source (who/what has access)
    source_type VARCHAR(50) NOT NULL,  -- USER, ROLE, SERVICE, GROUP, PUBLIC
    source_arn VARCHAR(500) NOT NULL,
    source_name VARCHAR(255),

    -- Target (what they have access to)
    target_asset_id UUID REFERENCES data_assets(id) ON DELETE CASCADE,
    target_arn VARCHAR(500),

    -- Access details
    permission_level VARCHAR(20),  -- READ, WRITE, ADMIN, FULL
    permissions TEXT[],  -- Specific actions: ['s3:GetObject', 's3:PutObject']

    -- How access is granted
    policy_id UUID REFERENCES access_policies(id),
    grant_type VARCHAR(50),  -- DIRECT, INHERITED, ASSUMED_ROLE, GROUP_MEMBERSHIP

    -- Flags
    is_direct BOOLEAN DEFAULT true,
    is_public BOOLEAN DEFAULT false,
    is_cross_account BOOLEAN DEFAULT false,

    -- Conditions
    conditions JSONB,  -- IAM conditions, IP restrictions, etc.

    discovered_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_edges_target ON access_edges(target_asset_id);
CREATE INDEX idx_edges_source ON access_edges(source_arn);
CREATE INDEX idx_edges_public ON access_edges(is_public) WHERE is_public = true;
CREATE INDEX idx_edges_level ON access_edges(permission_level);
```

### Findings

```sql
-- Security findings and risks
CREATE TABLE findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Scope
    account_id UUID REFERENCES cloud_accounts(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES data_assets(id) ON DELETE SET NULL,

    -- Finding details
    finding_type VARCHAR(100) NOT NULL,  -- PUBLIC_BUCKET, UNENCRYPTED_DATA, OVERPRIVILEGED_ACCESS
    severity VARCHAR(20) NOT NULL,       -- CRITICAL, HIGH, MEDIUM, LOW, INFO

    title VARCHAR(500) NOT NULL,
    description TEXT,
    remediation TEXT,

    -- Status
    status VARCHAR(20) DEFAULT 'open',  -- open, in_progress, resolved, suppressed, false_positive
    status_reason TEXT,
    assigned_to VARCHAR(255),

    -- Compliance
    compliance_frameworks TEXT[],  -- ['GDPR-Art32', 'HIPAA-164.312', 'PCI-DSS-3.4']

    -- Evidence
    evidence JSONB,
    resource_snapshot JSONB,  -- Point-in-time resource state

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP,
    first_seen_at TIMESTAMP DEFAULT NOW(),
    last_seen_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_findings_account ON findings(account_id);
CREATE INDEX idx_findings_asset ON findings(asset_id);
CREATE INDEX idx_findings_severity ON findings(severity);
CREATE INDEX idx_findings_status ON findings(status);
CREATE INDEX idx_findings_type ON findings(finding_type);
```

### Scan Jobs

```sql
-- Scan job tracking
CREATE TABLE scan_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID REFERENCES cloud_accounts(id) ON DELETE CASCADE,

    -- Job configuration
    scan_type VARCHAR(50) NOT NULL,  -- FULL, INCREMENTAL, ASSET_DISCOVERY, CLASSIFICATION, ACCESS_ANALYSIS
    scan_scope JSONB,  -- Specific resources to scan

    -- Status
    status VARCHAR(20) DEFAULT 'pending',  -- pending, running, completed, failed, cancelled

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
    triggered_by VARCHAR(100),  -- scheduled, manual, api
    worker_id VARCHAR(100)
);

CREATE INDEX idx_jobs_account ON scan_jobs(account_id);
CREATE INDEX idx_jobs_status ON scan_jobs(status);
CREATE INDEX idx_jobs_type ON scan_jobs(scan_type);
```

### Compliance Mappings

```sql
-- Compliance control mappings
CREATE TABLE compliance_controls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Framework
    framework VARCHAR(50) NOT NULL,  -- GDPR, HIPAA, PCI_DSS, SOC2, NIST
    control_id VARCHAR(50) NOT NULL,  -- Art.32, 164.312(a), 3.4.1
    control_name VARCHAR(255),
    control_description TEXT,

    -- Mapping to DSPM
    finding_types TEXT[],  -- Finding types that map to this control
    data_categories TEXT[],  -- Data categories relevant to this control

    -- Requirements
    requirements JSONB,

    UNIQUE(framework, control_id)
);

-- Asset-level compliance status
CREATE TABLE compliance_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES data_assets(id) ON DELETE CASCADE,
    control_id UUID NOT NULL REFERENCES compliance_controls(id),

    -- Status
    status VARCHAR(20) NOT NULL,  -- COMPLIANT, NON_COMPLIANT, PARTIALLY_COMPLIANT, NOT_APPLICABLE

    -- Evidence
    evidence JSONB,
    finding_ids UUID[],

    -- Evaluation
    evaluated_at TIMESTAMP DEFAULT NOW(),
    evaluated_by VARCHAR(100),  -- automated, manual

    notes TEXT,

    UNIQUE(asset_id, control_id)
);

CREATE INDEX idx_compliance_asset ON compliance_status(asset_id);
CREATE INDEX idx_compliance_status ON compliance_status(status);
```

---

## Neo4j Graph Schema

For complex access path analysis, we use Neo4j alongside PostgreSQL.

### Node Types

```cypher
// Cloud Account
(:CloudAccount {
    id: string,
    provider: string,
    externalId: string,
    displayName: string
})

// Data Asset
(:DataAsset {
    id: string,
    arn: string,
    resourceType: string,
    name: string,
    region: string,
    sensitivityLevel: string,
    encryptionStatus: string,
    publicAccess: boolean
})

// Principal (user, role, service account)
(:Principal {
    id: string,
    arn: string,
    type: string,  // USER, ROLE, SERVICE, GROUP
    name: string,
    accountId: string
})

// Policy
(:Policy {
    id: string,
    arn: string,
    name: string,
    policyType: string
})

// Data Classification
(:Classification {
    category: string,
    sensitivity: string
})
```

### Relationship Types

```cypher
// Account ownership
(asset:DataAsset)-[:BELONGS_TO]->(account:CloudAccount)
(principal:Principal)-[:BELONGS_TO]->(account:CloudAccount)

// Access relationships
(principal:Principal)-[:CAN_ACCESS {
    permissions: [string],
    permissionLevel: string,
    isDirect: boolean,
    isPublic: boolean,
    conditions: map
}]->(asset:DataAsset)

// Role assumption
(principal:Principal)-[:CAN_ASSUME]->(role:Principal)

// Policy attachments
(policy:Policy)-[:ATTACHED_TO]->(principal:Principal)
(policy:Policy)-[:GRANTS_ACCESS {actions: [string]}]->(asset:DataAsset)

// Data classification
(asset:DataAsset)-[:CONTAINS_DATA {
    count: int,
    objectPaths: [string]
}]->(classification:Classification)
```

### Example Queries

```cypher
// Find all paths to sensitive data from public access
MATCH path = (p:Principal {type: 'PUBLIC'})-[:CAN_ACCESS*1..5]->(a:DataAsset)
WHERE a.sensitivityLevel = 'CRITICAL'
RETURN path

// Who can access PII data?
MATCH (p:Principal)-[r:CAN_ACCESS]->(a:DataAsset)-[:CONTAINS_DATA]->(c:Classification)
WHERE c.category = 'PII'
RETURN p.arn AS principal, a.arn AS asset, r.permissions AS permissions

// Find over-privileged access (ADMIN on sensitive data)
MATCH (p:Principal)-[r:CAN_ACCESS]->(a:DataAsset)
WHERE r.permissionLevel = 'ADMIN' AND a.sensitivityLevel IN ['CRITICAL', 'HIGH']
RETURN p.arn, a.arn, r.permissions

// Cross-account access paths
MATCH (p:Principal)-[:BELONGS_TO]->(acc1:CloudAccount),
      (a:DataAsset)-[:BELONGS_TO]->(acc2:CloudAccount),
      path = (p)-[:CAN_ACCESS|CAN_ASSUME*1..3]->(a)
WHERE acc1.id <> acc2.id
RETURN path
```

---

## Go Struct Mappings

```go
// See internal/models/ for Go struct definitions that map to these tables
```
