# Building a DSPM from Scratch

A technical deep-dive into building a Data Security Posture Management solution. This document covers the architecture, data flows, and implementation details of discovering, classifying, and protecting sensitive data in cloud environments.

## What is DSPM?

DSPM answers the critical questions every security team needs to know:

| Question | DSPM Capability |
|----------|-----------------|
| Where is my sensitive data? | Discovery across cloud storage |
| What type of data is it? | Classification (PII, PHI, PCI, secrets) |
| Who has access to it? | Access path analysis via IAM graph |
| How does data flow? | Lineage tracking between systems |
| Is AI training on sensitive data? | AI/ML service monitoring |
| Is encryption properly configured? | Key management visibility |
| What can be auto-fixed? | Automated remediation |
| Are there insider threats? | Anomaly detection and threat scoring |

## The DSPM Journey

Every piece of data flows through a connected journey from discovery to remediation:

```mermaid
flowchart LR
    subgraph Discovery
        D1[Connect Cloud Account]
        D2[Discover Assets]
        D3[Sample Content]
    end

    subgraph Classification
        C1[Pattern Matching]
        C2[Context Validation]
        C3[ML Enhancement]
    end

    subgraph Analysis
        A1[Compliance Mapping]
        A2[Risk Scoring]
        A3[Lineage Tracking]
    end

    subgraph Action
        R1[Generate Findings]
        R2[Alert on Critical]
        R3[Auto-Remediate]
    end

    D1 --> D2 --> D3 --> C1 --> C2 --> C3 --> A1 --> A2 --> A3 --> R1 --> R2 --> R3
```

## Architecture

```mermaid
flowchart TB
    subgraph Frontend["Web Dashboard"]
        Dashboard[Dashboard]
        Assets[Assets View]
        Findings[Findings View]
        Rules[Rules View]
        Lineage[Lineage View]
        Remediation[Remediation View]
    end

    subgraph API["Go API Server"]
        Router[Chi Router]
        Auth[JWT Auth]
        Handlers[REST Handlers]
    end

    subgraph Core["Core Services"]
        Scanner[Scanner Service]
        Classifier[Regex Classifier]
        MLClassifier[ML Classifier]
        LineageSvc[Lineage Service]
        AITracker[AI Tracker]
        EncryptionSvc[Encryption Service]
    end

    subgraph SecurityOps["Security Operations"]
        RemediationSvc[Remediation Engine]
        AnomalyDetector[Anomaly Detector]
        ThreatScoring[Threat Scoring]
    end

    subgraph Connectors["Cloud Connectors"]
        AWS[AWS S3/KMS/CloudTrail]
        Azure[Azure Blob/KeyVault]
        GCP[GCS/KMS]
    end

    subgraph Storage["Data Stores"]
        PG[(PostgreSQL)]
        Redis[(Redis Queue)]
        Neo4j[(Neo4j Graph)]
    end

    Frontend --> API
    API --> Core
    API --> SecurityOps
    Core --> Connectors
    Core --> Storage
    SecurityOps --> Connectors
    SecurityOps --> Storage
    Scanner --> Classifier --> MLClassifier
```

## Scan Data Flow

When a scan is triggered, here's what happens:

```mermaid
sequenceDiagram
    participant User
    participant API
    participant Scanner
    participant Classifier
    participant MLClassifier
    participant Store
    participant Notifier

    User->>API: Trigger Scan
    API->>Scanner: Create Scan Job

    loop For Each Storage Asset
        Scanner->>Scanner: List Objects
        Scanner->>Scanner: Sample Files (1MB samples)
        Scanner->>Classifier: Classify Content
        Classifier->>Classifier: Apply Regex Patterns
        Classifier->>Classifier: Validate (Luhn, Format)
        Classifier->>MLClassifier: Get Confidence Score
        MLClassifier-->>Classifier: Score + Entities
        alt Low Confidence
            Classifier->>Store: Add to Review Queue
        else High Confidence
            Classifier-->>Scanner: Classifications
        end
        Scanner->>Store: Save Results
    end

    Scanner->>Store: Generate Findings
    Scanner->>Notifier: Send Critical Alerts
    Notifier->>User: Slack/Email
```

## Classification Pipeline

The classifier uses a multi-stage pipeline to reduce false positives:

