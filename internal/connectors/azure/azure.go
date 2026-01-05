package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/qualys/dspm/internal/connectors"
	"github.com/qualys/dspm/internal/models"
)

// Connector implements the Azure cloud connector
type Connector struct {
	credential     *azidentity.ClientSecretCredential
	subscriptionID string
	tenantID       string

	// Service clients
	storageClient *armstorage.AccountsClient
	blobClients   map[string]*azblob.Client
	authClient    *armauthorization.RoleAssignmentsClient
}

// Config holds Azure connector configuration
type Config struct {
	TenantID       string
	ClientID       string
	ClientSecret   string
	SubscriptionID string
}

// New creates a new Azure connector
func New(ctx context.Context, cfg Config) (*Connector, error) {
	credential, err := azidentity.NewClientSecretCredential(cfg.TenantID, cfg.ClientID, cfg.ClientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("creating credential: %w", err)
	}

	storageClient, err := armstorage.NewAccountsClient(cfg.SubscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}

	authClient, err := armauthorization.NewRoleAssignmentsClient(cfg.SubscriptionID, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("creating auth client: %w", err)
	}

	return &Connector{
		credential:     credential,
		subscriptionID: cfg.SubscriptionID,
		tenantID:       cfg.TenantID,
		storageClient:  storageClient,
		blobClients:    make(map[string]*azblob.Client),
		authClient:     authClient,
	}, nil
}

// Provider returns the cloud provider type
func (c *Connector) Provider() models.Provider {
	return models.ProviderAzure
}

// SubscriptionID returns the Azure subscription ID
func (c *Connector) SubscriptionID() string {
	return c.subscriptionID
}

// Validate tests the connection and permissions
func (c *Connector) Validate(ctx context.Context) error {
	pager := c.storageClient.NewListPager(nil)
	_, err := pager.NextPage(ctx)
	if err != nil {
		return fmt.Errorf("validating storage access: %w", err)
	}
	return nil
}

// Close releases any resources
func (c *Connector) Close() error {
	return nil
}

// getBlobClient returns a blob client for a storage account
func (c *Connector) getBlobClient(ctx context.Context, accountName string) (*azblob.Client, error) {
	if client, ok := c.blobClients[accountName]; ok {
		return client, nil
	}

	url := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	client, err := azblob.NewClient(url, c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("creating blob client: %w", err)
	}

	c.blobClients[accountName] = client
	return client, nil
}

// --- Storage Operations ---

// ListBuckets returns all storage accounts and their containers
func (c *Connector) ListBuckets(ctx context.Context) ([]connectors.BucketInfo, error) {
	var buckets []connectors.BucketInfo

	// List storage accounts
	pager := c.storageClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing storage accounts: %w", err)
		}

		for _, account := range page.Value {
			accountName := *account.Name
			location := *account.Location

			// Get blob client for this account
			blobClient, err := c.getBlobClient(ctx, accountName)
			if err != nil {
				continue // Skip accounts we can't access
			}

			// List containers in this account
			containerPager := blobClient.NewListContainersPager(nil)
			for containerPager.More() {
				containerPage, err := containerPager.NextPage(ctx)
				if err != nil {
					break // Skip if we can't list containers
				}

				for _, container := range containerPage.ContainerItems {
					buckets = append(buckets, connectors.BucketInfo{
						Name:   fmt.Sprintf("%s/%s", accountName, *container.Name),
						Region: location,
						ARN:    fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s",
							c.subscriptionID, extractResourceGroup(*account.ID), accountName, *container.Name),
					})
				}
			}
		}
	}

	return buckets, nil
}

