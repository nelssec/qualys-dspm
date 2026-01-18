package remediation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// AWSRemediator implements remediation actions for AWS resources
type AWSRemediator struct {
	s3Client  *s3.Client
	kmsClient *kms.Client
	logger    *slog.Logger
}

// NewAWSRemediator creates a new AWS remediator
func NewAWSRemediator(cfg aws.Config, logger *slog.Logger) *AWSRemediator {
	return &AWSRemediator{
		s3Client:  s3.NewFromConfig(cfg),
		kmsClient: kms.NewFromConfig(cfg),
		logger:    logger,
	}
}

// ValidateAction validates that an action can be executed
func (r *AWSRemediator) ValidateAction(ctx context.Context, action *Action) error {
	switch action.ActionType {
	case ActionEnableBucketEncryption:
		bucketName, ok := action.Parameters["bucket_name"].(string)
		if !ok || bucketName == "" {
			return fmt.Errorf("bucket_name parameter is required")
		}
		// Verify bucket exists and is accessible
		_, err := r.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return fmt.Errorf("bucket %s not accessible: %w", bucketName, err)
		}

	case ActionBlockPublicAccess:
		bucketName, ok := action.Parameters["bucket_name"].(string)
		if !ok || bucketName == "" {
			return fmt.Errorf("bucket_name parameter is required")
		}
		_, err := r.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return fmt.Errorf("bucket %s not accessible: %w", bucketName, err)
		}

	case ActionEnableKMSRotation:
		keyID, ok := action.Parameters["key_id"].(string)
		if !ok || keyID == "" {
			return fmt.Errorf("key_id parameter is required")
		}
		// Verify key exists and is a customer-managed key
		output, err := r.kmsClient.DescribeKey(ctx, &kms.DescribeKeyInput{
			KeyId: aws.String(keyID),
		})
		if err != nil {
			return fmt.Errorf("key %s not accessible: %w", keyID, err)
		}
		if output.KeyMetadata.KeyManager != "CUSTOMER" {
			return fmt.Errorf("key rotation can only be enabled for customer-managed keys")
		}

	case ActionRevokePublicACL:
		bucketName, ok := action.Parameters["bucket_name"].(string)
		if !ok || bucketName == "" {
			return fmt.Errorf("bucket_name parameter is required")
		}
		_, err := r.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return fmt.Errorf("bucket %s not accessible: %w", bucketName, err)
		}

	case ActionEnableVersioning:
		bucketName, ok := action.Parameters["bucket_name"].(string)
		if !ok || bucketName == "" {
			return fmt.Errorf("bucket_name parameter is required")
		}
		_, err := r.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return fmt.Errorf("bucket %s not accessible: %w", bucketName, err)
		}

	case ActionEnableLogging:
		bucketName, ok := action.Parameters["bucket_name"].(string)
		if !ok || bucketName == "" {
			return fmt.Errorf("bucket_name parameter is required")
		}
		targetBucket, ok := action.Parameters["target_bucket"].(string)
		if !ok || targetBucket == "" {
			return fmt.Errorf("target_bucket parameter is required")
		}
		// Verify both buckets exist
		_, err := r.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return fmt.Errorf("bucket %s not accessible: %w", bucketName, err)
		}
		_, err = r.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
			Bucket: aws.String(targetBucket),
		})
		if err != nil {
			return fmt.Errorf("target bucket %s not accessible: %w", targetBucket, err)
		}

	default:
		return fmt.Errorf("unsupported action type: %s", action.ActionType)
	}

	return nil
}

// Execute executes a remediation action
func (r *AWSRemediator) Execute(ctx context.Context, action *Action) (*ExecuteResult, error) {
	switch action.ActionType {
	case ActionEnableBucketEncryption:
		return r.enableBucketEncryption(ctx, action)
	case ActionBlockPublicAccess:
		return r.blockPublicAccess(ctx, action)
	case ActionEnableKMSRotation:
		return r.enableKMSRotation(ctx, action)
	case ActionRevokePublicACL:
		return r.revokePublicACL(ctx, action)
	case ActionEnableVersioning:
		return r.enableVersioning(ctx, action)
	case ActionEnableLogging:
		return r.enableLogging(ctx, action)
	default:
		return &ExecuteResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("unsupported action type: %s", action.ActionType),
		}, nil
	}
}

