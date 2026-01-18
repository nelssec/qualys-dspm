terraform {
  required_version = ">= 1.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

variable "dspm_external_id" {
  description = "External ID for cross-account role assumption (get from DSPM dashboard)"
  type        = string
}

variable "dspm_account_id" {
  description = "AWS Account ID where DSPM is deployed (or use SaaS account ID)"
  type        = string
  default     = ""
}

variable "role_name" {
  description = "Name for the DSPM scanner role"
  type        = string
  default     = "QualysDSPMScannerRole"
}

variable "scan_all_regions" {
  description = "Allow scanning in all regions"
  type        = bool
  default     = true
}

variable "target_buckets" {
  description = "List of S3 bucket ARNs to scan (empty = all buckets)"
  type        = list(string)
  default     = []
}

data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}

locals {
  dspm_principal = var.dspm_account_id != "" ? var.dspm_account_id : data.aws_caller_identity.current.account_id
  bucket_resources = length(var.target_buckets) > 0 ? var.target_buckets : ["arn:${data.aws_partition.current.partition}:s3:::*"]
}

resource "aws_iam_role" "dspm_scanner" {
  name        = var.role_name
  description = "Cross-account role for Qualys DSPM to scan cloud resources"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Principal = {
          AWS = "arn:${data.aws_partition.current.partition}:iam::${local.dspm_principal}:root"
        }
        Action = "sts:AssumeRole"
        Condition = {
          StringEquals = {
            "sts:ExternalId" = var.dspm_external_id
          }
        }
      }
    ]
  })

  tags = {
    Purpose   = "Qualys DSPM Scanner"
    ManagedBy = "Terraform"
  }
}

resource "aws_iam_role_policy" "dspm_s3_read" {
  name = "DSPM-S3-ReadOnly"
  role = aws_iam_role.dspm_scanner.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ListAllBuckets"
        Effect = "Allow"
        Action = [
          "s3:ListAllMyBuckets",
          "s3:GetBucketLocation",
          "s3:GetBucketTagging",
          "s3:GetBucketPolicy",
          "s3:GetBucketPolicyStatus",
          "s3:GetBucketPublicAccessBlock",
          "s3:GetBucketAcl",
          "s3:GetBucketVersioning",
          "s3:GetEncryptionConfiguration",
          "s3:GetBucketLogging"
        ]
        Resource = "*"
      },
      {
        Sid    = "ReadBucketContents"
        Effect = "Allow"
        Action = [
          "s3:ListBucket",
          "s3:GetObject",
          "s3:GetObjectVersion",
          "s3:GetObjectTagging"
        ]
        Resource = concat(
          local.bucket_resources,
          [for b in local.bucket_resources : "${b}/*"]
        )
      }
    ]
  })
}

resource "aws_iam_role_policy" "dspm_iam_read" {
  name = "DSPM-IAM-ReadOnly"
  role = aws_iam_role.dspm_scanner.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ReadIAMPolicies"
        Effect = "Allow"
        Action = [
          "iam:GetRole",
          "iam:GetRolePolicy",
          "iam:GetPolicy",
          "iam:GetPolicyVersion",
          "iam:ListRoles",
          "iam:ListRolePolicies",
          "iam:ListAttachedRolePolicies",
          "iam:ListPolicies",
          "iam:ListUsers",
          "iam:ListGroups",
          "iam:GetUser",
          "iam:GetGroup",
          "iam:ListGroupsForUser",
          "iam:ListUserPolicies",
          "iam:ListAttachedUserPolicies",
          "iam:ListGroupPolicies",
          "iam:ListAttachedGroupPolicies"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy" "dspm_kms_read" {
  name = "DSPM-KMS-ReadOnly"
  role = aws_iam_role.dspm_scanner.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ReadKMSKeys"
        Effect = "Allow"
        Action = [
          "kms:ListKeys",
          "kms:ListAliases",
          "kms:DescribeKey",
          "kms:GetKeyPolicy",
          "kms:GetKeyRotationStatus"
        ]
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy" "dspm_ai_services_read" {
  name = "DSPM-AIServices-ReadOnly"
  role = aws_iam_role.dspm_scanner.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "ReadSageMaker"
        Effect = "Allow"
        Action = [
          "sagemaker:ListModels",
          "sagemaker:DescribeModel",
          "sagemaker:ListTrainingJobs",
          "sagemaker:DescribeTrainingJob",
          "sagemaker:ListEndpoints",
          "sagemaker:DescribeEndpoint"
        ]
        Resource = "*"
      },
      {
        Sid    = "ReadBedrock"
        Effect = "Allow"
        Action = [
          "bedrock:ListCustomModels",
          "bedrock:GetCustomModel",
          "bedrock:ListModelCustomizationJobs"
        ]
        Resource = "*"
      }
    ]
  })
}

output "role_arn" {
  description = "ARN of the DSPM scanner role - add this to your DSPM configuration"
  value       = aws_iam_role.dspm_scanner.arn
}

output "external_id" {
  description = "External ID for role assumption"
  value       = var.dspm_external_id
  sensitive   = true
}

output "next_steps" {
  description = "Instructions for completing setup"
  value       = <<-EOT

    AWS Connector Created Successfully

    Add this account to DSPM:

    1. Go to DSPM Dashboard, Accounts, Add Account
    2. Select "AWS"
    3. Enter:
       - Account ID: ${data.aws_caller_identity.current.account_id}
       - Role ARN: ${aws_iam_role.dspm_scanner.arn}
       - External ID: (the value you provided)

    4. Click "Test Connection" then "Save"

  EOT
}
