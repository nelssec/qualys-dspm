package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudtrail"
	cloudtrailTypes "github.com/aws/aws-sdk-go-v2/service/cloudtrail/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sagemaker"
	sagemakerTypes "github.com/aws/aws-sdk-go-v2/service/sagemaker/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/qualys/dspm/internal/connectors"
	"github.com/qualys/dspm/internal/models"
)

type Connector struct {
	cfg       aws.Config
	accountID string
	region    string

	s3Client         *s3.Client
	iamClient        *iam.Client
	lambdaClient     *lambda.Client
	kmsClient        *kms.Client
	sagemakerClient  *sagemaker.Client
	bedrockClient    *bedrock.Client
	cloudtrailClient *cloudtrail.Client
}

type Config struct {
	Region          string
	AssumeRoleARN   string
	ExternalID      string
	AccessKeyID     string
	SecretAccessKey string
}

func New(ctx context.Context, cfg Config) (*Connector, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	if cfg.AssumeRoleARN != "" {
		stsClient := sts.NewFromConfig(awsCfg)
		creds := stscreds.NewAssumeRoleProvider(stsClient, cfg.AssumeRoleARN, func(o *stscreds.AssumeRoleOptions) {
			if cfg.ExternalID != "" {
				o.ExternalID = aws.String(cfg.ExternalID)
			}
		})
		awsCfg.Credentials = aws.NewCredentialsCache(creds)
	}

	stsClient := sts.NewFromConfig(awsCfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("getting caller identity: %w", err)
	}

	return &Connector{
		cfg:              awsCfg,
		accountID:        aws.ToString(identity.Account),
		region:           cfg.Region,
		s3Client:         s3.NewFromConfig(awsCfg),
		iamClient:        iam.NewFromConfig(awsCfg),
		lambdaClient:     lambda.NewFromConfig(awsCfg),
		kmsClient:        kms.NewFromConfig(awsCfg),
		sagemakerClient:  sagemaker.NewFromConfig(awsCfg),
		bedrockClient:    bedrock.NewFromConfig(awsCfg),
		cloudtrailClient: cloudtrail.NewFromConfig(awsCfg),
	}, nil
}

func (c *Connector) Provider() models.Provider {
	return models.ProviderAWS
}

func (c *Connector) AccountID() string {
	return c.accountID
}

func (c *Connector) Validate(ctx context.Context) error {
	_, err := c.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("validating S3 access: %w", err)
	}

	_, err = c.iamClient.ListRoles(ctx, &iam.ListRolesInput{MaxItems: aws.Int32(1)})
	if err != nil {
		return fmt.Errorf("validating IAM access: %w", err)
	}

	return nil
}

func (c *Connector) Close() error {
	return nil
}

func (c *Connector) S3ClientForRegion(region string) *s3.Client {
	if region == c.region || region == "" {
		return c.s3Client
	}
	return s3.NewFromConfig(c.cfg, func(o *s3.Options) {
		o.Region = region
	})
}

func (c *Connector) ListBuckets(ctx context.Context) ([]connectors.BucketInfo, error) {
	output, err := c.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("listing buckets: %w", err)
	}

	buckets := make([]connectors.BucketInfo, 0, len(output.Buckets))
	for _, b := range output.Buckets {
		locOutput, err := c.s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: b.Name,
		})

		region := "us-east-1" // default
		if err == nil && locOutput.LocationConstraint != "" {
			region = string(locOutput.LocationConstraint)
		}

		buckets = append(buckets, connectors.BucketInfo{
			Name:      aws.ToString(b.Name),
			Region:    region,
			CreatedAt: b.CreationDate.String(),
			ARN:       fmt.Sprintf("arn:aws:s3:::%s", aws.ToString(b.Name)),
		})
	}

	return buckets, nil
}

func (c *Connector) GetBucketMetadata(ctx context.Context, bucketName string) (*connectors.BucketMetadata, error) {
	locOutput, _ := c.s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})

	region := "us-east-1"
	if locOutput != nil && locOutput.LocationConstraint != "" {
		region = string(locOutput.LocationConstraint)
	}

	client := c.S3ClientForRegion(region)

	metadata := &connectors.BucketMetadata{
		Name:   bucketName,
		Region: region,
		ARN:    fmt.Sprintf("arn:aws:s3:::%s", bucketName),
	}

	encOutput, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && encOutput.ServerSideEncryptionConfiguration != nil {
		for _, rule := range encOutput.ServerSideEncryptionConfiguration.Rules {
			if rule.ApplyServerSideEncryptionByDefault != nil {
				metadata.Encryption.Enabled = true
				metadata.Encryption.Algorithm = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
				if rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID != nil {
					metadata.Encryption.KeyARN = aws.ToString(rule.ApplyServerSideEncryptionByDefault.KMSMasterKeyID)
					metadata.Encryption.Type = models.EncryptionSSEKMS
				} else {
					metadata.Encryption.Type = models.EncryptionSSE
				}
			}
		}
	}

	verOutput, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		metadata.Versioning = verOutput.Status == "Enabled"
	}

	logOutput, err := client.GetBucketLogging(ctx, &s3.GetBucketLoggingInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && logOutput.LoggingEnabled != nil {
		metadata.Logging.Enabled = true
		metadata.Logging.TargetBucket = aws.ToString(logOutput.LoggingEnabled.TargetBucket)
		metadata.Logging.TargetPrefix = aws.ToString(logOutput.LoggingEnabled.TargetPrefix)
	}

	pabOutput, err := client.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && pabOutput.PublicAccessBlockConfiguration != nil {
		pab := pabOutput.PublicAccessBlockConfiguration
		metadata.PublicAccessBlock = connectors.PublicAccessBlockConfig{
			BlockPublicAcls:       aws.ToBool(pab.BlockPublicAcls),
			IgnorePublicAcls:      aws.ToBool(pab.IgnorePublicAcls),
			BlockPublicPolicy:     aws.ToBool(pab.BlockPublicPolicy),
			RestrictPublicBuckets: aws.ToBool(pab.RestrictPublicBuckets),
		}
	}

	tagOutput, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil {
		metadata.Tags = make(map[string]string)
		for _, tag := range tagOutput.TagSet {
			metadata.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return metadata, nil
}

func (c *Connector) ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]connectors.ObjectInfo, error) {
	locOutput, _ := c.s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})

	region := "us-east-1"
	if locOutput != nil && locOutput.LocationConstraint != "" {
		region = string(locOutput.LocationConstraint)
	}

	client := c.S3ClientForRegion(region)

	var objects []connectors.ObjectInfo
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:  aws.String(bucketName),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int32(int32(maxKeys)),
	})

	for paginator.HasMorePages() && len(objects) < maxKeys {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing objects: %w", err)
		}

		for _, obj := range page.Contents {
			if len(objects) >= maxKeys {
				break
			}
			objects = append(objects, connectors.ObjectInfo{
				Key:          aws.ToString(obj.Key),
				Size:         aws.ToInt64(obj.Size),
				LastModified: obj.LastModified.String(),
				StorageClass: string(obj.StorageClass),
				ETag:         aws.ToString(obj.ETag),
			})
		}
	}

	return objects, nil
}