```mermaid
flowchart TB
    subgraph Stage1["Stage 1: Pattern Matching"]
        Content[File Content]
        Regex[12+ Regex Patterns]
        Matches[Candidate Matches]
    end

    subgraph Stage2["Stage 2: Context Validation"]
        Context[Context Patterns]
        Keywords[Nearby Keywords]
        Verify[Validation Check]
    end

    subgraph Stage3["Stage 3: Format Validation"]
        Luhn[Luhn Algorithm]
        IBAN[IBAN Checksum]
        Format[Format Rules]
    end

    subgraph Stage4["Stage 4: ML Enhancement"]
        NER[Named Entity Recognition]
        DocType[Document Classification]
        Confidence[Confidence Score]
    end

    subgraph Output["Output"]
        High[High Confidence → Store]
        Review[Low Confidence → Review Queue]
    end

    Content --> Regex --> Matches --> Context --> Keywords --> Verify
    Verify --> Luhn --> IBAN --> Format
    Format --> NER --> DocType --> Confidence
    Confidence -->|>= 0.85| High
    Confidence -->|< 0.85| Review
```

### Detection Rules

| Category | Rule | Pattern Example | Validation |
|----------|------|-----------------|------------|
| **PII** | SSN | `123-45-6789` | Format + Context |
| **PII** | Email | `user@example.com` | RFC 5322 |
| **PII** | Phone | `+1-555-123-4567` | E.164 format |
| **PII** | Date of Birth | `01/15/1990` | Date validation |
| **PCI** | Credit Card | `4111-1111-1111-1111` | Luhn algorithm |
| **PCI** | IBAN | `DE89370400440532013000` | IBAN checksum |
| **PHI** | Medical Record | `MRN:123456789` | Context required |
| **PHI** | ICD Code | `J45.20` | ICD-10 format |
| **Secrets** | AWS Key | `AKIA...` | 20-char format |
| **Secrets** | API Key | `api_key=...` | Context match |

### Sensitivity Categories

```mermaid
mindmap
  root((Sensitive Data))
    PII
      SSN
      Email
      Phone
      Address
      Date of Birth
    PHI
      Medical Records
      Insurance IDs
      Diagnoses
      ICD Codes
    PCI
      Credit Cards
      Bank Accounts
      Routing Numbers
      IBAN
    Secrets
      API Keys
      AWS Credentials
      Private Keys
      JWT Tokens
```

## ML-Enhanced Classification

The ML classifier handles edge cases that regex alone can't catch:

```mermaid
flowchart TB
    Input[Classification Result] --> Score{Confidence Score}

    Score -->|>= 0.85| High[High Confidence]
    Score -->|0.5 - 0.85| Medium[Medium Confidence]
    Score -->|< 0.5| Low[Low Confidence]

    High --> Store[Store Classification]
    Medium --> Review[Add to Review Queue]
    Low --> Review

    Review --> Analyst[Human Review]
    Analyst --> Approve{Decision}
    Approve -->|Confirm| Store
    Approve -->|Reject| Discard[Discard]
    Approve -->|Correct| Feedback[Training Feedback]
```

### Rule + ML Fusion

We combine rule-based patterns with ML for higher accuracy:

```mermaid
flowchart TB
    subgraph RuleBased["Rule-Based (Weight: 0.6)"]
        Regex[Regex Patterns]
        Context[Context Patterns]
        Validation[Validators]
    end

    subgraph MLBased["ML-Based (Weight: 0.4)"]
        NER[NER Model]
        DocClass[Doc Classifier]
    end

    subgraph Fusion["Score Fusion"]
        Combine[Weighted Average]
        Agree{Results Agree?}
        Boost[Boost 1.2x]
        Higher[Use Higher]
        Final[Final Score]
    end

    Regex --> Combine
    Context --> Combine
    Validation --> Combine
    NER --> Combine
    DocClass --> Combine
    Combine --> Agree
    Agree -->|Yes| Boost --> Final
    Agree -->|No| Higher --> Final
```

## Compliance Mapping

Every finding automatically maps to compliance frameworks:

```mermaid
flowchart LR
    subgraph Finding["Security Finding"]
        F1[Unencrypted PII]
        F2[Public S3 Bucket]
        F3[PHI Without Encryption]
    end

    subgraph Frameworks["Compliance Frameworks"]
        GDPR[GDPR]
        HIPAA[HIPAA]
        PCI[PCI-DSS]
        CCPA[CCPA]
        SOC2[SOC2]
    end

    subgraph Requirements["Specific Requirements"]
        G1[Art. 32 - Security]
        G2[Art. 33 - Breach Notification]
        H1[164.312 - Technical Safeguards]
        P1[Req 3 - Protect Stored Data]
        P2[Req 4 - Encrypt Transmission]
    end

    F1 --> GDPR --> G1
    F1 --> HIPAA --> H1
    F2 --> PCI --> P1
    F3 --> HIPAA
    F3 --> GDPR --> G2
```

