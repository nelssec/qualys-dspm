package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/cloudfunctions/v1"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/qualys/dspm/internal/connectors"
	"github.com/qualys/dspm/internal/models"
)

type Connector struct {
	projectID       string
	credentialsFile string

	storageClient   *storage.Client
	crmClient       *cloudresourcemanager.Service
	functionsClient *cloudfunctions.Service
}

type Config struct {
	ProjectID       string
	CredentialsFile string
}

func New(ctx context.Context, cfg Config) (*Connector, error) {
	var opts []option.ClientOption
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	storageClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}

	crmClient, err := cloudresourcemanager.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating resource manager client: %w", err)
	}

	functionsClient, err := cloudfunctions.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating functions client: %w", err)
	}

	return &Connector{
		projectID:       cfg.ProjectID,
		credentialsFile: cfg.CredentialsFile,
		storageClient:   storageClient,
		crmClient:       crmClient,
		functionsClient: functionsClient,
	}, nil
}

func (c *Connector) Provider() models.Provider {
	return models.ProviderGCP
}

func (c *Connector) ProjectID() string {
	return c.projectID
}

func (c *Connector) Validate(ctx context.Context) error {
	it := c.storageClient.Buckets(ctx, c.projectID)
	_, err := it.Next()
	if err != nil && err != iterator.Done {
		return fmt.Errorf("validating storage access: %w", err)
	}

	_, err = c.crmClient.Projects.Get(c.projectID).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("validating resource manager access: %w", err)
	}

	return nil
}

func (c *Connector) Close() error {
	if c.storageClient != nil {
		return c.storageClient.Close()
	}
	return nil
}

func (c *Connector) ListBuckets(ctx context.Context) ([]connectors.BucketInfo, error) {
	var buckets []connectors.BucketInfo

	it := c.storageClient.Buckets(ctx, c.projectID)
	for {
		bucket, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing buckets: %w", err)
		}

		buckets = append(buckets, connectors.BucketInfo{
			Name:      bucket.Name,
			Region:    bucket.Location,
			CreatedAt: bucket.Created.String(),
			ARN:       fmt.Sprintf("gs://%s", bucket.Name),
		})
	}

	return buckets, nil
}

func (c *Connector) GetBucketMetadata(ctx context.Context, bucketName string) (*connectors.BucketMetadata, error) {
	bucket := c.storageClient.Bucket(bucketName)
	attrs, err := bucket.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting bucket attributes: %w", err)
	}

	metadata := &connectors.BucketMetadata{
		Name:      bucketName,
		Region:    attrs.Location,
		ARN:       fmt.Sprintf("gs://%s", bucketName),
		CreatedAt: attrs.Created.String(),
	}

	if attrs.Encryption != nil && attrs.Encryption.DefaultKMSKeyName != "" {
		metadata.Encryption.Enabled = true
		metadata.Encryption.Type = models.EncryptionCMK
		metadata.Encryption.KeyARN = attrs.Encryption.DefaultKMSKeyName
	} else {
		metadata.Encryption.Enabled = true
		metadata.Encryption.Type = models.EncryptionSSE
	}

	metadata.Versioning = attrs.VersioningEnabled

	if attrs.Logging != nil && attrs.Logging.LogBucket != "" {
		metadata.Logging.Enabled = true
		metadata.Logging.TargetBucket = attrs.Logging.LogBucket
		metadata.Logging.TargetPrefix = attrs.Logging.LogObjectPrefix
	}

	policy, err := bucket.IAM().Policy(ctx)
	if err == nil {
		for _, members := range policy.InternalProto.GetBindings() {
			for _, member := range members.Members {
				if member == "allUsers" || member == "allAuthenticatedUsers" {
					metadata.PublicAccessBlock.BlockPublicAcls = false
					break
				}
			}
		}
	}

	if attrs.UniformBucketLevelAccess.Enabled {
		metadata.PublicAccessBlock.BlockPublicAcls = true
	}

	metadata.Tags = make(map[string]string)
	for k, v := range attrs.Labels {
		metadata.Tags[k] = v
	}

	return metadata, nil
}

func (c *Connector) ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]connectors.ObjectInfo, error) {
	var objects []connectors.ObjectInfo

	bucket := c.storageClient.Bucket(bucketName)
	query := &storage.Query{Prefix: prefix}

	it := bucket.Objects(ctx, query)
	count := 0
	for count < maxKeys {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("listing objects: %w", err)
		}

		objects = append(objects, connectors.ObjectInfo{
			Key:          obj.Name,
			Size:         obj.Size,
			LastModified: obj.Updated.String(),
			StorageClass: obj.StorageClass,
			ETag:         obj.Etag,
		})
		count++
	}

	return objects, nil
}

