package access

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/qualys/dspm/internal/models"
)

type Graph struct {
	driver neo4j.DriverWithContext
}

type Config struct {
	URI      string
	Username string
	Password string
}

func New(cfg Config) (*Graph, error) {
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("creating neo4j driver: %w", err)
	}

	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("verifying neo4j connectivity: %w", err)
	}

	g := &Graph{driver: driver}

	if err := g.createIndexes(ctx); err != nil {
		return nil, fmt.Errorf("creating indexes: %w", err)
	}

	return g, nil
}

func (g *Graph) Close(ctx context.Context) error {
	return g.driver.Close(ctx)
}

func (g *Graph) createIndexes(ctx context.Context) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS FOR (n:CloudAccount) ON (n.id)",
		"CREATE INDEX IF NOT EXISTS FOR (n:DataAsset) ON (n.id)",
		"CREATE INDEX IF NOT EXISTS FOR (n:DataAsset) ON (n.arn)",
		"CREATE INDEX IF NOT EXISTS FOR (n:Principal) ON (n.id)",
		"CREATE INDEX IF NOT EXISTS FOR (n:Principal) ON (n.arn)",
		"CREATE INDEX IF NOT EXISTS FOR (n:Policy) ON (n.id)",
		"CREATE INDEX IF NOT EXISTS FOR (n:Classification) ON (n.category)",
	}

	for _, idx := range indexes {
		_, err := session.Run(ctx, idx, nil)
		if err != nil {
			return fmt.Errorf("creating index: %w", err)
		}
	}

	return nil
}

func (g *Graph) UpsertAccount(ctx context.Context, account *models.CloudAccount) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (a:CloudAccount {id: $id})
		SET a.provider = $provider,
			a.externalId = $externalId,
			a.displayName = $displayName,
			a.status = $status
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":          account.ID.String(),
		"provider":    string(account.Provider),
		"externalId":  account.ExternalID,
		"displayName": account.DisplayName,
		"status":      account.Status,
	})

	return err
}

func (g *Graph) UpsertAsset(ctx context.Context, asset *models.DataAsset) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (a:DataAsset {id: $id})
		SET a.arn = $arn,
			a.resourceType = $resourceType,
			a.name = $name,
			a.region = $region,
			a.sensitivityLevel = $sensitivityLevel,
			a.encryptionStatus = $encryptionStatus,
			a.publicAccess = $publicAccess
		WITH a
		MATCH (acc:CloudAccount {id: $accountId})
		MERGE (a)-[:BELONGS_TO]->(acc)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":               asset.ID.String(),
		"arn":              asset.ResourceARN,
		"resourceType":     string(asset.ResourceType),
		"name":             asset.Name,
		"region":           asset.Region,
		"sensitivityLevel": string(asset.SensitivityLevel),
		"encryptionStatus": string(asset.EncryptionStatus),
		"publicAccess":     asset.PublicAccess,
		"accountId":        asset.AccountID.String(),
	})

	return err
}

func (g *Graph) UpsertPrincipal(ctx context.Context, accountID uuid.UUID, principal *Principal) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (p:Principal {arn: $arn})
		SET p.id = $id,
			p.name = $name,
			p.type = $type,
			p.accountId = $accountId
		WITH p
		MATCH (acc:CloudAccount {id: $accountId})
		MERGE (p)-[:BELONGS_TO]->(acc)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":        principal.ID.String(),
		"arn":       principal.ARN,
		"name":      principal.Name,
		"type":      principal.Type,
		"accountId": accountID.String(),
	})

	return err
}

func (g *Graph) UpsertPolicy(ctx context.Context, policy *models.AccessPolicy) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (p:Policy {arn: $arn})
		SET p.id = $id,
			p.name = $name,
			p.policyType = $policyType,
			p.allowsPublicAccess = $allowsPublicAccess
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":                 policy.ID.String(),
		"arn":                policy.PolicyARN,
		"name":               policy.PolicyName,
		"policyType":         policy.PolicyType,
		"allowsPublicAccess": policy.AllowsPublicAccess,
	})

	return err
}