### Framework Coverage

| Framework | Key Requirements | Auto-Mapped |
|-----------|-----------------|-------------|
| **GDPR** | Art. 4, 5, 25, 32, 33 | PII detection, encryption status |
| **HIPAA** | 164.308, 164.310, 164.312, 164.514 | PHI detection, access controls |
| **PCI-DSS** | Req 1, 3, 4, 7, 10 | Card data, encryption, logging |
| **CCPA** | Consumer data rights | PII categories |
| **SOC2** | Security, availability | Secrets, access patterns |

## Data Lineage Tracking

Track how sensitive data flows between systems:

```mermaid
flowchart LR
    subgraph Sources["Data Sources"]
        S3A[S3: raw-data]
        S3B[S3: uploads]
    end

    subgraph Processing["Processing"]
        Glue[AWS Glue ETL]
        Lambda[Lambda Functions]
        EMR[EMR Cluster]
    end

    subgraph Destinations["Destinations"]
        RDS[(RDS Database)]
        Redshift[(Redshift)]
        S3C[S3: analytics]
    end

    subgraph AI["AI Services"]
        SageMaker[SageMaker]
        Bedrock[Bedrock]
    end

    S3A -->|ETL Job| Glue
    S3B -->|Process| Lambda
    Glue -->|Transform| S3C
    Lambda -->|Load| RDS
    S3C -->|Batch| EMR
    EMR -->|Aggregate| Redshift
    S3C -->|Training| SageMaker
    S3C -->|Fine-tune| Bedrock
```

### Lineage Detection Methods

```mermaid
flowchart TB
    subgraph Detection["Data Sources"]
        CloudTrail[CloudTrail Events]
        CloudWatch[CloudWatch Logs]
        Naming[Naming Patterns]
    end

    subgraph Inference["Inference Engine"]
        Parse[Parse Events]
        Correlate[Correlate Access]
        Pattern[Match Patterns]
    end

    subgraph Output["Lineage Graph"]
        Events[Lineage Events]
        Paths[Data Flow Paths]
        Sensitive[Sensitive Flows Highlighted]
    end

    CloudTrail --> Parse
    CloudWatch --> Parse
    Naming --> Pattern
    Parse --> Correlate
    Correlate --> Events
    Pattern --> Paths
    Events --> Sensitive
    Paths --> Sensitive
```

## AI/ML Training Data Risk

Monitor AI services for sensitive data exposure:

```mermaid
flowchart TB
    subgraph AIServices["AI Services Discovered"]
        SageMaker[SageMaker Models]
        Bedrock[Bedrock Fine-tuning]
        Comprehend[Comprehend]
    end

    subgraph TrainingData["Training Data Analysis"]
        D1[Dataset Sources]
        D2[Sensitive Data Check]
        D3[Classification Results]
    end

    subgraph Risk["Risk Assessment"]
        Score[Risk Score Calculation]
        Factors[Risk Factors]
        Report[AI Risk Report]
    end

    SageMaker --> D1
    Bedrock --> D1
    D1 --> D2 --> D3 --> Score --> Factors --> Report
```

### AI Risk Factors

| Factor | Weight | Description |
|--------|--------|-------------|
| PII in Training Data | 35% | Personal data used for training |
| Public Model Access | 25% | Model accessible without auth |
| Cross-Account Data | 20% | Data from external accounts |
| Unencrypted Storage | 15% | Training data not encrypted |
| No Access Logging | 5% | Missing audit trail |

## Auto-Remediation System

One-click fixes for common security issues:

```mermaid
stateDiagram-v2
    [*] --> Pending: Create Action
    Pending --> Approved: Security Team Approves
    Pending --> Rejected: Reject
    Approved --> Executing: Execute
    Executing --> Completed: Success
    Executing --> Failed: Error
    Completed --> RolledBack: Rollback Request
    Failed --> Pending: Retry

    note right of Pending
        Actions require approval
        based on risk level
    end note

    note right of Completed
        Previous state captured
        for rollback
    end note
```

### Supported Remediation Actions