func (c *Connector) GetObject(ctx context.Context, bucketName, objectKey string, byteRange *connectors.ByteRange) (io.ReadCloser, error) {
	locOutput, _ := c.s3Client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucketName),
	})

	region := "us-east-1"
	if locOutput != nil && locOutput.LocationConstraint != "" {
		region = string(locOutput.LocationConstraint)
	}

	client := c.S3ClientForRegion(region)

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	if byteRange != nil {
		input.Range = aws.String(fmt.Sprintf("bytes=%d-%d", byteRange.Start, byteRange.End))
	}

	output, err := client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	return output.Body, nil
}

func (c *Connector) GetBucketPolicy(ctx context.Context, bucketName string) (*connectors.BucketPolicy, error) {
	output, err := c.s3Client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, err // No policy is a valid state
	}

	policy := &connectors.BucketPolicy{
		Policy: aws.ToString(output.Policy),
	}

	var doc connectors.PolicyDocument
	if err := json.Unmarshal([]byte(policy.Policy), &doc); err == nil {
		policy.PolicyDocument = &doc
		for _, stmt := range doc.Statements {
			for _, principal := range stmt.Principals {
				if principal == "*" && stmt.Effect == "Allow" {
					policy.IsPublic = true
					policy.PublicActions = append(policy.PublicActions, stmt.Actions...)
				}
			}
		}
	}

	return policy, nil
}

func (c *Connector) GetBucketACL(ctx context.Context, bucketName string) (*connectors.BucketACL, error) {
	output, err := c.s3Client.GetBucketAcl(ctx, &s3.GetBucketAclInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, fmt.Errorf("getting bucket ACL: %w", err)
	}

	acl := &connectors.BucketACL{
		Owner: aws.ToString(output.Owner.DisplayName),
	}

	for _, grant := range output.Grants {
		g := connectors.ACLGrant{
			Permission: string(grant.Permission),
		}

		if grant.Grantee != nil {
			g.GranteeType = string(grant.Grantee.Type)
			if grant.Grantee.URI != nil {
				g.Grantee = aws.ToString(grant.Grantee.URI)
				if *grant.Grantee.URI == "http://acs.amazonaws.com/groups/global/AllUsers" ||
					*grant.Grantee.URI == "http://acs.amazonaws.com/groups/global/AuthenticatedUsers" {
					g.IsPublic = true
				}
			} else if grant.Grantee.ID != nil {
				g.Grantee = aws.ToString(grant.Grantee.ID)
			}
		}

		acl.Grants = append(acl.Grants, g)
	}

	return acl, nil
}

func (c *Connector) ListUsers(ctx context.Context) ([]connectors.Principal, error) {
	var users []connectors.Principal
	paginator := iam.NewListUsersPaginator(c.iamClient, &iam.ListUsersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing users: %w", err)
		}

		for _, user := range page.Users {
			users = append(users, connectors.Principal{
				ARN:       aws.ToString(user.Arn),
				Name:      aws.ToString(user.UserName),
				Type:      "USER",
				CreatedAt: user.CreateDate.String(),
			})
		}
	}

	return users, nil
}

func (c *Connector) ListRoles(ctx context.Context) ([]connectors.Principal, error) {
	var roles []connectors.Principal
	paginator := iam.NewListRolesPaginator(c.iamClient, &iam.ListRolesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing roles: %w", err)
		}

		for _, role := range page.Roles {
			roles = append(roles, connectors.Principal{
				ARN:         aws.ToString(role.Arn),
				Name:        aws.ToString(role.RoleName),
				Type:        "ROLE",
				CreatedAt:   role.CreateDate.String(),
				Description: aws.ToString(role.Description),
			})
		}
	}

	return roles, nil
}

func (c *Connector) ListPolicies(ctx context.Context) ([]connectors.PolicyInfo, error) {
	var policies []connectors.PolicyInfo
	paginator := iam.NewListPoliciesPaginator(c.iamClient, &iam.ListPoliciesInput{
		Scope: "Local", // Only customer-managed policies
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing policies: %w", err)
		}

		for _, policy := range page.Policies {
			policies = append(policies, connectors.PolicyInfo{
				ARN:         aws.ToString(policy.Arn),
				Name:        aws.ToString(policy.PolicyName),
				Type:        "MANAGED",
				Description: aws.ToString(policy.Description),
				IsAttached:  aws.ToInt32(policy.AttachmentCount) > 0,
				AttachCount: int(aws.ToInt32(policy.AttachmentCount)),
			})
		}
	}

	return policies, nil
}

func (c *Connector) GetPolicy(ctx context.Context, policyARN string) (*connectors.PolicyDocument, error) {
	policyOutput, err := c.iamClient.GetPolicy(ctx, &iam.GetPolicyInput{
		PolicyArn: aws.String(policyARN),
	})
	if err != nil {
		return nil, fmt.Errorf("getting policy: %w", err)
	}

	versionOutput, err := c.iamClient.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
		PolicyArn: aws.String(policyARN),
		VersionId: policyOutput.Policy.DefaultVersionId,
	})
	if err != nil {
		return nil, fmt.Errorf("getting policy version: %w", err)
	}

	doc := &connectors.PolicyDocument{
		Raw: aws.ToString(versionOutput.PolicyVersion.Document),
	}

	if err := json.Unmarshal([]byte(doc.Raw), doc); err != nil {
		return doc, nil
	}

	return doc, nil
}

