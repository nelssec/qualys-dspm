# Cloud Provider Permissions

This document outlines the required permissions for DSPM scanning across AWS, Azure, and GCP.

---

## AWS Permissions

### IAM Policy

Create a role with the following policy for DSPM scanning:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "StorageDiscovery",
      "Effect": "Allow",
      "Action": [
        "s3:ListAllMyBuckets",
        "s3:GetBucketLocation",
        "s3:GetBucketPolicy",
        "s3:GetBucketPolicyStatus",
        "s3:GetBucketPublicAccessBlock",
        "s3:GetAccountPublicAccessBlock",
        "s3:GetBucketAcl",
        "s3:GetBucketEncryption",
        "s3:GetBucketVersioning",
        "s3:GetBucketLogging",
        "s3:ListBucket",
        "s3:GetObject",
        "s3:GetObjectAcl",
        "s3:GetObjectTagging"
      ],
      "Resource": "*"
    },
    {
      "Sid": "ServerlessDiscovery",
      "Effect": "Allow",
      "Action": [
        "lambda:ListFunctions",
        "lambda:GetFunction",
        "lambda:GetFunctionConfiguration",
        "lambda:GetPolicy",
        "lambda:ListTags"
      ],
      "Resource": "*"
    },
    {
      "Sid": "DatabaseDiscovery",
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBInstances",
        "rds:DescribeDBClusters",
        "rds:ListTagsForResource",
        "dynamodb:ListTables",
        "dynamodb:DescribeTable",
        "dynamodb:ListTagsOfResource",
        "redshift:DescribeClusters"
      ],
      "Resource": "*"
    },
    {
      "Sid": "IAMAnalysis",
      "Effect": "Allow",
      "Action": [
        "iam:ListRoles",
        "iam:ListUsers",
        "iam:ListPolicies",
        "iam:GetPolicy",
        "iam:GetPolicyVersion",
        "iam:ListRolePolicies",
        "iam:ListAttachedRolePolicies",
        "iam:GetRolePolicy",
        "iam:ListUserPolicies",
        "iam:ListAttachedUserPolicies",
        "iam:GetUserPolicy",
        "iam:ListGroupsForUser",
        "iam:ListAccessKeys",
        "iam:GetAccessKeyLastUsed"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EncryptionAnalysis",
      "Effect": "Allow",
      "Action": [
        "kms:ListKeys",
        "kms:DescribeKey",
        "kms:GetKeyPolicy",
        "kms:GetKeyRotationStatus",
        "kms:ListAliases"
      ],
      "Resource": "*"
    },
    {
      "Sid": "NetworkExposure",
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeNetworkInterfaces",
        "ec2:DescribeVpcs",
        "ec2:DescribeSubnets"
      ],
      "Resource": "*"
    },
    {
      "Sid": "AuditLogs",
      "Effect": "Allow",
      "Action": [
        "cloudtrail:LookupEvents",
        "cloudtrail:DescribeTrails"
      ],
      "Resource": "*"
    }
  ]
}
```

### Trust Policy (for cross-account access)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::DSPM_ACCOUNT_ID:role/DSPMScannerRole"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "UNIQUE_EXTERNAL_ID"
        }
      }
    }
  ]
}
```

---

## Azure Permissions

### Option 1: Built-in Roles

Assign these roles at subscription or management group level:

- `Reader` - General resource enumeration
- `Storage Blob Data Reader` - Content scanning
- `Key Vault Reader` - Encryption analysis

### Option 2: Custom Role Definition

```json
{
  "Name": "DSPM-Scanner",
  "Description": "Read-only access for DSPM scanning",
  "Actions": [
    "Microsoft.Storage/storageAccounts/read",
    "Microsoft.Storage/storageAccounts/listKeys/action",
    "Microsoft.Storage/storageAccounts/blobServices/read",
    "Microsoft.Storage/storageAccounts/blobServices/containers/read",

    "Microsoft.Web/sites/read",
    "Microsoft.Web/sites/config/read",
    "Microsoft.Web/sites/functions/read",

    "Microsoft.Sql/servers/read",
    "Microsoft.Sql/servers/databases/read",
    "Microsoft.DocumentDB/databaseAccounts/read",

    "Microsoft.Authorization/roleAssignments/read",
    "Microsoft.Authorization/roleDefinitions/read",
    "Microsoft.Authorization/permissions/read",

    "Microsoft.KeyVault/vaults/read",
    "Microsoft.KeyVault/vaults/keys/read",

    "Microsoft.Network/networkSecurityGroups/read",
    "Microsoft.Network/virtualNetworks/read"
  ],
  "DataActions": [
    "Microsoft.Storage/storageAccounts/blobServices/containers/blobs/read"
  ],
  "NotActions": [],
  "NotDataActions": [],
  "AssignableScopes": [
    "/subscriptions/{subscription-id}"
  ]
}
```

