# DSPM - Data Security Posture Management

A comprehensive Data Security Posture Management solution for multi-cloud environments. DSPM helps organizations discover, classify, and protect sensitive data across AWS, Azure, and GCP.

## Features

- **Multi-Cloud Support** - AWS S3, Azure Blob Storage, GCP Cloud Storage
- **Data Discovery** - Automatic scanning of cloud storage assets
- **Data Classification** - PII, PHI, PCI, and secrets detection with regex patterns
- **Custom Rules** - Create your own classification rules with regex patterns
- **Access Analysis** - IAM policy analysis with Neo4j graph database
- **Scheduled Scans** - Cron-based automatic scanning
- **Notifications** - Slack and email alerts for security findings
- **Reports** - PDF and CSV export for compliance reporting
- **JWT Authentication** - Secure API with role-based access control

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     React Dashboard (:3000)                      │
├─────────────────────────────────────────────────────────────────┤
│                      Go API Server (:8080)                       │
├─────────┬─────────────┬─────────────┬───────────────────────────┤
│ Scanner │ Classifier  │ Access Graph│ Queue Workers              │
├─────────┴─────────────┴─────────────┴───────────────────────────┤
│   AWS        Azure        GCP        Connectors                  │
├─────────────────────────────────────────────────────────────────┤
│ PostgreSQL    │    Redis    │    Neo4j                          │
│  (metadata)   │   (queue)   │  (IAM graph)                      │
└───────────────┴─────────────┴───────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+
- Neo4j 5+ (optional, for access graph)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/qualys/dspm.git
   cd dspm
   ```

2. **Start infrastructure**
   ```bash
   docker-compose up -d
   ```

3. **Run database migrations**
   ```bash
   make setup-db
   ```

4. **Configure the application**
   ```bash
   cp config.example.yaml config.yaml
   # Edit config.yaml with your settings
   ```

5. **Start the backend**
   ```bash
   make run
   ```

6. **Start the frontend** (in a new terminal)
   ```bash
   cd web
   npm install
   npm run dev
   ```

7. **Access the dashboard**
   - Frontend: http://localhost:3000
   - API: http://localhost:8080
   - Default login: `admin@dspm.local` / `admin123`

## Configuration

### config.yaml

```yaml
server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

database:
  host: localhost
  port: 5432
  user: dspm
  password: dspm
  database: dspm
  ssl_mode: disable

redis:
  host: localhost
  port: 6379

neo4j:
  uri: bolt://localhost:7687
  user: neo4j
  password: password

auth:
  jwt_secret: your-secret-key-change-in-production
  access_token_expiry: 15m
  refresh_token_expiry: 168h

notifications:
  min_severity: high
  slack:
    enabled: false
    webhook_url: https://hooks.slack.com/services/...
    channel: "#security-alerts"
  email:
    enabled: false
    smtp_host: smtp.example.com
    smtp_port: 587
    username: alerts@example.com
    password: password
    from: alerts@example.com
    to:
      - security@example.com

aws:
  region: us-east-1
  # Use IAM roles or provide credentials
  # access_key_id: ...
  # secret_access_key: ...

azure:
  tenant_id: your-tenant-id
  client_id: your-client-id
  client_secret: your-client-secret
  subscription_id: your-subscription-id

gcp:
  project_id: your-project-id
  credentials_file: /path/to/credentials.json
```

## Cloud Permissions

### AWS IAM Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets",
        "s3:GetBucketLocation",
        "s3:GetBucketPolicy",
        "s3:GetBucketAcl",
        "s3:GetBucketPublicAccessBlock",
        "s3:ListBucket",
        "s3:GetObject"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "iam:GetPolicy",
        "iam:GetPolicyVersion",
        "iam:ListPolicies",
        "iam:ListRoles",
        "iam:ListUsers",
        "iam:GetRole",
        "iam:GetUser"
      ],
      "Resource": "*"
    }
  ]
}
```

### Azure RBAC

Required roles:
- Storage Blob Data Reader
- Reader (subscription level)

### GCP IAM

Required roles:
- `roles/storage.objectViewer`
- `roles/iam.securityReviewer`

## API Reference

### Authentication