func (c *Connector) ListAttachedPolicies(ctx context.Context, principalARN string) ([]connectors.PolicyInfo, error) {
	var policies []connectors.PolicyInfo

	paginator := iam.NewListAttachedRolePoliciesPaginator(c.iamClient, &iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(principalARN), // This should be parsed from ARN
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing attached policies: %w", err)
		}

		for _, policy := range page.AttachedPolicies {
			policies = append(policies, connectors.PolicyInfo{
				ARN:        aws.ToString(policy.PolicyArn),
				Name:       aws.ToString(policy.PolicyName),
				Type:       "MANAGED",
				IsAttached: true,
			})
		}
	}

	return policies, nil
}

func (c *Connector) GetServiceAccounts(ctx context.Context) ([]connectors.Principal, error) {
	var serviceRoles []connectors.Principal
	paginator := iam.NewListRolesPaginator(c.iamClient, &iam.ListRolesInput{
		PathPrefix: aws.String("/aws-service-role/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing service roles: %w", err)
		}

		for _, role := range page.Roles {
			serviceRoles = append(serviceRoles, connectors.Principal{
				ARN:         aws.ToString(role.Arn),
				Name:        aws.ToString(role.RoleName),
				Type:        "SERVICE",
				CreatedAt:   role.CreateDate.String(),
				Description: aws.ToString(role.Description),
			})
		}
	}

	return serviceRoles, nil
}

func (c *Connector) ListFunctions(ctx context.Context) ([]connectors.FunctionInfo, error) {
	var functions []connectors.FunctionInfo
	paginator := lambda.NewListFunctionsPaginator(c.lambdaClient, &lambda.ListFunctionsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing functions: %w", err)
		}

		for _, fn := range page.Functions {
			functions = append(functions, connectors.FunctionInfo{
				ARN:          aws.ToString(fn.FunctionArn),
				Name:         aws.ToString(fn.FunctionName),
				Runtime:      string(fn.Runtime),
				Handler:      aws.ToString(fn.Handler),
				MemorySize:   int(aws.ToInt32(fn.MemorySize)),
				Timeout:      int(aws.ToInt32(fn.Timeout)),
				LastModified: aws.ToString(fn.LastModified),
			})
		}
	}

	return functions, nil
}

func (c *Connector) GetFunctionConfig(ctx context.Context, functionName string) (*connectors.FunctionConfig, error) {
	output, err := c.lambdaClient.GetFunction(ctx, &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return nil, fmt.Errorf("getting function: %w", err)
	}

	cfg := output.Configuration
	fnConfig := &connectors.FunctionConfig{
		FunctionInfo: connectors.FunctionInfo{
			ARN:          aws.ToString(cfg.FunctionArn),
			Name:         aws.ToString(cfg.FunctionName),
			Runtime:      string(cfg.Runtime),
			Handler:      aws.ToString(cfg.Handler),
			MemorySize:   int(aws.ToInt32(cfg.MemorySize)),
			Timeout:      int(aws.ToInt32(cfg.Timeout)),
			LastModified: aws.ToString(cfg.LastModified),
		},
		Role:      aws.ToString(cfg.Role),
		KMSKeyARN: aws.ToString(cfg.KMSKeyArn),
	}

	if cfg.Environment != nil {
		fnConfig.Environment = cfg.Environment.Variables
	}

	if cfg.VpcConfig != nil {
		fnConfig.VPCConfig = &connectors.VPCConfig{
			SubnetIDs:        cfg.VpcConfig.SubnetIds,
			SecurityGroupIDs: cfg.VpcConfig.SecurityGroupIds,
			VPCID:            aws.ToString(cfg.VpcConfig.VpcId),
		}
	}

	for _, layer := range cfg.Layers {
		fnConfig.Layers = append(fnConfig.Layers, aws.ToString(layer.Arn))
	}

	return fnConfig, nil
}

func (c *Connector) GetFunctionPolicy(ctx context.Context, functionName string) (*connectors.PolicyDocument, error) {
	output, err := c.lambdaClient.GetPolicy(ctx, &lambda.GetPolicyInput{
		FunctionName: aws.String(functionName),
	})
	if err != nil {
		return nil, err // No policy is valid
	}

	doc := &connectors.PolicyDocument{
		Raw: aws.ToString(output.Policy),
	}

	_ = json.Unmarshal([]byte(doc.Raw), doc)
	return doc, nil
}

func (c *Connector) ListKeys(ctx context.Context) ([]connectors.KeyInfo, error) {
	var keys []connectors.KeyInfo
	paginator := kms.NewListKeysPaginator(c.kmsClient, &kms.ListKeysInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing keys: %w", err)
		}

		for _, key := range page.Keys {
			keys = append(keys, connectors.KeyInfo{
				ID:  aws.ToString(key.KeyId),
				ARN: aws.ToString(key.KeyArn),
			})
		}
	}

	return keys, nil
}

func (c *Connector) GetKeyMetadata(ctx context.Context, keyID string) (*connectors.KeyMetadata, error) {
	output, err := c.kmsClient.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return nil, fmt.Errorf("describing key: %w", err)
	}

	km := output.KeyMetadata
	metadata := &connectors.KeyMetadata{
		KeyInfo: connectors.KeyInfo{
			ID:          aws.ToString(km.KeyId),
			ARN:         aws.ToString(km.Arn),
			Description: aws.ToString(km.Description),
			Enabled:     km.Enabled,
			KeyManager:  string(km.KeyManager),
		},
		CreatedAt: km.CreationDate.String(),
		KeyState:  string(km.KeyState),
		KeyUsage:  string(km.KeyUsage),
		Origin:    string(km.Origin),
	}

	rotOutput, err := c.kmsClient.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{
		KeyId: aws.String(keyID),
	})
	if err == nil {
		metadata.RotationEnabled = rotOutput.KeyRotationEnabled
	}

	return metadata, nil
}