// Rollback rolls back a remediation action
func (r *AWSRemediator) Rollback(ctx context.Context, action *Action) (*RollbackResult, error) {
	switch action.ActionType {
	case ActionBlockPublicAccess:
		return r.rollbackBlockPublicAccess(ctx, action)
	case ActionEnableKMSRotation:
		return r.rollbackKMSRotation(ctx, action)
	case ActionRevokePublicACL:
		return r.rollbackPublicACL(ctx, action)
	case ActionEnableLogging:
		return r.rollbackLogging(ctx, action)
	default:
		return &RollbackResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("rollback not supported for action type: %s", action.ActionType),
		}, nil
	}
}

// enableBucketEncryption enables server-side encryption on an S3 bucket
func (r *AWSRemediator) enableBucketEncryption(ctx context.Context, action *Action) (*ExecuteResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)

	// Get current encryption state
	currentEnc, _ := r.s3Client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	})

	previousState := make(map[string]interface{})
	if currentEnc != nil && currentEnc.ServerSideEncryptionConfiguration != nil {
		previousState["encryption_enabled"] = true
		if len(currentEnc.ServerSideEncryptionConfiguration.Rules) > 0 {
			rule := currentEnc.ServerSideEncryptionConfiguration.Rules[0]
			if rule.ApplyServerSideEncryptionByDefault != nil {
				previousState["algorithm"] = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
				if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != nil {
					previousState["kms_key_id"] = *rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID
				}
			}
		}
	} else {
		previousState["encryption_enabled"] = false
	}

	// Determine encryption settings
	algorithm := s3types.ServerSideEncryptionAes256
	var kmsKeyID *string

	if keyID, ok := action.Parameters["kms_key_id"].(string); ok && keyID != "" {
		algorithm = s3types.ServerSideEncryptionAwsKms
		kmsKeyID = aws.String(keyID)
	}
	if alg, ok := action.Parameters["algorithm"].(string); ok && alg == "aws:kms" {
		algorithm = s3types.ServerSideEncryptionAwsKms
	}

	// Apply encryption
	encryptionConfig := &s3types.ServerSideEncryptionConfiguration{
		Rules: []s3types.ServerSideEncryptionRule{
			{
				ApplyServerSideEncryptionByDefault: &s3types.ServerSideEncryptionByDefault{
					SSEAlgorithm:   algorithm,
					KMSMasterKeyID: kmsKeyID,
				},
				BucketKeyEnabled: aws.Bool(true),
			},
		},
	}

	_, err := r.s3Client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket:                            aws.String(bucketName),
		ServerSideEncryptionConfiguration: encryptionConfig,
	})
	if err != nil {
		return &ExecuteResult{
			Success:       false,
			PreviousState: previousState,
			ErrorMessage:  err.Error(),
		}, nil
	}

	newState := map[string]interface{}{
		"encryption_enabled": true,
		"algorithm":          string(algorithm),
	}
	if kmsKeyID != nil {
		newState["kms_key_id"] = *kmsKeyID
	}

	r.logger.Info("bucket encryption enabled",
		"bucket", bucketName,
		"algorithm", algorithm)

	return &ExecuteResult{
		Success:       true,
		PreviousState: previousState,
		NewState:      newState,
	}, nil
}

