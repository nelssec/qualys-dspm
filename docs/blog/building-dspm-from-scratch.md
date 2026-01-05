# Building a Data Security Posture Management (DSPM) Solution from Scratch

Data Security Posture Management (DSPM) has emerged as a critical component of modern cloud security. In this post, we'll walk through the architecture and implementation of a complete DSPM solution that supports AWS, Azure, and GCP.

## What is DSPM?

DSPM solutions help organizations answer critical questions about their data:

- **Where is my sensitive data?** - Discovery across cloud storage
- **What type of data is it?** - Classification (PII, PHI, PCI, secrets)
- **Who has access to it?** - Access path analysis
- **Is it properly protected?** - Security posture assessment
- **Am I compliant?** - Regulatory framework mapping

## High-Level Architecture

```mermaid
flowchart TB
    subgraph Frontend["Frontend (React)"]
        Dashboard[Dashboard]
        Assets[Asset Explorer]
        Findings[Findings Manager]
        Rules[Rules Editor]
        Reports[Report Generator]
    end

    subgraph API["API Layer (Go)"]
        Router[Chi Router]
        Auth[JWT Auth]
        Handlers[API Handlers]
    end

    subgraph Core["Core Services"]
        Scanner[Scanner Service]
        Classifier[Classification Engine]
        Scheduler[Job Scheduler]
        Notifier[Notification Service]
    end

    subgraph Connectors["Cloud Connectors"]
        AWS[AWS Connector]
        Azure[Azure Connector]
        GCP[GCP Connector]
    end

    subgraph Storage["Data Stores"]
        PG[(PostgreSQL)]
        Redis[(Redis)]
        Neo4j[(Neo4j)]
    end

    subgraph Cloud["Cloud Providers"]
        S3[S3 Buckets]
        Blob[Azure Blob]
        GCS[Cloud Storage]
        IAM[IAM Policies]
    end

    Frontend --> API
    API --> Core
    Core --> Connectors
    Connectors --> Cloud
    Core --> Storage
    Scanner --> Classifier
    Scanner --> Notifier
```

## Data Flow

The DSPM solution follows a clear data flow from discovery to remediation:

```mermaid
sequenceDiagram
    participant User
    participant API
    participant Scanner
    participant Classifier
    participant Store
    participant Notifier

    User->>API: Trigger Scan
    API->>Scanner: Create Scan Job
    Scanner->>Store: Get Account Config

    loop For Each Storage Asset
        Scanner->>Scanner: List Objects
        Scanner->>Scanner: Sample Files
        Scanner->>Classifier: Classify Content
        Classifier->>Classifier: Apply Patterns
        Classifier->>Classifier: Validate Matches
        Classifier-->>Scanner: Classifications
        Scanner->>Store: Save Results
    end

    Scanner->>Store: Update Scan Status
    Scanner->>Notifier: Send Alerts
    Notifier->>User: Slack/Email
    API-->>User: Scan Complete
```

## Classification Pipeline

The classification engine is the heart of DSPM. It uses a multi-stage pipeline to accurately identify sensitive data:

```mermaid
flowchart LR
    subgraph Input
        Content[File Content]
    end

    subgraph Stage1["Stage 1: Pattern Matching"]
        Regex[Regex Patterns]
        Match[Pattern Matcher]
    end

    subgraph Stage2["Stage 2: Context Analysis"]
        Context[Context Patterns]
        Verify[Context Verification]
    end

    subgraph Stage3["Stage 3: Validation"]
        Luhn[Luhn Algorithm]
        Format[Format Validation]
        Checksum[Checksum Verification]
    end

    subgraph Output
        Result[Classification Result]
    end

    Content --> Regex
    Regex --> Match
    Match --> Context
    Context --> Verify
    Verify --> Luhn
    Luhn --> Format
    Format --> Checksum
    Checksum --> Result
```

### Classification Categories

We classify data into four main categories:

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
      Prescriptions
    PCI
      Credit Cards
      Bank Accounts
      Routing Numbers
    Secrets
      API Keys
      Passwords
      Private Keys
      JWT Tokens
```

## Access Graph Analysis

Understanding who can access sensitive data requires analyzing IAM policies. We use Neo4j to model access relationships:

```mermaid
graph LR
    subgraph Principals
        User1[User: Alice]
        User2[User: Bob]
        Role1[Role: DataAdmin]
        Role2[Role: ReadOnly]
    end

    subgraph Policies
        Policy1[Policy: S3FullAccess]
        Policy2[Policy: S3ReadOnly]
    end

    subgraph Resources
        Bucket1[Bucket: sensitive-data]
        Bucket2[Bucket: public-assets]
    end

    User1 -->|assumes| Role1
    User2 -->|assumes| Role2
    Role1 -->|attached| Policy1
    Role2 -->|attached| Policy2
    Policy1 -->|grants| Bucket1
    Policy1 -->|grants| Bucket2
    Policy2 -->|grants| Bucket2