func (c *Connector) GetKeyPolicy(ctx context.Context, keyID string) (*connectors.PolicyDocument, error) {
	output, err := c.kmsClient.GetKeyPolicy(ctx, &kms.GetKeyPolicyInput{
		KeyId:      aws.String(keyID),
		PolicyName: aws.String("default"),
	})
	if err != nil {
		return nil, fmt.Errorf("getting key policy: %w", err)
	}

	doc := &connectors.PolicyDocument{
		Raw: aws.ToString(output.Policy),
	}

	_ = json.Unmarshal([]byte(doc.Raw), doc)
	return doc, nil
}

// =====================================================
// Lineage Connector Implementation
// =====================================================

func (c *Connector) GetFunctionEventSources(ctx context.Context, functionName string) ([]connectors.EventSourceMapping, error) {
	var mappings []connectors.EventSourceMapping

	paginator := lambda.NewListEventSourceMappingsPaginator(c.lambdaClient, &lambda.ListEventSourceMappingsInput{
		FunctionName: aws.String(functionName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing event source mappings: %w", err)
		}

		for _, esm := range page.EventSourceMappings {
			mappings = append(mappings, connectors.EventSourceMapping{
				UUID:             aws.ToString(esm.UUID),
				EventSourceARN:   aws.ToString(esm.EventSourceArn),
				FunctionARN:      aws.ToString(esm.FunctionArn),
				BatchSize:        int(aws.ToInt32(esm.BatchSize)),
				StartingPosition: string(esm.StartingPosition),
				State:            aws.ToString(esm.State),
				StateReason:      aws.ToString(esm.StateTransitionReason),
				LastModified:     esm.LastModified.String(),
			})
		}
	}

	return mappings, nil
}

func (c *Connector) GetReplicationConfigurations(ctx context.Context, bucketName string) ([]connectors.ReplicationConfig, error) {
	output, err := c.s3Client.GetBucketReplication(ctx, &s3.GetBucketReplicationInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		// No replication configuration is a valid state
		return nil, nil
	}

	var configs []connectors.ReplicationConfig
	if output.ReplicationConfiguration != nil {
		for _, rule := range output.ReplicationConfiguration.Rules {
			config := connectors.ReplicationConfig{
				ID:           aws.ToString(rule.ID),
				Status:       string(rule.Status),
				Priority:     int(aws.ToInt32(rule.Priority)),
				SourceBucket: bucketName,
			}

			if rule.Destination != nil {
				config.DestinationBucket = aws.ToString(rule.Destination.Bucket)
				config.DestinationAccount = aws.ToString(rule.Destination.Account)
				config.StorageClass = string(rule.Destination.StorageClass)
				if rule.Destination.EncryptionConfiguration != nil {
					config.ReplicaKMSKeyID = aws.ToString(rule.Destination.EncryptionConfiguration.ReplicaKmsKeyID)
				}
			}

			// Filter can have different structures based on SDK version
			// Simplified handling for compatibility

			configs = append(configs, config)
		}
	}

	return configs, nil
}

func (c *Connector) GetDataExportConfigs(ctx context.Context, bucketName string) ([]connectors.DataExportConfig, error) {
	var configs []connectors.DataExportConfig

	// Get inventory configurations
	invOutput, err := c.s3Client.ListBucketInventoryConfigurations(ctx, &s3.ListBucketInventoryConfigurationsInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && invOutput.InventoryConfigurationList != nil {
		for _, inv := range invOutput.InventoryConfigurationList {
			status := "DISABLED"
			if aws.ToBool(inv.IsEnabled) {
				status = "ENABLED"
			}
			config := connectors.DataExportConfig{
				ID:         aws.ToString(inv.Id),
				ExportType: "INVENTORY",
				Status:     status,
				Frequency:  string(inv.Schedule.Frequency),
				Format:     string(inv.Destination.S3BucketDestination.Format),
			}

			if inv.Destination.S3BucketDestination != nil {
				config.DestinationARN = aws.ToString(inv.Destination.S3BucketDestination.Bucket)
				config.Prefix = aws.ToString(inv.Destination.S3BucketDestination.Prefix)
				config.Encryption = inv.Destination.S3BucketDestination.Encryption != nil
			}

			configs = append(configs, config)
		}
	}

	// Get analytics configurations
	anOutput, err := c.s3Client.ListBucketAnalyticsConfigurations(ctx, &s3.ListBucketAnalyticsConfigurationsInput{
		Bucket: aws.String(bucketName),
	})
	if err == nil && anOutput.AnalyticsConfigurationList != nil {
		for _, an := range anOutput.AnalyticsConfigurationList {
			config := connectors.DataExportConfig{
				ID:         aws.ToString(an.Id),
				ExportType: "ANALYTICS",
				Status:     "ENABLED",
			}

			if an.StorageClassAnalysis != nil && an.StorageClassAnalysis.DataExport != nil {
				de := an.StorageClassAnalysis.DataExport
				if de.Destination != nil && de.Destination.S3BucketDestination != nil {
					config.DestinationARN = aws.ToString(de.Destination.S3BucketDestination.Bucket)
					config.Format = string(de.Destination.S3BucketDestination.Format)
					config.Prefix = aws.ToString(de.Destination.S3BucketDestination.Prefix)
				}
			}

			configs = append(configs, config)
		}
	}

	return configs, nil
}

// =====================================================
// AI Connector Implementation (SageMaker)
// =====================================================

func (c *Connector) ListSageMakerModels(ctx context.Context) ([]connectors.SageMakerModel, error) {
	var models []connectors.SageMakerModel

	paginator := sagemaker.NewListModelsPaginator(c.sagemakerClient, &sagemaker.ListModelsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing SageMaker models: %w", err)
		}

		for _, m := range page.Models {
			models = append(models, connectors.SageMakerModel{
				ModelARN:     aws.ToString(m.ModelArn),
				ModelName:    aws.ToString(m.ModelName),
				CreationTime: m.CreationTime.String(),
			})
		}
	}

	return models, nil
}

func (c *Connector) GetSageMakerModelDetails(ctx context.Context, modelName string) (*connectors.SageMakerModelDetails, error) {
	output, err := c.sagemakerClient.DescribeModel(ctx, &sagemaker.DescribeModelInput{
		ModelName: aws.String(modelName),
	})
	if err != nil {
		return nil, fmt.Errorf("describing SageMaker model: %w", err)
	}

	details := &connectors.SageMakerModelDetails{
		SageMakerModel: connectors.SageMakerModel{
			ModelARN:         aws.ToString(output.ModelArn),
			ModelName:        aws.ToString(output.ModelName),
			CreationTime:     output.CreationTime.String(),
			ExecutionRoleARN: aws.ToString(output.ExecutionRoleArn),
		},
		EnableNetworkIsolation: aws.ToBool(output.EnableNetworkIsolation),
	}

	if output.PrimaryContainer != nil {
		details.PrimaryContainer = connectors.ContainerDefinition{
			Image:        aws.ToString(output.PrimaryContainer.Image),
			Mode:         string(output.PrimaryContainer.Mode),
			ModelDataURL: aws.ToString(output.PrimaryContainer.ModelDataUrl),
			Environment:  output.PrimaryContainer.Environment,
		}
	}

	if output.VpcConfig != nil {
		details.VPCConfig = &connectors.VPCConfig{
			SubnetIDs:        output.VpcConfig.Subnets,
			SecurityGroupIDs: output.VpcConfig.SecurityGroupIds,
		}
	}

	for _, container := range output.Containers {
		details.Containers = append(details.Containers, connectors.ContainerDefinition{
			Image:        aws.ToString(container.Image),
			Mode:         string(container.Mode),
			ModelDataURL: aws.ToString(container.ModelDataUrl),
			Environment:  container.Environment,
		})
	}

	// Get tags
	tagsOutput, err := c.sagemakerClient.ListTags(ctx, &sagemaker.ListTagsInput{
		ResourceArn: output.ModelArn,
	})
	if err == nil {
		details.Tags = make(map[string]string)
		for _, tag := range tagsOutput.Tags {
			details.Tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
	}

	return details, nil
}

func (c *Connector) ListTrainingJobs(ctx context.Context) ([]connectors.TrainingJob, error) {
	var jobs []connectors.TrainingJob

	paginator := sagemaker.NewListTrainingJobsPaginator(c.sagemakerClient, &sagemaker.ListTrainingJobsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing training jobs: %w", err)
		}

		for _, job := range page.TrainingJobSummaries {
			jobs = append(jobs, connectors.TrainingJob{
				TrainingJobARN:    aws.ToString(job.TrainingJobArn),
				TrainingJobName:   aws.ToString(job.TrainingJobName),
				TrainingJobStatus: string(job.TrainingJobStatus),
				CreationTime:      job.CreationTime.String(),
				LastModifiedTime:  job.LastModifiedTime.String(),
			})
		}
	}

	return jobs, nil
}

func (c *Connector) GetTrainingJobDetails(ctx context.Context, jobName string) (*connectors.TrainingJobDetails, error) {
	output, err := c.sagemakerClient.DescribeTrainingJob(ctx, &sagemaker.DescribeTrainingJobInput{
		TrainingJobName: aws.String(jobName),
	})
	if err != nil {
		return nil, fmt.Errorf("describing training job: %w", err)
	}

	details := &connectors.TrainingJobDetails{
		TrainingJob: connectors.TrainingJob{
			TrainingJobARN:    aws.ToString(output.TrainingJobArn),
			TrainingJobName:   aws.ToString(output.TrainingJobName),
			TrainingJobStatus: string(output.TrainingJobStatus),
			CreationTime:      output.CreationTime.String(),
			RoleARN:           aws.ToString(output.RoleArn),
		},
		HyperParameters: output.HyperParameters,
	}

	if output.LastModifiedTime != nil {
		details.LastModifiedTime = output.LastModifiedTime.String()
	}
	if output.TrainingStartTime != nil {
		details.TrainingStartTime = output.TrainingStartTime.String()
	}
	if output.TrainingEndTime != nil {
		details.TrainingEndTime = output.TrainingEndTime.String()
	}
	if output.ModelArtifacts != nil {
		details.ModelArtifacts = aws.ToString(output.ModelArtifacts.S3ModelArtifacts)
	}
	if output.BillableTimeInSeconds != nil {
		details.BillableTimeSeconds = int(*output.BillableTimeInSeconds)
	}

	// Input data configuration
	for _, channel := range output.InputDataConfig {
		dataChannel := connectors.DataChannelConfig{
			ChannelName:     aws.ToString(channel.ChannelName),
			ContentType:     aws.ToString(channel.ContentType),
			CompressionType: string(channel.CompressionType),
			RecordWrapper:   string(channel.RecordWrapperType),
			InputMode:       string(channel.InputMode),
		}

		if channel.DataSource != nil && channel.DataSource.S3DataSource != nil {
			dataChannel.S3DataSource = &connectors.S3DataSourceConfig{
				S3URI:              aws.ToString(channel.DataSource.S3DataSource.S3Uri),
				S3DataType:         string(channel.DataSource.S3DataSource.S3DataType),
				S3DataDistribution: string(channel.DataSource.S3DataSource.S3DataDistributionType),
			}
		}

		details.InputDataConfig = append(details.InputDataConfig, dataChannel)
	}

	// Output data configuration
	if output.OutputDataConfig != nil {
		details.OutputDataConfig = connectors.OutputDataConfig{
			S3OutputPath: aws.ToString(output.OutputDataConfig.S3OutputPath),
			KMSKeyID:     aws.ToString(output.OutputDataConfig.KmsKeyId),
		}
	}

	// Resource configuration
	if output.ResourceConfig != nil {
		details.ResourceConfig = connectors.ResourceConfig{
			InstanceType:   string(output.ResourceConfig.InstanceType),
			InstanceCount:  int(aws.ToInt32(output.ResourceConfig.InstanceCount)),
			VolumeSizeGB:   int(aws.ToInt32(output.ResourceConfig.VolumeSizeInGB)),
			VolumeKmsKeyID: aws.ToString(output.ResourceConfig.VolumeKmsKeyId),
		}
	}

	// Stopping condition
	if output.StoppingCondition != nil {
		details.StoppingCondition = connectors.StoppingCondition{
			MaxRuntimeSeconds:  int(aws.ToInt32(output.StoppingCondition.MaxRuntimeInSeconds)),
			MaxWaitTimeSeconds: int(aws.ToInt32(output.StoppingCondition.MaxWaitTimeInSeconds)),
		}
	}

	return details, nil
}

func (c *Connector) ListEndpoints(ctx context.Context) ([]connectors.SageMakerEndpoint, error) {
	var endpoints []connectors.SageMakerEndpoint

	paginator := sagemaker.NewListEndpointsPaginator(c.sagemakerClient, &sagemaker.ListEndpointsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing endpoints: %w", err)
		}

		for _, ep := range page.Endpoints {
			endpoints = append(endpoints, connectors.SageMakerEndpoint{
				EndpointARN:      aws.ToString(ep.EndpointArn),
				EndpointName:     aws.ToString(ep.EndpointName),
				EndpointStatus:   string(ep.EndpointStatus),
				CreationTime:     ep.CreationTime.String(),
				LastModifiedTime: ep.LastModifiedTime.String(),
			})
		}
	}

	return endpoints, nil
}