// blockPublicAccess enables S3 Block Public Access settings
func (r *AWSRemediator) blockPublicAccess(ctx context.Context, action *Action) (*ExecuteResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)

	// Get current public access block state
	currentPAB, _ := r.s3Client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
	})

	previousState := make(map[string]interface{})
	if currentPAB != nil && currentPAB.PublicAccessBlockConfiguration != nil {
		pab := currentPAB.PublicAccessBlockConfiguration
		previousState["block_public_acls"] = aws.ToBool(pab.BlockPublicAcls)
		previousState["ignore_public_acls"] = aws.ToBool(pab.IgnorePublicAcls)
		previousState["block_public_policy"] = aws.ToBool(pab.BlockPublicPolicy)
		previousState["restrict_public_buckets"] = aws.ToBool(pab.RestrictPublicBuckets)
	}

	// Apply block public access settings
	_, err := r.s3Client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	})
	if err != nil {
		return &ExecuteResult{
			Success:       false,
			PreviousState: previousState,
			ErrorMessage:  err.Error(),
		}, nil
	}

	newState := map[string]interface{}{
		"block_public_acls":       true,
		"ignore_public_acls":      true,
		"block_public_policy":     true,
		"restrict_public_buckets": true,
	}

	r.logger.Info("public access blocked",
		"bucket", bucketName)

	return &ExecuteResult{
		Success:       true,
		PreviousState: previousState,
		NewState:      newState,
	}, nil
}

// rollbackBlockPublicAccess restores previous public access block settings
func (r *AWSRemediator) rollbackBlockPublicAccess(ctx context.Context, action *Action) (*RollbackResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)

	if action.PreviousState == nil {
		// If there was no previous state, remove the block entirely
		_, err := r.s3Client.DeletePublicAccessBlock(ctx, &s3.DeletePublicAccessBlockInput{
			Bucket: aws.String(bucketName),
		})
		if err != nil {
			return &RollbackResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}, nil
		}
	} else {
		// Restore previous settings
		_, err := r.s3Client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
			Bucket: aws.String(bucketName),
			PublicAccessBlockConfiguration: &s3types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       aws.Bool(getBoolParam(action.PreviousState, "block_public_acls")),
				IgnorePublicAcls:      aws.Bool(getBoolParam(action.PreviousState, "ignore_public_acls")),
				BlockPublicPolicy:     aws.Bool(getBoolParam(action.PreviousState, "block_public_policy")),
				RestrictPublicBuckets: aws.Bool(getBoolParam(action.PreviousState, "restrict_public_buckets")),
			},
		})
		if err != nil {
			return &RollbackResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}, nil
		}
	}

	r.logger.Info("public access block rolled back",
		"bucket", bucketName)

	return &RollbackResult{Success: true}, nil
}

// enableKMSRotation enables automatic rotation for a KMS key
func (r *AWSRemediator) enableKMSRotation(ctx context.Context, action *Action) (*ExecuteResult, error) {
	keyID := action.Parameters["key_id"].(string)

	// Get current rotation status
	currentStatus, err := r.kmsClient.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return &ExecuteResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to get rotation status: %s", err.Error()),
		}, nil
	}

	previousState := map[string]interface{}{
		"rotation_enabled": currentStatus.KeyRotationEnabled,
	}

	// Enable rotation
	_, err = r.kmsClient.EnableKeyRotation(ctx, &kms.EnableKeyRotationInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return &ExecuteResult{
			Success:       false,
			PreviousState: previousState,
			ErrorMessage:  err.Error(),
		}, nil
	}

	newState := map[string]interface{}{
		"rotation_enabled":   true,
		"rotation_period":    365,
	}

	r.logger.Info("KMS key rotation enabled",
		"key_id", keyID)

	return &ExecuteResult{
		Success:       true,
		PreviousState: previousState,
		NewState:      newState,
	}, nil
}