func (g *Graph) CreateAccessEdge(ctx context.Context, edge *models.AccessEdge) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (p:Principal {arn: $sourceArn})
		MATCH (a:DataAsset {id: $targetAssetId})
		MERGE (p)-[r:CAN_ACCESS]->(a)
		SET r.id = $id,
			r.permissions = $permissions,
			r.permissionLevel = $permissionLevel,
			r.isDirect = $isDirect,
			r.isPublic = $isPublic,
			r.isCrossAccount = $isCrossAccount
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":              edge.ID.String(),
		"sourceArn":       edge.SourceARN,
		"targetAssetId":   edge.TargetAssetID.String(),
		"permissions":     edge.Permissions,
		"permissionLevel": string(edge.PermissionLevel),
		"isDirect":        edge.IsDirect,
		"isPublic":        edge.IsPublic,
		"isCrossAccount":  edge.IsCrossAccount,
	})

	return err
}

func (g *Graph) CreatePublicAccess(ctx context.Context, assetID uuid.UUID, permissions []string) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (p:Principal {arn: 'public', type: 'PUBLIC'})
		WITH p
		MATCH (a:DataAsset {id: $assetId})
		MERGE (p)-[r:CAN_ACCESS]->(a)
		SET r.isPublic = true,
			r.permissions = $permissions
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"assetId":     assetID.String(),
		"permissions": permissions,
	})

	return err
}

func (g *Graph) CreateRoleAssumption(ctx context.Context, sourceARN, targetRoleARN string) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (source:Principal {arn: $sourceArn})
		MATCH (target:Principal {arn: $targetArn, type: 'ROLE'})
		MERGE (source)-[r:CAN_ASSUME]->(target)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"sourceArn": sourceARN,
		"targetArn": targetRoleARN,
	})

	return err
}

func (g *Graph) CreatePolicyAttachment(ctx context.Context, policyARN, principalARN string) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (policy:Policy {arn: $policyArn})
		MATCH (principal:Principal {arn: $principalArn})
		MERGE (policy)-[r:ATTACHED_TO]->(principal)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"policyArn":    policyARN,
		"principalArn": principalARN,
	})

	return err
}

func (g *Graph) AddClassification(ctx context.Context, assetID uuid.UUID, category models.Category, sensitivity models.Sensitivity, count int) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (c:Classification {category: $category, sensitivity: $sensitivity})
		WITH c
		MATCH (a:DataAsset {id: $assetId})
		MERGE (a)-[r:CONTAINS_DATA]->(c)
		SET r.count = $count
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"assetId":     assetID.String(),
		"category":    string(category),
		"sensitivity": string(sensitivity),
		"count":       count,
	})

	return err
}

type PathResult struct {
	Source      string   `json:"source"`
	Target      string   `json:"target"`
	Path        []string `json:"path"`
	Permissions []string `json:"permissions"`
	IsPublic    bool     `json:"is_public"`
	HopCount    int      `json:"hop_count"`
}