// =====================================================
// AI Connector Implementation (Bedrock)
// =====================================================

func (c *Connector) ListBedrockModels(ctx context.Context) ([]connectors.BedrockModel, error) {
	var models []connectors.BedrockModel

	output, err := c.bedrockClient.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{})
	if err != nil {
		return nil, fmt.Errorf("listing Bedrock models: %w", err)
	}

	for _, m := range output.ModelSummaries {
		model := connectors.BedrockModel{
			ModelID:                 aws.ToString(m.ModelId),
			ModelARN:               aws.ToString(m.ModelArn),
			ModelName:              aws.ToString(m.ModelName),
			ProviderName:           aws.ToString(m.ProviderName),
			InputModalities:        toStringSlice(m.InputModalities),
			OutputModalities:       toStringSlice(m.OutputModalities),
			CustomizationsSupported: toModelCustomizationSlice(m.CustomizationsSupported),
			InferenceTypesSupported: toInferenceTypeSlice(m.InferenceTypesSupported),
		}
		models = append(models, model)
	}

	return models, nil
}

func (c *Connector) ListModelInvocations(ctx context.Context, modelID string, since string) ([]connectors.ModelInvocation, error) {
	// Note: Bedrock doesn't have a direct API to list invocations.
	// This would typically come from CloudWatch Logs or CloudTrail.
	// For now, return an empty slice with a note that this requires additional setup.
	return []connectors.ModelInvocation{}, nil
}