// GetBucketMetadata returns detailed metadata for a container
func (c *Connector) GetBucketMetadata(ctx context.Context, bucketName string) (*connectors.BucketMetadata, error) {
	parts := strings.SplitN(bucketName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid bucket name format, expected 'accountName/containerName'")
	}
	accountName, containerName := parts[0], parts[1]

	// Get storage account details
	pager := c.storageClient.NewListPager(nil)
	var account *armstorage.Account
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing storage accounts: %w", err)
		}
		for _, acc := range page.Value {
			if *acc.Name == accountName {
				account = acc
				break
			}
		}
		if account != nil {
			break
		}
	}

	if account == nil {
		return nil, fmt.Errorf("storage account not found: %s", accountName)
	}

	metadata := &connectors.BucketMetadata{
		Name:   bucketName,
		Region: *account.Location,
		ARN:    fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s/blobServices/default/containers/%s",
			c.subscriptionID, extractResourceGroup(*account.ID), accountName, containerName),
	}

	// Check encryption
	if account.Properties != nil && account.Properties.Encryption != nil {
		metadata.Encryption.Enabled = true
		if account.Properties.Encryption.KeySource != nil {
			switch *account.Properties.Encryption.KeySource {
			case armstorage.KeySourceMicrosoftStorage:
				metadata.Encryption.Type = models.EncryptionSSE
			case armstorage.KeySourceMicrosoftKeyvault:
				metadata.Encryption.Type = models.EncryptionCMK
				if account.Properties.Encryption.KeyVaultProperties != nil {
					metadata.Encryption.KeyARN = ptrToString(account.Properties.Encryption.KeyVaultProperties.KeyVaultURI)
				}
			}
		}
	}

	// Check public access
	if account.Properties != nil {
		if account.Properties.AllowBlobPublicAccess != nil {
			metadata.PublicAccessBlock.BlockPublicAcls = !*account.Properties.AllowBlobPublicAccess
		}
	}

	// Get container properties
	blobClient, err := c.getBlobClient(ctx, accountName)
	if err == nil {
		containerClient := blobClient.ServiceClient().NewContainerClient(containerName)
		props, err := containerClient.GetProperties(ctx, nil)
		if err == nil {
			if props.BlobPublicAccess != nil && *props.BlobPublicAccess != "" {
				metadata.PublicAccessBlock.BlockPublicAcls = false
			}
		}
	}

	// Get tags
	if account.Tags != nil {
		metadata.Tags = make(map[string]string)
		for k, v := range account.Tags {
			if v != nil {
				metadata.Tags[k] = *v
			}
		}
	}

	return metadata, nil
}

// ListObjects lists blobs in a container
func (c *Connector) ListObjects(ctx context.Context, bucketName, prefix string, maxKeys int) ([]connectors.ObjectInfo, error) {
	parts := strings.SplitN(bucketName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid bucket name format")
	}
	accountName, containerName := parts[0], parts[1]

	blobClient, err := c.getBlobClient(ctx, accountName)
	if err != nil {
		return nil, err
	}

	containerClient := blobClient.ServiceClient().NewContainerClient(containerName)
	var objects []connectors.ObjectInfo

	pager := containerClient.NewListBlobsFlatPager(&azblob.ListBlobsFlatOptions{
		Prefix:     &prefix,
		MaxResults: int32Ptr(int32(maxKeys)),
	})

	for pager.More() && len(objects) < maxKeys {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			if len(objects) >= maxKeys {
				break
			}
			obj := connectors.ObjectInfo{
				Key:  *blob.Name,
				ETag: string(*blob.Properties.ETag),
			}
			if blob.Properties.ContentLength != nil {
				obj.Size = *blob.Properties.ContentLength
			}
			if blob.Properties.LastModified != nil {
				obj.LastModified = blob.Properties.LastModified.String()
			}
			if blob.Properties.AccessTier != nil {
				obj.StorageClass = string(*blob.Properties.AccessTier)
			}
			objects = append(objects, obj)
		}
	}

	return objects, nil
}