// rollbackKMSRotation disables automatic rotation for a KMS key
func (r *AWSRemediator) rollbackKMSRotation(ctx context.Context, action *Action) (*RollbackResult, error) {
	keyID := action.Parameters["key_id"].(string)

	wasEnabled := getBoolParam(action.PreviousState, "rotation_enabled")
	if wasEnabled {
		// Rotation was already enabled, nothing to rollback
		return &RollbackResult{Success: true}, nil
	}

	_, err := r.kmsClient.DisableKeyRotation(ctx, &kms.DisableKeyRotationInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return &RollbackResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	r.logger.Info("KMS key rotation disabled (rolled back)",
		"key_id", keyID)

	return &RollbackResult{Success: true}, nil
}

// revokePublicACL removes public grants from bucket ACL
func (r *AWSRemediator) revokePublicACL(ctx context.Context, action *Action) (*ExecuteResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)

	// Get current ACL
	currentACL, err := r.s3Client.GetBucketAcl(ctx, &s3.GetBucketAclInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return &ExecuteResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to get bucket ACL: %s", err.Error()),
		}, nil
	}

	// Store previous state
	previousGrants := make([]map[string]interface{}, 0)
	for _, grant := range currentACL.Grants {
		g := map[string]interface{}{
			"permission": string(grant.Permission),
		}
		if grant.Grantee != nil {
			g["grantee_type"] = string(grant.Grantee.Type)
			if grant.Grantee.URI != nil {
				g["grantee_uri"] = *grant.Grantee.URI
			}
			if grant.Grantee.ID != nil {
				g["grantee_id"] = *grant.Grantee.ID
			}
		}
		previousGrants = append(previousGrants, g)
	}

	previousState := map[string]interface{}{
		"owner_id": aws.ToString(currentACL.Owner.ID),
		"grants":   previousGrants,
	}

	// Filter out public grants
	var newGrants []s3types.Grant
	for _, grant := range currentACL.Grants {
		if grant.Grantee != nil && grant.Grantee.URI != nil {
			uri := *grant.Grantee.URI
			if uri == "http://acs.amazonaws.com/groups/global/AllUsers" ||
				uri == "http://acs.amazonaws.com/groups/global/AuthenticatedUsers" {
				continue // Skip public grants
			}
		}
		newGrants = append(newGrants, grant)
	}

	// Apply new ACL
	_, err = r.s3Client.PutBucketAcl(ctx, &s3.PutBucketAclInput{
		Bucket: aws.String(bucketName),
		AccessControlPolicy: &s3types.AccessControlPolicy{
			Owner:  currentACL.Owner,
			Grants: newGrants,
		},
	})
	if err != nil {
		return &ExecuteResult{
			Success:       false,
			PreviousState: previousState,
			ErrorMessage:  err.Error(),
		}, nil
	}

	newState := map[string]interface{}{
		"public_grants_removed": true,
		"remaining_grants":      len(newGrants),
	}

	r.logger.Info("public ACL grants revoked",
		"bucket", bucketName)

	return &ExecuteResult{
		Success:       true,
		PreviousState: previousState,
		NewState:      newState,
	}, nil
}

// rollbackPublicACL restores the previous ACL
func (r *AWSRemediator) rollbackPublicACL(ctx context.Context, action *Action) (*RollbackResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)

	if action.PreviousState == nil {
		return &RollbackResult{
			Success:      false,
			ErrorMessage: "no previous state available",
		}, nil
	}

	// Reconstruct the ACL from previous state
	ownerID, _ := action.PreviousState["owner_id"].(string)
	grants, _ := action.PreviousState["grants"].([]interface{})

	var s3Grants []s3types.Grant
	for _, g := range grants {
		gMap := g.(map[string]interface{})
		grant := s3types.Grant{
			Permission: s3types.Permission(gMap["permission"].(string)),
			Grantee: &s3types.Grantee{
				Type: s3types.Type(gMap["grantee_type"].(string)),
			},
		}
		if uri, ok := gMap["grantee_uri"].(string); ok {
			grant.Grantee.URI = aws.String(uri)
		}
		if id, ok := gMap["grantee_id"].(string); ok {
			grant.Grantee.ID = aws.String(id)
		}
		s3Grants = append(s3Grants, grant)
	}

	_, err := r.s3Client.PutBucketAcl(ctx, &s3.PutBucketAclInput{
		Bucket: aws.String(bucketName),
		AccessControlPolicy: &s3types.AccessControlPolicy{
			Owner: &s3types.Owner{
				ID: aws.String(ownerID),
			},
			Grants: s3Grants,
		},
	})
	if err != nil {
		return &RollbackResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	r.logger.Info("bucket ACL rolled back",
		"bucket", bucketName)

	return &RollbackResult{Success: true}, nil
}