// Helper functions for Bedrock type conversions
func toStringSlice(modalities []bedrocktypes.ModelModality) []string {
	result := make([]string, len(modalities))
	for i, m := range modalities {
		result[i] = string(m)
	}
	return result
}

func toModelCustomizationSlice(customizations []bedrocktypes.ModelCustomization) []string {
	result := make([]string, len(customizations))
	for i, c := range customizations {
		result[i] = string(c)
	}
	return result
}

func toInferenceTypeSlice(inferenceTypes []bedrocktypes.InferenceType) []string {
	result := make([]string, len(inferenceTypes))
	for i, t := range inferenceTypes {
		result[i] = string(t)
	}
	return result
}

// =====================================================
// CloudTrail Data Access Events
// =====================================================

// LookupEvents searches CloudTrail for events matching the specified criteria
func (c *Connector) LookupEvents(ctx context.Context, startTime, endTime time.Time, eventName string) ([]connectors.CloudTrailEvent, error) {
	var events []connectors.CloudTrailEvent

	input := &cloudtrail.LookupEventsInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
	}

	if eventName != "" {
		input.LookupAttributes = []cloudtrailTypes.LookupAttribute{
			{
				AttributeKey:   cloudtrailTypes.LookupAttributeKeyEventName,
				AttributeValue: aws.String(eventName),
			},
		}
	}

	paginator := cloudtrail.NewLookupEventsPaginator(c.cloudtrailClient, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("looking up CloudTrail events: %w", err)
		}

		for _, event := range page.Events {
			ctEvent := connectors.CloudTrailEvent{
				EventID:        aws.ToString(event.EventId),
				EventName:      aws.ToString(event.EventName),
				EventSource:    aws.ToString(event.EventSource),
				EventTime:      aws.ToTime(event.EventTime),
				Username:       aws.ToString(event.Username),
				AccessKeyID:    aws.ToString(event.AccessKeyId),
				ReadOnly:       aws.ToString(event.ReadOnly),
				CloudTrailJSON: aws.ToString(event.CloudTrailEvent),
			}

			for _, resource := range event.Resources {
				ctEvent.Resources = append(ctEvent.Resources, connectors.CloudTrailResource{
					ResourceType: aws.ToString(resource.ResourceType),
					ResourceName: aws.ToString(resource.ResourceName),
				})
			}

			events = append(events, ctEvent)
		}
	}

	return events, nil
}

// GetEventsByResource retrieves CloudTrail events for a specific resource
func (c *Connector) GetEventsByResource(ctx context.Context, resourceARN string, since time.Time) ([]connectors.CloudTrailEvent, error) {
	var events []connectors.CloudTrailEvent

	input := &cloudtrail.LookupEventsInput{
		StartTime: aws.Time(since),
		EndTime:   aws.Time(time.Now()),
		LookupAttributes: []cloudtrailTypes.LookupAttribute{
			{
				AttributeKey:   cloudtrailTypes.LookupAttributeKeyResourceName,
				AttributeValue: aws.String(resourceARN),
			},
		},
	}

	paginator := cloudtrail.NewLookupEventsPaginator(c.cloudtrailClient, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("looking up events for resource %s: %w", resourceARN, err)
		}

		for _, event := range page.Events {
			ctEvent := connectors.CloudTrailEvent{
				EventID:        aws.ToString(event.EventId),
				EventName:      aws.ToString(event.EventName),
				EventSource:    aws.ToString(event.EventSource),
				EventTime:      aws.ToTime(event.EventTime),
				Username:       aws.ToString(event.Username),
				AccessKeyID:    aws.ToString(event.AccessKeyId),
				ReadOnly:       aws.ToString(event.ReadOnly),
				CloudTrailJSON: aws.ToString(event.CloudTrailEvent),
			}

			for _, resource := range event.Resources {
				ctEvent.Resources = append(ctEvent.Resources, connectors.CloudTrailResource{
					ResourceType: aws.ToString(resource.ResourceType),
					ResourceName: aws.ToString(resource.ResourceName),
				})
			}

			events = append(events, ctEvent)
		}
	}

	return events, nil
}

