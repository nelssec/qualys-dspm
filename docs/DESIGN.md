# DSPM (Data Security Posture Management) Solution Design

## Overview

A cloud-native DSPM solution for discovering, classifying, and protecting sensitive data across cloud resources (AWS, Azure, GCP).

## Core Pillars

### 1. Data Discovery Engine
- Scan cloud storage (S3, Azure Blob, GCS)
- Index serverless functions (Lambda, Azure Functions, Cloud Functions)
- Discover databases (RDS, DynamoDB, Cosmos DB, Cloud SQL)
- Track data flows between services

### 2. Data Classification
- PII detection (SSN, email, phone, addresses)
- PHI/healthcare data patterns
- PCI data (credit cards, financial)
- Custom sensitive data patterns (API keys, secrets)
- ML-based classification for unstructured data

### 3. Access & Permission Analysis
- IAM policy evaluation
- Cross-account access mapping
- Public exposure detection
- Service-to-service permissions
- Least privilege analysis

### 4. Risk Assessment
- Encryption status (at-rest, in-transit)
- Misconfiguration detection
- Data residency/sovereignty issues
- Shadow data discovery
- Backup/retention policy gaps

### 5. Compliance Mapping
- GDPR, CCPA, HIPAA, PCI-DSS, SOC2
- Control-to-data mapping
- Automated evidence collection

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         API Gateway                              │
│                        (Go HTTP Server)                          │
├──────────────────────┬──────────────────────────────────────────┤
│                      │                                          │
│   ┌──────────────────▼──────────────────┐                      │
│   │           Control Plane              │                      │
│   │  - Account management                │                      │
│   │  - Scan orchestration                │                      │
│   │  - Policy management                 │                      │
│   │  - Compliance reporting              │                      │
│   └──────────────────┬──────────────────┘                      │
│                      │                                          │
│   ┌──────────────────▼──────────────────┐                      │
│   │         Message Queue (Redis)        │                      │
│   └──────────────────┬──────────────────┘                      │
│                      │                                          │
│   ┌──────────────────▼──────────────────┐                      │
│   │          Scanner Workers             │                      │
│   │  - Asset discovery                   │                      │
│   │  - Content sampling                  │                      │
│   │  - Pattern matching                  │                      │
│   │  - Classification                    │                      │
│   └──────────────────┬──────────────────┘                      │
│                      │                                          │
├─────────────────────────────────────────────────────────────────┤
│   PostgreSQL          Neo4j            Redis         S3        │
│   (metadata)       (access graph)     (cache)     (evidence)   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Foundation
- Cloud account connectivity (assume-role, service principals)
- Asset inventory collection
- Basic storage scanning (S3, Blob, GCS)

### Phase 2: Classification
- Regex-based PII/PCI detection
- Sampling strategy for large datasets
- Classification rules engine

### Phase 3: Access Analysis
- IAM policy parser
- Graph-based access modeling
- Public exposure alerts

### Phase 4: Risk & Compliance
- Risk scoring framework
- Compliance control mapping
- Remediation recommendations

### Phase 5: Advanced
- Data flow tracking
- ML-enhanced classification
- Automated remediation

---

## Tech Stack

| Layer | Technology | Rationale |
|-------|------------|-----------|
| **Language** | Go | High throughput, concurrency, cloud SDK support |
| **API** | Go + Chi router | Lightweight, idiomatic |
| **Workers** | Go routines + Redis | Distributed scanning jobs |
| **Primary DB** | PostgreSQL | Relational + JSONB flexibility |
| **Graph DB** | Neo4j | Access path analysis |
| **Cache/Queue** | Redis | Job queues, caching |
| **Object Store** | S3/MinIO | Scan results, evidence |
| **Cloud SDKs** | aws-sdk-go-v2, azure-sdk-for-go, google-cloud-go |

---

## Related Documentation

- [Cloud Permissions](./PERMISSIONS.md) - Required IAM permissions per cloud provider
- [Data Model](./DATA_MODEL.md) - Database schema and entity relationships
- [Classification Rules](./CLASSIFICATION.md) - Data classification patterns and rules
