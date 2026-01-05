package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/qualys/dspm/internal/connectors"
	"github.com/qualys/dspm/internal/models"
)

type Connector struct {
	cfg       aws.Config
	accountID string
	region    string

	s3Client     *s3.Client
	iamClient    *iam.Client
	lambdaClient *lambda.Client
	kmsClient    *kms.Client
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
		cfg:          awsCfg,
		accountID:    aws.ToString(identity.Account),
		region:       cfg.Region,
		s3Client:     s3.NewFromConfig(awsCfg),
		iamClient:    iam.NewFromConfig(awsCfg),
		lambdaClient: lambda.NewFromConfig(awsCfg),
		kmsClient:    kms.NewFromConfig(awsCfg),
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

	json.Unmarshal([]byte(doc.Raw), doc)
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

	json.Unmarshal([]byte(doc.Raw), doc)
	return doc, nil
}