// GetS3DataAccessEvents retrieves S3 data access events (GetObject, PutObject, etc.)
func (c *Connector) GetS3DataAccessEvents(ctx context.Context, bucketName string, since time.Time) ([]connectors.CloudTrailEvent, error) {
	var allEvents []connectors.CloudTrailEvent

	dataAccessEvents := []string{"GetObject", "PutObject", "DeleteObject", "CopyObject", "HeadObject"}

	for _, eventName := range dataAccessEvents {
		events, err := c.LookupEvents(ctx, since, time.Now(), eventName)
		if err != nil {
			continue // Log but don't fail on individual event type failures
		}

		// Filter events for the specific bucket
		for _, event := range events {
			for _, resource := range event.Resources {
				if resource.ResourceType == "AWS::S3::Bucket" && resource.ResourceName == bucketName {
					allEvents = append(allEvents, event)
					break
				}
			}
		}
	}

	return allEvents, nil
}

// =====================================================
// Enhanced KMS Methods
// =====================================================

// ListKeyGrants returns all grants for a specific KMS key
func (c *Connector) ListKeyGrants(ctx context.Context, keyID string) ([]connectors.KeyGrant, error) {
	var grants []connectors.KeyGrant

	paginator := kms.NewListGrantsPaginator(c.kmsClient, &kms.ListGrantsInput{
		KeyId: aws.String(keyID),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing key grants: %w", err)
		}

		for _, grant := range page.Grants {
			g := connectors.KeyGrant{
				GrantID:          aws.ToString(grant.GrantId),
				KeyID:            aws.ToString(grant.KeyId),
				GranteePrincipal: aws.ToString(grant.GranteePrincipal),
				RetiringPrincipal: aws.ToString(grant.RetiringPrincipal),
				IssuingAccount:   aws.ToString(grant.IssuingAccount),
				Name:             aws.ToString(grant.Name),
			}

			for _, op := range grant.Operations {
				g.Operations = append(g.Operations, string(op))
			}

			if grant.Constraints != nil {
				if grant.Constraints.EncryptionContextEquals != nil {
					g.Constraints = make(map[string]string)
					for k, v := range grant.Constraints.EncryptionContextEquals {
						g.Constraints[k] = v
					}
				}
			}

			if grant.CreationDate != nil {
				g.CreationDate = grant.CreationDate.String()
			}

			grants = append(grants, g)
		}
	}

	return grants, nil
}

// GetKeyRotationSchedule returns the rotation schedule for a KMS key
func (c *Connector) GetKeyRotationSchedule(ctx context.Context, keyID string) (*connectors.KeyRotationInfo, error) {
	// Check if rotation is enabled
	rotationOutput, err := c.kmsClient.GetKeyRotationStatus(ctx, &kms.GetKeyRotationStatusInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting key rotation status: %w", err)
	}

	info := &connectors.KeyRotationInfo{
		KeyID:           keyID,
		RotationEnabled: rotationOutput.KeyRotationEnabled,
	}

	// Get key metadata for additional info
	keyOutput, err := c.kmsClient.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	})
	if err == nil && keyOutput.KeyMetadata != nil {
		info.KeyManager = string(keyOutput.KeyMetadata.KeyManager)
		info.KeyState = string(keyOutput.KeyMetadata.KeyState)
		if keyOutput.KeyMetadata.CreationDate != nil {
			info.CreationDate = keyOutput.KeyMetadata.CreationDate.String()
		}
	}

	// Note: AWS KMS automatic rotation period is fixed at 365 days for customer-managed keys
	if info.RotationEnabled {
		info.RotationPeriodDays = 365
	}

	return info, nil
}

// ListKeyAliases returns all aliases for KMS keys
func (c *Connector) ListKeyAliases(ctx context.Context) ([]connectors.KeyAlias, error) {
	var aliases []connectors.KeyAlias

	paginator := kms.NewListAliasesPaginator(c.kmsClient, &kms.ListAliasesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing key aliases: %w", err)
		}

		for _, alias := range page.Aliases {
			aliases = append(aliases, connectors.KeyAlias{
				AliasName:   aws.ToString(alias.AliasName),
				AliasARN:    aws.ToString(alias.AliasArn),
				TargetKeyID: aws.ToString(alias.TargetKeyId),
				CreationDate: func() string {
					if alias.CreationDate != nil {
						return alias.CreationDate.String()
					}
					return ""
				}(),
				LastUpdatedDate: func() string {
					if alias.LastUpdatedDate != nil {
						return alias.LastUpdatedDate.String()
					}
					return ""
				}(),
			})
		}
	}

	return aliases, nil
}

// =====================================================
// Enhanced SageMaker Methods
// =====================================================

// ListProcessingJobs returns SageMaker processing jobs
func (c *Connector) ListProcessingJobs(ctx context.Context) ([]connectors.ProcessingJob, error) {
	var jobs []connectors.ProcessingJob

	paginator := sagemaker.NewListProcessingJobsPaginator(c.sagemakerClient, &sagemaker.ListProcessingJobsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing processing jobs: %w", err)
		}

		for _, job := range page.ProcessingJobSummaries {
			j := connectors.ProcessingJob{
				ProcessingJobARN:    aws.ToString(job.ProcessingJobArn),
				ProcessingJobName:   aws.ToString(job.ProcessingJobName),
				ProcessingJobStatus: string(job.ProcessingJobStatus),
				CreationTime:        job.CreationTime.String(),
			}
			if job.ProcessingEndTime != nil {
				j.ProcessingEndTime = job.ProcessingEndTime.String()
			}
			if job.LastModifiedTime != nil {
				j.LastModifiedTime = job.LastModifiedTime.String()
			}
			jobs = append(jobs, j)
		}
	}

	return jobs, nil
}