func (c *Connector) GetObject(ctx context.Context, bucketName, objectKey string, byteRange *connectors.ByteRange) (io.ReadCloser, error) {
	bucket := c.storageClient.Bucket(bucketName)
	obj := bucket.Object(objectKey)

	var reader *storage.Reader
	var err error

	if byteRange != nil {
		reader, err = obj.NewRangeReader(ctx, byteRange.Start, byteRange.End-byteRange.Start+1)
	} else {
		reader, err = obj.NewReader(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("getting object: %w", err)
	}

	return reader, nil
}

func (c *Connector) GetBucketPolicy(ctx context.Context, bucketName string) (*connectors.BucketPolicy, error) {
	bucket := c.storageClient.Bucket(bucketName)
	policy, err := bucket.IAM().Policy(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting bucket policy: %w", err)
	}

	result := &connectors.BucketPolicy{}

	for _, binding := range policy.InternalProto.GetBindings() {
		for _, member := range binding.Members {
			if member == "allUsers" || member == "allAuthenticatedUsers" {
				result.IsPublic = true
				result.PublicActions = append(result.PublicActions, binding.Role)
			}
		}
	}

	policyBytes, _ := json.Marshal(policy)
	result.Policy = string(policyBytes)

	return result, nil
}

func (c *Connector) GetBucketACL(ctx context.Context, bucketName string) (*connectors.BucketACL, error) {
	bucket := c.storageClient.Bucket(bucketName)
	acl, err := bucket.ACL().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting bucket ACL: %w", err)
	}

	result := &connectors.BucketACL{}

	for _, rule := range acl {
		grant := connectors.ACLGrant{
			Grantee:    string(rule.Entity),
			Permission: string(rule.Role),
		}

		if rule.Entity == storage.AllUsers || rule.Entity == storage.AllAuthenticatedUsers {
			grant.IsPublic = true
		}

		result.Grants = append(result.Grants, grant)
	}

	return result, nil
}

func (c *Connector) ListUsers(ctx context.Context) ([]connectors.Principal, error) {
	var principals []connectors.Principal

	policy, err := c.crmClient.Projects.GetIamPolicy(c.projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting IAM policy: %w", err)
	}

	seen := make(map[string]bool)
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if seen[member] {
				continue
			}
			seen[member] = true

			principal := connectors.Principal{
				ARN:  member,
				Name: member,
			}

			switch {
			case len(member) > 5 && member[:5] == "user:":
				principal.Type = "USER"
				principal.Name = member[5:]
			case len(member) > 6 && member[:6] == "group:":
				principal.Type = "GROUP"
				principal.Name = member[6:]
			case len(member) > 15 && member[:15] == "serviceAccount:":
				principal.Type = "SERVICE"
				principal.Name = member[15:]
			}

			if principal.Type == "USER" {
				principals = append(principals, principal)
			}
		}
	}

	return principals, nil
}

func (c *Connector) ListRoles(ctx context.Context) ([]connectors.Principal, error) {
	var roles []connectors.Principal

	policy, err := c.crmClient.Projects.GetIamPolicy(c.projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting IAM policy: %w", err)
	}

	seen := make(map[string]bool)
	for _, binding := range policy.Bindings {
		if seen[binding.Role] {
			continue
		}
		seen[binding.Role] = true

		roles = append(roles, connectors.Principal{
			ARN:  binding.Role,
			Name: binding.Role,
			Type: "ROLE",
		})
	}

	return roles, nil
}

func (c *Connector) ListPolicies(ctx context.Context) ([]connectors.PolicyInfo, error) {
	var policies []connectors.PolicyInfo

	policy, err := c.crmClient.Projects.GetIamPolicy(c.projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting IAM policy: %w", err)
	}

	for _, binding := range policy.Bindings {
		policies = append(policies, connectors.PolicyInfo{
			ARN:         binding.Role,
			Name:        binding.Role,
			Type:        "IAM_BINDING",
			AttachCount: len(binding.Members),
			IsAttached:  len(binding.Members) > 0,
		})
	}

	return policies, nil
}

func (c *Connector) GetPolicy(ctx context.Context, policyARN string) (*connectors.PolicyDocument, error) {
	policy, err := c.crmClient.Projects.GetIamPolicy(c.projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting IAM policy: %w", err)
	}

	doc := &connectors.PolicyDocument{
		Version: fmt.Sprintf("%d", policy.Version),
	}

	for _, binding := range policy.Bindings {
		if binding.Role == policyARN {
			stmt := connectors.PolicyStatement{
				Effect:     "Allow",
				Actions:    []string{binding.Role},
				Principals: binding.Members,
			}
			if binding.Condition != nil {
				stmt.Conditions = map[string]interface{}{
					"expression":  binding.Condition.Expression,
					"title":       binding.Condition.Title,
					"description": binding.Condition.Description,
				}
			}
			doc.Statements = append(doc.Statements, stmt)
		}
	}

	policyBytes, _ := json.Marshal(policy)
	doc.Raw = string(policyBytes)

	return doc, nil
}