```bash
# Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@dspm.local", "password": "admin123"}'

# Use token
curl http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer <access_token>"
```

### Accounts

```bash
# List accounts
GET /api/v1/accounts

# Create account
POST /api/v1/accounts
{
  "provider": "aws",
  "external_id": "123456789012",
  "display_name": "Production AWS",
  "connector_config": {
    "role_arn": "arn:aws:iam::123456789012:role/DSPMRole"
  }
}

# Trigger scan
POST /api/v1/accounts/{id}/scan
{
  "scan_type": "full"
}
```

### Findings

```bash
# List findings
GET /api/v1/findings?severity=critical&status=open

# Update status
PATCH /api/v1/findings/{id}/status
{
  "status": "resolved",
  "reason": "Data has been encrypted"
}
```

### Custom Rules

```bash
# Create rule
POST /api/v1/rules
{
  "name": "Custom SSN Pattern",
  "description": "Detects Social Security Numbers",
  "category": "pii",
  "sensitivity": "high",
  "patterns": ["\\b\\d{3}-\\d{2}-\\d{4}\\b"],
  "context_patterns": ["(?i)ssn|social.security"],
  "context_required": true,
  "enabled": true
}

# Test rule
POST /api/v1/rules/test
{
  "rule": {
    "patterns": ["\\b\\d{3}-\\d{2}-\\d{4}\\b"]
  },
  "content": "SSN: 123-45-6789"
}
```

### Reports

```bash
# Generate PDF report
POST /api/v1/reports/generate
{
  "type": "findings",
  "format": "pdf",
  "title": "Security Findings Report",
  "severities": ["critical", "high"]
}

# Stream CSV export
GET /api/v1/reports/stream?type=assets
```

## Project Structure

```
.
├── cmd/dspm/              # Application entry point
├── internal/
│   ├── api/               # REST API handlers
│   ├── auth/              # JWT authentication
│   ├── access/            # Neo4j access graph
│   ├── classifier/        # Data classification engine
│   ├── config/            # Configuration management
│   ├── connectors/        # Cloud provider connectors
│   │   ├── aws/
│   │   ├── azure/
│   │   └── gcp/
│   ├── models/            # Domain models
│   ├── notifications/     # Slack/email alerts
│   ├── queue/             # Redis job queue
│   ├── reports/           # PDF/CSV generation
│   ├── rules/             # Custom classification rules
│   ├── scanner/           # Scan orchestration
│   ├── scheduler/         # Cron job scheduler
│   └── store/             # PostgreSQL data layer
├── migrations/            # Database migrations
├── web/                   # React frontend
│   ├── src/
│   │   ├── api/           # API client
│   │   ├── components/    # Reusable components
│   │   ├── pages/         # Page components
│   │   └── types/         # TypeScript types
│   └── package.json
├── config.example.yaml
├── docker-compose.yaml
├── Dockerfile
├── Makefile
└── go.mod
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/classifier/...
```

### Building

```bash
# Build binary
make build

# Build Docker image
docker build -t dspm:latest .
```

### Database Migrations

```bash
# Apply migrations
make setup-db

# Or manually
psql -h localhost -U dspm -d dspm -f migrations/001_initial.sql
psql -h localhost -U dspm -d dspm -f migrations/002_auth_scheduler_rules.sql
```

## Classification Categories

| Category | Description | Examples |
|----------|-------------|----------|
| PII | Personally Identifiable Information | SSN, Email, Phone, Address |
| PHI | Protected Health Information | Medical records, Insurance IDs |
| PCI | Payment Card Industry | Credit card numbers, CVV |
| Secrets | Credentials and keys | API keys, passwords, tokens |

## Compliance Frameworks

DSPM helps with compliance for:

- **GDPR** - EU General Data Protection Regulation
- **HIPAA** - Health Insurance Portability and Accountability Act
- **PCI-DSS** - Payment Card Industry Data Security Standard
- **SOC 2** - Service Organization Control 2
- **CCPA** - California Consumer Privacy Act

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Documentation: [docs/](docs/)
- Issues: [GitHub Issues](https://github.com/qualys/dspm/issues)
- Email: support@qualys.com
