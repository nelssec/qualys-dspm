# Qualys DSPM

Data Security Posture Management for cloud environments. Discover, classify, and protect sensitive data across AWS with an interactive web dashboard.

## What It Does

- **Discovers** data assets (S3 buckets, RDS, DynamoDB)
- **Classifies** sensitive data (PII, PHI, PCI, secrets)
- **Maps** to compliance frameworks (GDPR, HIPAA, PCI-DSS)
- **Tracks** data lineage and AI/ML service exposure
- **Remediates** security issues with one-click fixes

## Quick Deploy (AWS ECS)

### Prerequisites

1. AWS account with ECS and RDS access
2. Docker installed locally
3. AWS CLI configured

### Deploy in 5 Steps

```bash
# 1. Build the image
docker build -t qualys-dspm .

# 2. Push to ECR
aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com
docker tag qualys-dspm:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/qualys-dspm:latest
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/qualys-dspm:latest

# 3. Create RDS PostgreSQL instance (if needed)
aws rds create-db-instance \
  --db-instance-identifier dspm-postgres \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --master-username dspmadmin \
  --master-user-password <password> \
  --allocated-storage 20

# 4. Create ECS service using deploy/ecs-task-definition.json

# 5. Access the dashboard at your ALB endpoint
```

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL hostname | `dspm-postgres.xxx.us-east-1.rds.amazonaws.com` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database username | `dspmadmin` |
| `DB_PASSWORD` | Database password | `your-secure-password` |
| `DB_NAME` | Database name | `dspm` |
| `JWT_SECRET` | JWT signing secret | `random-secure-string` |
| `AWS_REGION` | AWS region | `us-east-1` |

## AWS Permissions Required

The DSPM needs IAM permissions to scan your AWS resources:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets",
        "s3:ListBucket",
        "s3:GetObject",
        "s3:GetBucketEncryption",
        "s3:GetBucketPublicAccessBlock",
        "s3:GetBucketPolicy",
        "s3:GetBucketAcl",
        "kms:ListKeys",
        "kms:DescribeKey",
        "kms:ListAliases",
        "iam:ListRoles",
        "iam:ListPolicies"
      ],
      "Resource": "*"
    }
  ]
}
```

For auto-remediation, add write permissions:
```json
{
  "Action": [
    "s3:PutBucketEncryption",
    "s3:PutPublicAccessBlock",
    "kms:EnableKeyRotation"
  ]
}
```

## Running a Scan

1. **Login** to the dashboard (default: `admin@dspm.local` / `admin123`)
2. **Add Account** - Enter your AWS account ID and cross-account IAM role ARN
3. **Run Scan** - Click "Scan" on any connected account
4. **Review Results** - Check Assets, Findings, and Compliance views

The scan will:
- Discover all S3 buckets in the account
- Sample files for sensitive data (1MB samples for large files)
- Classify content using 12+ detection rules
- Generate findings with severity levels
- Map to compliance frameworks

## Local Development

```bash
# Start dependencies
docker-compose up -d postgres redis

# Run migrations
psql -h localhost -U dspmadmin -d dspm -f migrations/001_initial.sql
psql -h localhost -U dspmadmin -d dspm -f migrations/002_access_policies.sql
psql -h localhost -U dspmadmin -d dspm -f migrations/003_phase2_expansion.sql

# Run the server
go run ./cmd/dspm

# Access at http://localhost:8080
```

## Detection Rules

| Category | Rules |
|----------|-------|
| **PII** | SSN, Email, Phone, Date of Birth, Passport |
| **PCI** | Credit Card (Luhn validated), IBAN, Bank Account |
| **PHI** | Medical Record Number, Health Insurance ID, ICD Codes |
| **Secrets** | AWS Access Key, API Key, JWT Token |

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Web Dashboard                         │
├─────────────────────────────────────────────────────────┤
│                    Go API Server                         │
├──────────────┬──────────────┬──────────────────────────┤
│   Scanner    │  Classifier  │     Remediation          │
├──────────────┴──────────────┴──────────────────────────┤
│                  AWS Connector                           │
│              (S3, KMS, IAM, CloudTrail)                 │
├─────────────────────────────────────────────────────────┤
│                    PostgreSQL                            │
└─────────────────────────────────────────────────────────┘
```

## Technical Documentation

See [docs/OVERVIEW.md](docs/OVERVIEW.md) for detailed architecture, API reference, and feature documentation.

## License

MIT License