// GetObject retrieves a blob's content
func (c *Connector) GetObject(ctx context.Context, bucketName, objectKey string, byteRange *connectors.ByteRange) (io.ReadCloser, error) {
	parts := strings.SplitN(bucketName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid bucket name format")
	}
	accountName, containerName := parts[0], parts[1]

	blobClient, err := c.getBlobClient(ctx, accountName)
	if err != nil {
		return nil, err
	}

	containerClient := blobClient.ServiceClient().NewContainerClient(containerName)
	blobDownloadClient := containerClient.NewBlobClient(objectKey)

	var opts *azblob.DownloadStreamOptions
	if byteRange != nil {
		opts = &azblob.DownloadStreamOptions{
			Range: azblob.HTTPRange{
				Offset: byteRange.Start,
				Count:  byteRange.End - byteRange.Start + 1,
			},
		}
	}

	resp, err := blobDownloadClient.DownloadStream(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("downloading blob: %w", err)
	}

	return resp.Body, nil
}

// GetBucketPolicy returns the container access policy
func (c *Connector) GetBucketPolicy(ctx context.Context, bucketName string) (*connectors.BucketPolicy, error) {
	parts := strings.SplitN(bucketName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid bucket name format")
	}
	accountName, containerName := parts[0], parts[1]

	blobClient, err := c.getBlobClient(ctx, accountName)
	if err != nil {
		return nil, err
	}

	containerClient := blobClient.ServiceClient().NewContainerClient(containerName)
	props, err := containerClient.GetProperties(ctx, nil)
	if err != nil {
		return nil, err
	}

	policy := &connectors.BucketPolicy{}

	if props.BlobPublicAccess != nil {
		accessLevel := string(*props.BlobPublicAccess)
		if accessLevel == "container" || accessLevel == "blob" {
			policy.IsPublic = true
			policy.PublicActions = []string{"read"}
			if accessLevel == "container" {
				policy.PublicActions = append(policy.PublicActions, "list")
			}
		}
	}

	return policy, nil
}

// GetBucketACL returns the container ACL
func (c *Connector) GetBucketACL(ctx context.Context, bucketName string) (*connectors.BucketACL, error) {
	parts := strings.SplitN(bucketName, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid bucket name format")
	}
	accountName, containerName := parts[0], parts[1]

	blobClient, err := c.getBlobClient(ctx, accountName)
	if err != nil {
		return nil, err
	}

	containerClient := blobClient.ServiceClient().NewContainerClient(containerName)
	acl, err := containerClient.GetAccessPolicy(ctx, nil)
	if err != nil {
		return nil, err
	}

	result := &connectors.BucketACL{}

	for _, signedID := range acl.SignedIdentifiers {
		grant := connectors.ACLGrant{
			Grantee:    *signedID.ID,
			Permission: *signedID.AccessPolicy.Permission,
		}
		result.Grants = append(result.Grants, grant)
	}

	return result, nil
}

// --- IAM Operations ---

// ListUsers returns all users (via role assignments)
func (c *Connector) ListUsers(ctx context.Context) ([]connectors.Principal, error) {
	var principals []connectors.Principal

	pager := c.authClient.NewListForSubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing role assignments: %w", err)
		}

		for _, assignment := range page.Value {
			if assignment.Properties != nil && assignment.Properties.PrincipalType != nil {
				principalType := string(*assignment.Properties.PrincipalType)
				if principalType == "User" {
					principals = append(principals, connectors.Principal{
						ARN:  *assignment.Properties.PrincipalID,
						Type: "USER",
					})
				}
			}
		}
	}

	return principals, nil
}

// ListRoles returns all role definitions
func (c *Connector) ListRoles(ctx context.Context) ([]connectors.Principal, error) {
	var roles []connectors.Principal

	roleDefsClient, err := armauthorization.NewRoleDefinitionsClient(c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("creating role definitions client: %w", err)
	}

	scope := fmt.Sprintf("/subscriptions/%s", c.subscriptionID)
	pager := roleDefsClient.NewListPager(scope, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing role definitions: %w", err)
		}

		for _, role := range page.Value {
			if role.Properties != nil {
				roles = append(roles, connectors.Principal{
					ARN:         *role.ID,
					Name:        ptrToString(role.Properties.RoleName),
					Type:        "ROLE",
					Description: ptrToString(role.Properties.Description),
				})
			}
		}
	}

	return roles, nil
}