| Action | Description | Risk | Rollback |
|--------|-------------|------|----------|
| `ENABLE_BUCKET_ENCRYPTION` | Apply SSE-S3 or SSE-KMS | LOW | Yes |
| `BLOCK_PUBLIC_ACCESS` | Enable S3 public access block | MEDIUM | Yes |
| `ENABLE_KMS_ROTATION` | Enable automatic key rotation | LOW | No |
| `REVOKE_PUBLIC_ACL` | Remove public ACL grants | MEDIUM | Yes |
| `ENABLE_VERSIONING` | Enable bucket versioning | LOW | Yes |
| `ENABLE_LOGGING` | Configure S3 access logs | LOW | Yes |

### Remediation Architecture

```mermaid
flowchart TB
    subgraph Trigger["Trigger"]
        Finding[Security Finding]
        Manual[Manual Request]
    end

    subgraph Approval["Approval Workflow"]
        Create[Create Action]
        Pending[Pending State]
        Review{Risk Level?}
        Auto[Auto-Approve]
        Human[Human Approval]
    end

    subgraph Execution["Execution"]
        Capture[Capture Current State]
        Execute[Execute via AWS SDK]
        Verify[Verify Success]
    end

    subgraph Rollback["Rollback Capability"]
        RollReq[Rollback Requested]
        Restore[Restore Previous State]
    end

    Finding --> Create
    Manual --> Create
    Create --> Pending --> Review
    Review -->|Low| Auto --> Capture
    Review -->|Medium/High| Human --> Capture
    Capture --> Execute --> Verify
    Verify --> RollReq --> Restore
```

## Anomaly Detection

Statistical analysis detects unusual data access patterns:

```mermaid
flowchart TB
    subgraph Collection["Data Collection"]
        CloudTrail[CloudTrail Events]
        Parse[Parse S3 Access Events]
    end

    subgraph Baseline["Baseline Building"]
        Aggregate[Aggregate by Principal]
        Stats[Calculate Mean + StdDev]
        Store[Store 30-Day Baseline]
    end

    subgraph Detection["Real-Time Detection"]
        Compare[Compare to Baseline]
        Deviation{Deviation > 3σ?}
        Alert[Create Anomaly]
        Normal[Normal Activity]
    end

    subgraph Scoring["Threat Scoring"]
        Score[Update Threat Score]
        Risk{Risk Level}
        Critical[Urgent Alert]
        Review[Queue Review]
    end

    CloudTrail --> Parse --> Aggregate --> Stats --> Store
    Parse --> Compare --> Deviation
    Deviation -->|Yes| Alert --> Score --> Risk
    Deviation -->|No| Normal
    Risk -->|Critical| Critical
    Risk -->|High/Medium| Review
```

### Anomaly Types

| Type | Description | Severity |
|------|-------------|----------|
| `VOLUME_SPIKE` | Data access volume exceeds 3σ | HIGH |
| `FREQUENCY_SPIKE` | Access frequency exceeds 3σ | MEDIUM |
| `NEW_DESTINATION` | Data flowing to new destination | MEDIUM |
| `OFF_HOURS_ACCESS` | Access outside business hours | LOW |
| `BULK_DOWNLOAD` | >100MB extracted in 1 hour | CRITICAL |
| `GEO_ANOMALY` | Access from unusual location | HIGH |

### Threat Score Calculation

```mermaid
flowchart LR
    subgraph Factors["Risk Factors"]
        A[Recent Anomalies 30%]
        B[Critical Data Access 25%]
        C[Volume Deviation 20%]
        D[Off-Hours Activity 15%]
        E[New Destinations 10%]
    end

    subgraph Calculation["Score"]
        Sum[Weighted Sum]
        Normalize[Normalize 0-100]
    end

    subgraph Output["Risk Level"]
        Low[0-25: LOW]
        Med[26-50: MEDIUM]
        High[51-75: HIGH]
        Crit[76-100: CRITICAL]
    end

    A --> Sum
    B --> Sum
    C --> Sum
    D --> Sum
    E --> Sum
    Sum --> Normalize --> Low & Med & High & Crit
```

## Encryption Visibility

Track encryption keys and compliance:

```mermaid
flowchart TB
    subgraph Keys["Encryption Keys"]
        KMS[AWS KMS Keys]
        Aliases[Key Aliases]
        Grants[Key Grants]
    end

    subgraph Usage["Key Usage Tracking"]
        S3[S3 Buckets]
        EBS[EBS Volumes]
        RDS[RDS Databases]
    end

    subgraph Compliance["Compliance Checks"]
        AtRest[At-Rest Encryption]
        Rotation[Key Rotation Status]
        CMK[Customer Managed vs AWS Managed]
    end

    subgraph Score["Encryption Score"]
        Calculate[Calculate Coverage]
        Gaps[Identify Gaps]
        Recommend[Recommendations]
    end

    KMS --> S3 & EBS & RDS
    S3 --> AtRest --> Calculate
    KMS --> Rotation --> Calculate
    KMS --> CMK --> Calculate
    Calculate --> Gaps --> Recommend
```

## Database Schema

```mermaid
erDiagram
    cloud_accounts ||--o{ data_assets : contains
    cloud_accounts ||--o{ scan_jobs : triggers
    data_assets ||--o{ classifications : has
    data_assets ||--o{ findings : generates
    data_assets ||--o{ remediation_actions : targets
    classifications ||--o{ review_queue : may_require
    findings ||--o{ remediation_actions : resolves

    cloud_accounts {
        uuid id PK
        string provider
        string external_id
        string role_arn
        string status
    }

    data_assets {
        uuid id PK
        uuid account_id FK
        string arn
        string resource_type
        string sensitivity_level
        boolean public_access
        boolean encrypted
    }

    classifications {
        uuid id PK
        uuid asset_id FK
        string category
        string rule_id
        float confidence
        jsonb matches
    }

    findings {
        uuid id PK
        uuid asset_id FK
        string severity
        string status
        string finding_type
        jsonb compliance_frameworks
    }

    remediation_actions {
        uuid id PK
        uuid asset_id FK
        string action_type
        string status
        jsonb parameters
        jsonb previous_state
    }
```

## API Reference

### Core Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/v1/auth/login` | Authenticate user |
| `GET` | `/api/v1/accounts` | List cloud accounts |
| `POST` | `/api/v1/accounts/{id}/scan` | Trigger scan |
| `GET` | `/api/v1/assets` | List data assets |
| `GET` | `/api/v1/findings` | List findings |
| `GET` | `/api/v1/rules` | List detection rules |

### Advanced Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/lineage/graph` | Get lineage graph |
| `GET` | `/api/v1/lineage/sensitive-flows` | Sensitive data flows |
| `GET` | `/api/v1/ai/services` | AI service inventory |
| `GET` | `/api/v1/ai/risk-report` | AI risk assessment |
| `GET` | `/api/v1/encryption/overview` | Encryption status |
| `GET` | `/api/v1/anomalies` | Detected anomalies |
| `POST` | `/api/v1/remediation` | Create remediation |
| `POST` | `/api/v1/remediation/{id}/execute` | Execute fix |

## Performance: Smart Sampling

Handle large buckets efficiently:

```mermaid
flowchart TB
    Start[List Bucket] --> Count{Object Count}
    Count -->|< 1000| Full[Scan All Objects]
    Count -->|1000 - 10000| Sample1[10% Random Sample]
    Count -->|> 10000| Sample2[1000 Objects + Recent]

    Full --> Scan[Scan Objects]
    Sample1 --> Scan
    Sample2 --> Scan

    Scan --> Size{File Size}
    Size -->|< 1MB| ReadAll[Read Full File]
    Size -->|>= 1MB| ReadSample[Read 1MB Sample]

    ReadAll --> Classify[Classify Content]
    ReadSample --> Classify
```

## Security Model

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Auth
    participant AWS

    Client->>API: POST /auth/login
    API->>Auth: Validate Credentials
    Auth-->>API: JWT Token (15min expiry)
    API-->>Client: {access_token, refresh_token}

    Client->>API: GET /api/v1/assets
    Note over API: Validate JWT
    API->>AWS: AssumeRole (cross-account)
    AWS-->>API: Temporary Credentials
    API->>AWS: ListBuckets
    AWS-->>API: Bucket List
    API-->>Client: Assets Response
```

## Key Takeaways

Building a DSPM requires integrating multiple systems:

1. **Multi-stage classification** - Regex → Context → Validation → ML for accuracy
2. **Human-in-the-loop** - Review queue for uncertain classifications
3. **Compliance automation** - Auto-map findings to regulatory frameworks
4. **Data lineage** - Track sensitive data flow, especially to AI services
5. **Proactive remediation** - One-click fixes with approval workflows
6. **Anomaly detection** - Statistical analysis for insider threat detection
7. **Smart sampling** - Handle petabyte-scale buckets efficiently

---

*Built with Go, PostgreSQL, and the AWS SDK*