### Service Principal Setup

```bash
# Create service principal
az ad sp create-for-rbac --name "DSPM-Scanner" --role "DSPM-Scanner" \
  --scopes /subscriptions/{subscription-id}

# Output will include:
# - appId (client_id)
# - password (client_secret)
# - tenant (tenant_id)
```

---

## GCP Permissions

### Option 1: Predefined Roles

- `roles/viewer` - Project level viewer
- `roles/storage.objectViewer` - Content scanning
- `roles/iam.securityReviewer` - IAM analysis

### Option 2: Custom Role

```yaml
title: "DSPM Scanner"
description: "Read-only access for DSPM scanning"
stage: "GA"
includedPermissions:
  # Storage
  - storage.buckets.list
  - storage.buckets.get
  - storage.buckets.getIamPolicy
  - storage.objects.list
  - storage.objects.get

  # Cloud Functions
  - cloudfunctions.functions.list
  - cloudfunctions.functions.get

  # Cloud Run
  - run.services.list
  - run.services.get

  # Databases
  - cloudsql.instances.list
  - cloudsql.instances.get
  - bigquery.datasets.get
  - bigquery.tables.list
  - bigquery.tables.get
  - spanner.instances.list
  - spanner.databases.list

  # IAM
  - iam.roles.list
  - iam.roles.get
  - iam.serviceAccounts.list
  - iam.serviceAccountKeys.list
  - resourcemanager.projects.getIamPolicy

  # KMS
  - cloudkms.keyRings.list
  - cloudkms.cryptoKeys.list
  - cloudkms.cryptoKeys.getIamPolicy

  # Networking
  - compute.firewalls.list
  - compute.networks.list
  - compute.subnetworks.list
```

### Service Account Setup

```bash
# Create service account
gcloud iam service-accounts create dspm-scanner \
  --display-name="DSPM Scanner"

# Create custom role
gcloud iam roles create dspm_scanner \
  --project=PROJECT_ID \
  --file=dspm-role.yaml

# Bind role to service account
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:dspm-scanner@PROJECT_ID.iam.gserviceaccount.com" \
  --role="projects/PROJECT_ID/roles/dspm_scanner"

# Create key file
gcloud iam service-accounts keys create dspm-key.json \
  --iam-account=dspm-scanner@PROJECT_ID.iam.gserviceaccount.com
```

---

## Permission Summary Matrix

| Capability | AWS | Azure | GCP |
|------------|-----|-------|-----|
| **Storage Discovery** | s3:List*, s3:Get* | Storage Account Reader | storage.buckets.* |
| **Content Scanning** | s3:GetObject | Storage Blob Data Reader | storage.objects.get |
| **Serverless** | lambda:Get*, lambda:List* | Web Sites Reader | cloudfunctions.* |
| **Databases** | rds:Describe*, dynamodb:* | SQL Reader, CosmosDB Reader | cloudsql.*, bigquery.* |
| **IAM Analysis** | iam:List*, iam:Get* | Authorization Reader | iam.*, resourcemanager.* |
| **Encryption** | kms:List*, kms:Describe* | Key Vault Reader | cloudkms.* |
| **Network** | ec2:DescribeSecurity* | Network Reader | compute.firewalls.* |

---

## Security Considerations

1. **Least Privilege**: All permissions are read-only
2. **External ID**: Use external IDs for AWS cross-account roles
3. **Credential Rotation**: Rotate service account keys regularly
4. **Audit Logging**: Enable CloudTrail/Activity Logs to monitor DSPM access
5. **Network Restrictions**: Consider VPC endpoints / Private Link for scanning