func (g *Graph) FindPublicAccessPaths(ctx context.Context, accountID *uuid.UUID, maxHops int) ([]PathResult, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH path = (p:Principal {type: 'PUBLIC'})-[:CAN_ACCESS|CAN_ASSUME*1..` + fmt.Sprintf("%d", maxHops) + `]->(a:DataAsset)
		WHERE a.sensitivityLevel IN ['CRITICAL', 'HIGH']
	`

	if accountID != nil {
		query += ` AND a.accountId = $accountId`
	}

	query += `
		RETURN p.arn as source,
			   a.arn as target,
			   [n in nodes(path) | coalesce(n.arn, n.name)] as pathNodes,
			   length(path) as hops
		ORDER BY hops ASC
		LIMIT 100
	`

	params := map[string]interface{}{}
	if accountID != nil {
		params["accountId"] = accountID.String()
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}

	var paths []PathResult
	for result.Next(ctx) {
		record := result.Record()
		source, _ := record.Get("source")
		target, _ := record.Get("target")
		pathNodes, _ := record.Get("pathNodes")
		hops, _ := record.Get("hops")

		path := PathResult{
			Source:   source.(string),
			Target:   target.(string),
			IsPublic: true,
			HopCount: int(hops.(int64)),
		}

		if nodes, ok := pathNodes.([]interface{}); ok {
			for _, n := range nodes {
				if s, ok := n.(string); ok {
					path.Path = append(path.Path, s)
				}
			}
		}

		paths = append(paths, path)
	}

	return paths, nil
}

func (g *Graph) FindAccessToPII(ctx context.Context, accountID *uuid.UUID) ([]AccessRecord, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (p:Principal)-[r:CAN_ACCESS]->(a:DataAsset)-[:CONTAINS_DATA]->(c:Classification)
		WHERE c.category = 'PII'
	`

	if accountID != nil {
		query += ` AND a.accountId = $accountId`
	}

	query += `
		RETURN p.arn as principal,
			   p.type as principalType,
			   a.arn as asset,
			   a.name as assetName,
			   r.permissions as permissions,
			   c.sensitivity as sensitivity
		ORDER BY c.sensitivity DESC
	`

	params := map[string]interface{}{}
	if accountID != nil {
		params["accountId"] = accountID.String()
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}

	var records []AccessRecord
	for result.Next(ctx) {
		rec := result.Record()
		principal, _ := rec.Get("principal")
		principalType, _ := rec.Get("principalType")
		asset, _ := rec.Get("asset")
		assetName, _ := rec.Get("assetName")
		permissions, _ := rec.Get("permissions")
		sensitivity, _ := rec.Get("sensitivity")

		record := AccessRecord{
			PrincipalARN:  principal.(string),
			PrincipalType: principalType.(string),
			AssetARN:      asset.(string),
			AssetName:     assetName.(string),
			Sensitivity:   sensitivity.(string),
		}

		if perms, ok := permissions.([]interface{}); ok {
			for _, p := range perms {
				if s, ok := p.(string); ok {
					record.Permissions = append(record.Permissions, s)
				}
			}
		}

		records = append(records, record)
	}

	return records, nil
}