```

This graph structure allows us to answer queries like:
- "Who can access bucket X?"
- "What resources can user Y access?"
- "Is there any public access to sensitive data?"

## Scan Scheduling

The scheduler uses cron expressions to automate scans:

```mermaid
stateDiagram-v2
    [*] --> Pending: Job Created
    Pending --> Running: Cron Trigger
    Running --> Completed: Success
    Running --> Failed: Error
    Completed --> [*]
    Failed --> Pending: Retry
    Failed --> [*]: Max Retries
```

### Job Types

| Job Type | Description | Typical Schedule |
|----------|-------------|------------------|
| `scan_account` | Scan specific account | `0 */6 * * *` (every 6h) |
| `scan_all_accounts` | Full scan all accounts | `0 2 * * *` (daily 2am) |
| `sync_access_graph` | Update IAM graph | `0 3 * * 0` (weekly) |
| `cleanup_old` | Remove old data | `0 4 1 * *` (monthly) |
| `generate_report` | Scheduled reports | `0 8 * * 1` (weekly Mon) |

## Notification Flow

When critical findings are detected, notifications are sent through multiple channels:

```mermaid
flowchart TB
    Finding[New Finding] --> Check{Severity >= Threshold?}
    Check -->|Yes| Route[Route to Channels]
    Check -->|No| Log[Log Only]

    Route --> Slack[Slack Webhook]
    Route --> Email[SMTP Email]

    Slack --> Format1[Format Message]
    Email --> Format2[HTML Template]

    Format1 --> Send1[POST to Webhook]
    Format2 --> Send2[Send via SMTP]

    Send1 --> Done[Notification Sent]
    Send2 --> Done
```

### Slack Message Format

```json
{
  "attachments": [{
    "color": "#FF0000",
    "title": "CRITICAL Security Finding",
    "text": "Credit card numbers detected in s3://sensitive-bucket/data.csv",
    "fields": [
      {"title": "Severity", "value": "Critical", "short": true},
      {"title": "Category", "value": "PCI", "short": true},
      {"title": "Asset", "value": "s3://sensitive-bucket", "short": false}
    ],
    "footer": "DSPM Alert System"
  }]
}
```

## Report Generation

The reporting system supports multiple formats and report types:

```mermaid
flowchart LR
    subgraph Request
        Type[Report Type]
        Format[Format]
        Filters[Filters]
    end

    subgraph Generator
        Fetch[Fetch Data]
        Transform[Transform]
        Render[Render Output]
    end

    subgraph Output
        CSV[CSV File]
        PDF[PDF Document]
    end

    Type --> Fetch
    Format --> Render
    Filters --> Fetch
    Fetch --> Transform
    Transform --> Render
    Render --> CSV
    Render --> PDF
```

### Report Types

```mermaid
graph TB
    subgraph Reports
        Findings[Findings Report]
        Assets[Asset Inventory]
        Classification[Classification Summary]
        Executive[Executive Summary]
        Compliance[Compliance Report]
    end

    subgraph Findings Content
        F1[Finding Details]
        F2[Severity Breakdown]
        F3[Remediation Steps]
    end

    subgraph Compliance Content
        C1[GDPR Status]
        C2[HIPAA Status]
        C3[PCI-DSS Status]
        C4[SOC 2 Status]
    end

    Findings --> F1
    Findings --> F2
    Findings --> F3
    Compliance --> C1
    Compliance --> C2
    Compliance --> C3
    Compliance --> C4