func (c *Connector) ListAttachedPolicies(ctx context.Context, principalARN string) ([]connectors.PolicyInfo, error) {
	var policies []connectors.PolicyInfo

	policy, err := c.crmClient.Projects.GetIamPolicy(c.projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting IAM policy: %w", err)
	}

	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member == principalARN {
				policies = append(policies, connectors.PolicyInfo{
					ARN:        binding.Role,
					Name:       binding.Role,
					Type:       "IAM_BINDING",
					IsAttached: true,
				})
			}
		}
	}

	return policies, nil
}

func (c *Connector) GetServiceAccounts(ctx context.Context) ([]connectors.Principal, error) {
	var principals []connectors.Principal

	policy, err := c.crmClient.Projects.GetIamPolicy(c.projectID, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting IAM policy: %w", err)
	}

	seen := make(map[string]bool)
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if seen[member] {
				continue
			}

			if len(member) > 15 && member[:15] == "serviceAccount:" {
				seen[member] = true
				principals = append(principals, connectors.Principal{
					ARN:  member,
					Name: member[15:],
					Type: "SERVICE",
				})
			}
		}
	}

	return principals, nil
}

func (c *Connector) ListFunctions(ctx context.Context) ([]connectors.FunctionInfo, error) {
	var functions []connectors.FunctionInfo

	parent := fmt.Sprintf("projects/%s/locations/-", c.projectID)
	resp, err := c.functionsClient.Projects.Locations.Functions.List(parent).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("listing functions: %w", err)
	}

	for _, fn := range resp.Functions {
		info := connectors.FunctionInfo{
			ARN:          fn.Name,
			Name:         extractFunctionName(fn.Name),
			Runtime:      fn.Runtime,
			LastModified: fn.UpdateTime,
		}

		if fn.AvailableMemoryMb != 0 {
			info.MemorySize = int(fn.AvailableMemoryMb)
		}
		if fn.Timeout != "" {
			info.Timeout = 60 // Default
		}

		parts := splitFunctionName(fn.Name)
		if len(parts) >= 4 {
			info.Region = parts[3]
		}

		if fn.Labels != nil {
			info.Tags = fn.Labels
		}

		functions = append(functions, info)
	}

	return functions, nil
}

func (c *Connector) GetFunctionConfig(ctx context.Context, functionName string) (*connectors.FunctionConfig, error) {
	fn, err := c.functionsClient.Projects.Locations.Functions.Get(functionName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting function: %w", err)
	}

	config := &connectors.FunctionConfig{
		FunctionInfo: connectors.FunctionInfo{
			ARN:          fn.Name,
			Name:         extractFunctionName(fn.Name),
			Runtime:      fn.Runtime,
			Handler:      fn.EntryPoint,
			LastModified: fn.UpdateTime,
		},
		Role:      fn.ServiceAccountEmail,
		KMSKeyARN: fn.KmsKeyName,
	}

	if fn.AvailableMemoryMb != 0 {
		config.MemorySize = int(fn.AvailableMemoryMb)
	}

	if fn.EnvironmentVariables != nil {
		config.Environment = fn.EnvironmentVariables
	}

	if fn.VpcConnector != "" {
		config.VPCConfig = &connectors.VPCConfig{
			VPCID: fn.VpcConnector,
		}
	}

	if fn.Labels != nil {
		config.Tags = fn.Labels
	}

	return config, nil
}

func (c *Connector) GetFunctionPolicy(ctx context.Context, functionName string) (*connectors.PolicyDocument, error) {
	policy, err := c.functionsClient.Projects.Locations.Functions.GetIamPolicy(functionName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("getting function policy: %w", err)
	}

	doc := &connectors.PolicyDocument{
		Version: fmt.Sprintf("%d", policy.Version),
	}

	for _, binding := range policy.Bindings {
		stmt := connectors.PolicyStatement{
			Effect:     "Allow",
			Actions:    []string{binding.Role},
			Principals: binding.Members,
		}
		doc.Statements = append(doc.Statements, stmt)
	}

	policyBytes, _ := json.Marshal(policy)
	doc.Raw = string(policyBytes)

	return doc, nil
}

func extractFunctionName(fullName string) string {
	parts := splitFunctionName(fullName)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

func splitFunctionName(fullName string) []string {
	var parts []string
	current := ""
	for _, c := range fullName {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

var (
	_ connectors.StorageConnector    = (*Connector)(nil)
	_ connectors.IAMConnector        = (*Connector)(nil)
	_ connectors.ServerlessConnector = (*Connector)(nil)
)