func (g *Graph) FindOverprivilegedAccess(ctx context.Context, accountID *uuid.UUID) ([]AccessRecord, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (p:Principal)-[r:CAN_ACCESS]->(a:DataAsset)
		WHERE r.permissionLevel IN ['ADMIN', 'FULL']
		  AND a.sensitivityLevel IN ['CRITICAL', 'HIGH']
	`

	if accountID != nil {
		query += ` AND a.accountId = $accountId`
	}

	query += `
		RETURN p.arn as principal,
			   p.type as principalType,
			   a.arn as asset,
			   a.name as assetName,
			   r.permissionLevel as permissionLevel,
			   a.sensitivityLevel as sensitivity
		ORDER BY a.sensitivityLevel DESC, r.permissionLevel DESC
	`

	params := map[string]interface{}{}
	if accountID != nil {
		params["accountId"] = accountID.String()
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}

	var records []AccessRecord
	for result.Next(ctx) {
		rec := result.Record()
		principal, _ := rec.Get("principal")
		principalType, _ := rec.Get("principalType")
		asset, _ := rec.Get("asset")
		assetName, _ := rec.Get("assetName")
		permLevel, _ := rec.Get("permissionLevel")
		sensitivity, _ := rec.Get("sensitivity")

		records = append(records, AccessRecord{
			PrincipalARN:    principal.(string),
			PrincipalType:   principalType.(string),
			AssetARN:        asset.(string),
			AssetName:       assetName.(string),
			PermissionLevel: permLevel.(string),
			Sensitivity:     sensitivity.(string),
		})
	}

	return records, nil
}

func (g *Graph) FindCrossAccountAccess(ctx context.Context, accountID uuid.UUID) ([]AccessRecord, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (p:Principal)-[r:CAN_ACCESS]->(a:DataAsset)-[:BELONGS_TO]->(acc:CloudAccount {id: $accountId})
		WHERE r.isCrossAccount = true OR NOT (p)-[:BELONGS_TO]->(acc)
		RETURN p.arn as principal,
			   p.type as principalType,
			   a.arn as asset,
			   a.name as assetName,
			   r.permissions as permissions,
			   a.sensitivityLevel as sensitivity
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"accountId": accountID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}

	var records []AccessRecord
	for result.Next(ctx) {
		rec := result.Record()
		principal, _ := rec.Get("principal")
		principalType, _ := rec.Get("principalType")
		asset, _ := rec.Get("asset")
		assetName, _ := rec.Get("assetName")
		permissions, _ := rec.Get("permissions")
		sensitivity, _ := rec.Get("sensitivity")

		record := AccessRecord{
			PrincipalARN:  principal.(string),
			PrincipalType: principalType.(string),
			AssetARN:      asset.(string),
			AssetName:     assetName.(string),
			Sensitivity:   sensitivity.(string),
			CrossAccount:  true,
		}

		if perms, ok := permissions.([]interface{}); ok {
			for _, p := range perms {
				if s, ok := p.(string); ok {
					record.Permissions = append(record.Permissions, s)
				}
			}
		}

		records = append(records, record)
	}

	return records, nil
}

func (g *Graph) GetAccessStats(ctx context.Context, accountID *uuid.UUID) (*AccessStats, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	stats := &AccessStats{}

	query := `
		MATCH (p:Principal)
		WHERE $accountId IS NULL OR p.accountId = $accountId
		RETURN p.type as type, count(p) as count
	`

	params := map[string]interface{}{"accountId": nil}
	if accountID != nil {
		params["accountId"] = accountID.String()
	}

	result, err := session.Run(ctx, query, params)
	if err == nil {
		stats.PrincipalsByType = make(map[string]int)
		for result.Next(ctx) {
			rec := result.Record()
			pType, _ := rec.Get("type")
			count, _ := rec.Get("count")
			stats.PrincipalsByType[pType.(string)] = int(count.(int64))
		}
	}

	query = `
		MATCH (p:Principal {type: 'PUBLIC'})-[:CAN_ACCESS]->(a:DataAsset)
		WHERE $accountId IS NULL OR a.accountId = $accountId
		RETURN count(a) as count
	`

	result, err = session.Run(ctx, query, params)
	if err == nil && result.Next(ctx) {
		count, _ := result.Record().Get("count")
		stats.PublicAccessCount = int(count.(int64))
	}

	query = `
		MATCH ()-[r:CAN_ACCESS {isCrossAccount: true}]->()
		RETURN count(r) as count
	`

	result, err = session.Run(ctx, query, nil)
	if err == nil && result.Next(ctx) {
		count, _ := result.Record().Get("count")
		stats.CrossAccountCount = int(count.(int64))
	}

	query = `
		MATCH ()-[r:CAN_ACCESS]->(a:DataAsset)
		WHERE r.permissionLevel IN ['ADMIN', 'FULL']
		  AND a.sensitivityLevel IN ['CRITICAL', 'HIGH']
		RETURN count(r) as count
	`

	result, err = session.Run(ctx, query, nil)
	if err == nil && result.Next(ctx) {
		count, _ := result.Record().Get("count")
		stats.OverprivilegedCount = int(count.(int64))
	}

	return stats, nil
}

type Principal struct {
	ID   uuid.UUID
	ARN  string
	Name string
	Type string // USER, ROLE, SERVICE, GROUP
}

type AccessRecord struct {
	PrincipalARN    string   `json:"principal_arn"`
	PrincipalType   string   `json:"principal_type"`
	AssetARN        string   `json:"asset_arn"`
	AssetName       string   `json:"asset_name"`
	Permissions     []string `json:"permissions"`
	PermissionLevel string   `json:"permission_level"`
	Sensitivity     string   `json:"sensitivity"`
	CrossAccount    bool     `json:"cross_account"`
}

type AccessStats struct {
	PrincipalsByType    map[string]int `json:"principals_by_type"`
	PublicAccessCount   int            `json:"public_access_count"`
	CrossAccountCount   int            `json:"cross_account_count"`
	OverprivilegedCount int            `json:"overprivileged_count"`
}