// GetProcessingJobDetails returns detailed information about a processing job
func (c *Connector) GetProcessingJobDetails(ctx context.Context, jobName string) (*connectors.ProcessingJobDetails, error) {
	output, err := c.sagemakerClient.DescribeProcessingJob(ctx, &sagemaker.DescribeProcessingJobInput{
		ProcessingJobName: aws.String(jobName),
	})
	if err != nil {
		return nil, fmt.Errorf("describing processing job: %w", err)
	}

	details := &connectors.ProcessingJobDetails{
		ProcessingJob: connectors.ProcessingJob{
			ProcessingJobARN:    aws.ToString(output.ProcessingJobArn),
			ProcessingJobName:   aws.ToString(output.ProcessingJobName),
			ProcessingJobStatus: string(output.ProcessingJobStatus),
			CreationTime:        output.CreationTime.String(),
		},
		RoleARN:          aws.ToString(output.RoleArn),
		ExitMessage:      aws.ToString(output.ExitMessage),
		FailureReason:    aws.ToString(output.FailureReason),
	}

	if output.ProcessingEndTime != nil {
		details.ProcessingEndTime = output.ProcessingEndTime.String()
	}
	if output.LastModifiedTime != nil {
		details.LastModifiedTime = output.LastModifiedTime.String()
	}
	if output.ProcessingStartTime != nil {
		details.ProcessingStartTime = output.ProcessingStartTime.String()
	}

	// Processing inputs
	for _, input := range output.ProcessingInputs {
		pi := connectors.ProcessingInput{
			InputName: aws.ToString(input.InputName),
		}
		if input.S3Input != nil {
			pi.S3URI = aws.ToString(input.S3Input.S3Uri)
			pi.S3DataType = string(input.S3Input.S3DataType)
			pi.S3InputMode = string(input.S3Input.S3InputMode)
			pi.LocalPath = aws.ToString(input.S3Input.LocalPath)
		}
		details.ProcessingInputs = append(details.ProcessingInputs, pi)
	}

	// Processing output
	if output.ProcessingOutputConfig != nil {
		details.OutputConfig = connectors.ProcessingOutputConfig{
			KMSKeyID: aws.ToString(output.ProcessingOutputConfig.KmsKeyId),
		}
		for _, out := range output.ProcessingOutputConfig.Outputs {
			po := connectors.ProcessingOutput{
				OutputName: aws.ToString(out.OutputName),
			}
			if out.S3Output != nil {
				po.S3URI = aws.ToString(out.S3Output.S3Uri)
				po.S3UploadMode = string(out.S3Output.S3UploadMode)
				po.LocalPath = aws.ToString(out.S3Output.LocalPath)
			}
			details.OutputConfig.Outputs = append(details.OutputConfig.Outputs, po)
		}
	}

	// Resource configuration
	if output.ProcessingResources != nil && output.ProcessingResources.ClusterConfig != nil {
		cc := output.ProcessingResources.ClusterConfig
		details.ResourceConfig = connectors.ProcessingResourceConfig{
			InstanceType:   string(cc.InstanceType),
			InstanceCount:  int(aws.ToInt32(cc.InstanceCount)),
			VolumeSizeGB:   int(aws.ToInt32(cc.VolumeSizeInGB)),
			VolumeKmsKeyID: aws.ToString(cc.VolumeKmsKeyId),
		}
	}

	return details, nil
}

// ListNotebookInstances returns SageMaker notebook instances
func (c *Connector) ListNotebookInstances(ctx context.Context) ([]connectors.NotebookInstance, error) {
	var notebooks []connectors.NotebookInstance

	paginator := sagemaker.NewListNotebookInstancesPaginator(c.sagemakerClient, &sagemaker.ListNotebookInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing notebook instances: %w", err)
		}

		for _, nb := range page.NotebookInstances {
			notebook := connectors.NotebookInstance{
				NotebookInstanceARN:    aws.ToString(nb.NotebookInstanceArn),
				NotebookInstanceName:   aws.ToString(nb.NotebookInstanceName),
				NotebookInstanceStatus: string(nb.NotebookInstanceStatus),
				InstanceType:           string(nb.InstanceType),
				CreationTime:           nb.CreationTime.String(),
				URL:                    aws.ToString(nb.Url),
			}
			if nb.LastModifiedTime != nil {
				notebook.LastModifiedTime = nb.LastModifiedTime.String()
			}
			notebooks = append(notebooks, notebook)
		}
	}

	return notebooks, nil
}

// GetNotebookInstanceDetails returns detailed information about a notebook instance
func (c *Connector) GetNotebookInstanceDetails(ctx context.Context, instanceName string) (*connectors.NotebookInstanceDetails, error) {
	output, err := c.sagemakerClient.DescribeNotebookInstance(ctx, &sagemaker.DescribeNotebookInstanceInput{
		NotebookInstanceName: aws.String(instanceName),
	})
	if err != nil {
		return nil, fmt.Errorf("describing notebook instance: %w", err)
	}

	details := &connectors.NotebookInstanceDetails{
		NotebookInstance: connectors.NotebookInstance{
			NotebookInstanceARN:    aws.ToString(output.NotebookInstanceArn),
			NotebookInstanceName:   aws.ToString(output.NotebookInstanceName),
			NotebookInstanceStatus: string(output.NotebookInstanceStatus),
			InstanceType:           string(output.InstanceType),
			CreationTime:           output.CreationTime.String(),
			URL:                    aws.ToString(output.Url),
		},
		RoleARN:                  aws.ToString(output.RoleArn),
		KMSKeyID:                 aws.ToString(output.KmsKeyId),
		NetworkInterfaceID:       aws.ToString(output.NetworkInterfaceId),
		SubnetID:                 aws.ToString(output.SubnetId),
		VolumeSizeGB:             int(aws.ToInt32(output.VolumeSizeInGB)),
		DirectInternetAccess:     string(output.DirectInternetAccess),
		RootAccess:               string(output.RootAccess),
		SecurityGroups:           output.SecurityGroups,
		AcceleratorTypes:         toAcceleratorTypeStrings(output.AcceleratorTypes),
		DefaultCodeRepository:    aws.ToString(output.DefaultCodeRepository),
		AdditionalCodeRepositories: output.AdditionalCodeRepositories,
		PlatformIdentifier:       aws.ToString(output.PlatformIdentifier),
	}

	if output.LastModifiedTime != nil {
		details.LastModifiedTime = output.LastModifiedTime.String()
	}

	return details, nil
}

// Helper function for accelerator types
func toAcceleratorTypeStrings(types []sagemakerTypes.NotebookInstanceAcceleratorType) []string {
	result := make([]string, len(types))
	for i, t := range types {
		result[i] = string(t)
	}
	return result
}