// enableVersioning enables versioning on an S3 bucket
func (r *AWSRemediator) enableVersioning(ctx context.Context, action *Action) (*ExecuteResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)

	// Get current versioning state
	currentVersioning, _ := r.s3Client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})

	previousState := map[string]interface{}{
		"status": string(currentVersioning.Status),
	}

	// Enable versioning
	_, err := r.s3Client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: aws.String(bucketName),
		VersioningConfiguration: &s3types.VersioningConfiguration{
			Status: s3types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return &ExecuteResult{
			Success:       false,
			PreviousState: previousState,
			ErrorMessage:  err.Error(),
		}, nil
	}

	newState := map[string]interface{}{
		"status": "Enabled",
	}

	r.logger.Info("bucket versioning enabled",
		"bucket", bucketName)

	return &ExecuteResult{
		Success:       true,
		PreviousState: previousState,
		NewState:      newState,
	}, nil
}

// enableLogging enables access logging for an S3 bucket
func (r *AWSRemediator) enableLogging(ctx context.Context, action *Action) (*ExecuteResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)
	targetBucket := action.Parameters["target_bucket"].(string)
	targetPrefix := ""
	if prefix, ok := action.Parameters["target_prefix"].(string); ok {
		targetPrefix = prefix
	}

	// Get current logging state
	currentLogging, _ := r.s3Client.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{
		Bucket: aws.String(bucketName),
	})

	previousState := make(map[string]interface{})
	if currentLogging != nil && currentLogging.LoggingEnabled != nil {
		previousState["logging_enabled"] = true
		previousState["target_bucket"] = aws.ToString(currentLogging.LoggingEnabled.TargetBucket)
		previousState["target_prefix"] = aws.ToString(currentLogging.LoggingEnabled.TargetPrefix)
	} else {
		previousState["logging_enabled"] = false
	}

	// Enable logging
	_, err := r.s3Client.PutBucketLogging(ctx, &s3.PutBucketLoggingInput{
		Bucket: aws.String(bucketName),
		BucketLoggingStatus: &s3types.BucketLoggingStatus{
			LoggingEnabled: &s3types.LoggingEnabled{
				TargetBucket: aws.String(targetBucket),
				TargetPrefix: aws.String(targetPrefix),
			},
		},
	})
	if err != nil {
		return &ExecuteResult{
			Success:       false,
			PreviousState: previousState,
			ErrorMessage:  err.Error(),
		}, nil
	}

	newState := map[string]interface{}{
		"logging_enabled": true,
		"target_bucket":   targetBucket,
		"target_prefix":   targetPrefix,
	}

	r.logger.Info("bucket logging enabled",
		"bucket", bucketName,
		"target_bucket", targetBucket)

	return &ExecuteResult{
		Success:       true,
		PreviousState: previousState,
		NewState:      newState,
	}, nil
}

// rollbackLogging disables or restores logging configuration
func (r *AWSRemediator) rollbackLogging(ctx context.Context, action *Action) (*RollbackResult, error) {
	bucketName := action.Parameters["bucket_name"].(string)

	if action.PreviousState == nil || !getBoolParam(action.PreviousState, "logging_enabled") {
		// Disable logging
		_, err := r.s3Client.PutBucketLogging(ctx, &s3.PutBucketLoggingInput{
			Bucket:              aws.String(bucketName),
			BucketLoggingStatus: &s3types.BucketLoggingStatus{},
		})
		if err != nil {
			return &RollbackResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}, nil
		}
	} else {
		// Restore previous logging configuration
		targetBucket, _ := action.PreviousState["target_bucket"].(string)
		targetPrefix, _ := action.PreviousState["target_prefix"].(string)

		_, err := r.s3Client.PutBucketLogging(ctx, &s3.PutBucketLoggingInput{
			Bucket: aws.String(bucketName),
			BucketLoggingStatus: &s3types.BucketLoggingStatus{
				LoggingEnabled: &s3types.LoggingEnabled{
					TargetBucket: aws.String(targetBucket),
					TargetPrefix: aws.String(targetPrefix),
				},
			},
		})
		if err != nil {
			return &RollbackResult{
				Success:      false,
				ErrorMessage: err.Error(),
			}, nil
		}
	}

	r.logger.Info("bucket logging rolled back",
		"bucket", bucketName)

	return &RollbackResult{Success: true}, nil
}

// Helper function to get bool parameter from map
func getBoolParam(m map[string]interface{}, key string) bool {
	if m == nil {
		return false
	}
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