```

## Database Schema

The PostgreSQL schema captures the full data model:

```mermaid
erDiagram
    cloud_accounts ||--o{ data_assets : contains
    cloud_accounts ||--o{ scan_jobs : triggers
    data_assets ||--o{ classifications : has
    data_assets ||--o{ findings : generates

    cloud_accounts {
        uuid id PK
        string provider
        string external_id
        string display_name
        jsonb connector_config
        string status
    }

    data_assets {
        uuid id PK
        uuid account_id FK
        string resource_type
        string resource_id
        string region
        string sensitivity_level
        boolean public_access
        jsonb metadata
    }

    classifications {
        uuid id PK
        uuid asset_id FK
        string category
        string sensitivity
        string file_path
        float confidence
        jsonb matches
    }

    findings {
        uuid id PK
        uuid asset_id FK
        string finding_type
        string severity
        string status
        jsonb details
        string remediation
    }

    scan_jobs {
        uuid id PK
        uuid account_id FK
        string scan_type
        string status
        timestamp started_at
        timestamp completed_at
        jsonb results_summary
    }
```

## Custom Rules Engine

The rules engine allows security teams to define custom classification patterns:

```mermaid
flowchart TB
    subgraph Rule Definition
        Name[Rule Name]
        Patterns[Regex Patterns]
        Context[Context Patterns]
        Config[Configuration]
    end

    subgraph Compilation
        Parse[Parse Regex]
        Validate[Validate Syntax]
        Compile[Compile Patterns]
    end

    subgraph Runtime
        Load[Load Rules]
        Priority[Sort by Priority]
        Execute[Execute Matching]
    end

    Name --> Compile
    Patterns --> Parse
    Parse --> Validate
    Validate --> Compile
    Context --> Compile
    Config --> Compile
    Compile --> Load
    Load --> Priority
    Priority --> Execute
```

### Rule Example

```yaml
name: "Employee ID Pattern"
description: "Detects internal employee IDs"
category: "pii"
sensitivity: "medium"
patterns:
  - "\\bEMP-\\d{6}\\b"
  - "\\bEID:\\s*\\d{6}\\b"
context_patterns:
  - "(?i)employee|staff|personnel"
context_required: true
priority: 75
enabled: true
```

## Security Architecture

The API is secured with JWT authentication and role-based access control:

```mermaid
sequenceDiagram
    participant Client
    participant API
    participant Auth
    participant DB

    Client->>API: POST /auth/login
    API->>Auth: Validate Credentials
    Auth->>DB: Check User
    DB-->>Auth: User Found
    Auth->>Auth: Generate JWT
    Auth-->>API: Token Pair
    API-->>Client: {access_token, refresh_token}

    Client->>API: GET /api/v1/findings
    API->>Auth: Validate Token
    Auth->>Auth: Check Expiry
    Auth->>Auth: Verify Signature
    Auth-->>API: Claims {user_id, role}
    API->>API: Check Role Permission
    API->>DB: Fetch Data
    DB-->>API: Results
    API-->>Client: Findings List
```

### Role Hierarchy

```mermaid
graph TB
    Admin[Admin Role]
    User[User Role]
    Viewer[Viewer Role]

    Admin -->|can do everything| User
    User -->|can do everything| Viewer

    subgraph Admin Permissions
        A1[Manage Users]
        A2[Configure Notifications]
        A3[Delete Resources]
    end

    subgraph User Permissions
        U1[Create Accounts]
        U2[Trigger Scans]
        U3[Manage Rules]
        U4[Generate Reports]
    end

    subgraph Viewer Permissions
        V1[View Dashboard]
        V2[View Findings]
        V3[View Assets]
    end

    Admin --> A1
    Admin --> A2
    Admin --> A3
    User --> U1
    User --> U2
    User --> U3
    User --> U4
    Viewer --> V1
    Viewer --> V2
    Viewer --> V3
```

## Deployment Architecture

For production deployments, we recommend a containerized architecture:

```mermaid
flowchart TB
    subgraph Internet
        Users[Users]
    end

    subgraph LoadBalancer
        LB[Load Balancer]
    end

    subgraph Kubernetes["Kubernetes Cluster"]
        subgraph Frontend["Frontend Pods"]
            FE1[React App]
            FE2[React App]
        end

        subgraph Backend["Backend Pods"]
            API1[API Server]
            API2[API Server]
            Worker1[Queue Worker]
            Worker2[Queue Worker]
        end

        subgraph Scheduler["Scheduler Pod"]
            Cron[Cron Scheduler]
        end
    end

    subgraph Data["Data Layer"]
        PG[(PostgreSQL)]
        Redis[(Redis)]
        Neo4j[(Neo4j)]
    end

    subgraph Cloud["Cloud Accounts"]
        AWS[AWS]
        Azure[Azure]
        GCP[GCP]
    end

    Users --> LB
    LB --> FE1
    LB --> FE2
    FE1 --> API1
    FE2 --> API2
    API1 --> PG
    API2 --> PG
    API1 --> Redis
    API2 --> Redis
    Worker1 --> Cloud
    Worker2 --> Cloud
    Cron --> Redis
```

## Performance Considerations

### Sampling Strategy

For large buckets with millions of objects, we use intelligent sampling:

```mermaid
flowchart TB
    Start[List Bucket Objects] --> Count{Object Count}
    Count -->|< 1000| Full[Scan All]
    Count -->|1000 - 10000| Sample1[10% Random Sample]
    Count -->|> 10000| Sample2[1000 + Recent Files]

    Full --> Scan[Scan Objects]
    Sample1 --> Scan
    Sample2 --> Scan

    Scan --> Classify[Classify Content]
    Classify --> Store[Store Results]
```

### Parallel Processing

```mermaid
flowchart LR
    Queue[Redis Queue] --> Worker1[Worker 1]
    Queue --> Worker2[Worker 2]
    Queue --> Worker3[Worker 3]
    Queue --> WorkerN[Worker N]

    Worker1 --> Results[(Results)]
    Worker2 --> Results
    Worker3 --> Results
    WorkerN --> Results
```

## Conclusion

Building a DSPM solution requires careful consideration of:

1. **Scalability** - Cloud environments can have millions of objects
2. **Accuracy** - Classification must minimize false positives
3. **Performance** - Scanning should not impact production workloads
4. **Security** - The DSPM itself must be secure
5. **Compliance** - Reports must meet regulatory requirements

This architecture provides a solid foundation for discovering, classifying, and protecting sensitive data across multi-cloud environments.

## Next Steps

To extend this solution, consider adding:

- **Machine Learning Classification** - Use ML models for better accuracy
- **Data Lineage** - Track data flow between systems
- **Risk Scoring** - Quantify risk based on sensitivity and exposure
- **Auto-Remediation** - Automatically fix common issues
- **SIEM Integration** - Feed findings to security operations

---

*Built with Go, React, PostgreSQL, Redis, and Neo4j*