// ListPolicies returns role definitions as policies
func (c *Connector) ListPolicies(ctx context.Context) ([]connectors.PolicyInfo, error) {
	var policies []connectors.PolicyInfo

	roleDefsClient, err := armauthorization.NewRoleDefinitionsClient(c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("creating role definitions client: %w", err)
	}

	scope := fmt.Sprintf("/subscriptions/%s", c.subscriptionID)
	pager := roleDefsClient.NewListPager(scope, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing role definitions: %w", err)
		}

		for _, role := range page.Value {
			if role.Properties != nil {
				policies = append(policies, connectors.PolicyInfo{
					ARN:         *role.ID,
					Name:        ptrToString(role.Properties.RoleName),
					Type:        string(*role.Properties.Type),
					Description: ptrToString(role.Properties.Description),
				})
			}
		}
	}

	return policies, nil
}

// GetPolicy returns a role definition
func (c *Connector) GetPolicy(ctx context.Context, policyARN string) (*connectors.PolicyDocument, error) {
	roleDefsClient, err := armauthorization.NewRoleDefinitionsClient(c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("creating role definitions client: %w", err)
	}

	role, err := roleDefsClient.GetByID(ctx, policyARN, nil)
	if err != nil {
		return nil, fmt.Errorf("getting role definition: %w", err)
	}

	doc := &connectors.PolicyDocument{}

	if role.Properties != nil && role.Properties.Permissions != nil {
		for _, perm := range role.Properties.Permissions {
			stmt := connectors.PolicyStatement{
				Effect: "Allow",
			}
			for _, action := range perm.Actions {
				stmt.Actions = append(stmt.Actions, *action)
			}
			for _, notAction := range perm.NotActions {
				stmt.Actions = append(stmt.Actions, "NOT:"+*notAction)
			}
			doc.Statements = append(doc.Statements, stmt)
		}
	}

	rawBytes, _ := json.Marshal(role)
	doc.Raw = string(rawBytes)

	return doc, nil
}

// ListAttachedPolicies returns role assignments for a principal
func (c *Connector) ListAttachedPolicies(ctx context.Context, principalARN string) ([]connectors.PolicyInfo, error) {
	var policies []connectors.PolicyInfo

	filter := fmt.Sprintf("principalId eq '%s'", principalARN)
	pager := c.authClient.NewListForSubscriptionPager(&armauthorization.RoleAssignmentsClientListForSubscriptionOptions{
		Filter: &filter,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing role assignments: %w", err)
		}

		for _, assignment := range page.Value {
			if assignment.Properties != nil {
				policies = append(policies, connectors.PolicyInfo{
					ARN:        *assignment.Properties.RoleDefinitionID,
					Type:       "ROLE_ASSIGNMENT",
					IsAttached: true,
				})
			}
		}
	}

	return policies, nil
}

// GetServiceAccounts returns service principals
func (c *Connector) GetServiceAccounts(ctx context.Context) ([]connectors.Principal, error) {
	var principals []connectors.Principal

	pager := c.authClient.NewListForSubscriptionPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing role assignments: %w", err)
		}

		for _, assignment := range page.Value {
			if assignment.Properties != nil && assignment.Properties.PrincipalType != nil {
				principalType := string(*assignment.Properties.PrincipalType)
				if principalType == "ServicePrincipal" {
					principals = append(principals, connectors.Principal{
						ARN:  *assignment.Properties.PrincipalID,
						Type: "SERVICE",
					})
				}
			}
		}
	}

	return principals, nil
}

// Helper functions

func extractResourceGroup(resourceID string) string {
	parts := strings.Split(resourceID, "/")
	for i, part := range parts {
		if part == "resourceGroups" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func int32Ptr(i int32) *int32 {
	return &i
}
