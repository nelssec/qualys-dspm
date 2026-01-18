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

// =====================================================
// Phase 2: Data Lineage Relationships
// =====================================================

// LineageEdge represents a data flow relationship
type LineageEdge struct {
	ID              uuid.UUID
	SourceARN       string
	SourceType      string
	TargetARN       string
	TargetType      string
	FlowType        string // READS_FROM, WRITES_TO, EXPORTS_TO, REPLICATES_TO
	AccessMethod    string
	InferredFrom    string
	ConfidenceScore float64
	Evidence        map[string]interface{}
}

// LineagePathResult represents a data lineage path
type LineagePathResult struct {
	OriginARN      string   `json:"origin_arn"`
	DestinationARN string   `json:"destination_arn"`
	Path           []string `json:"path"`
	FlowTypes      []string `json:"flow_types"`
	HopCount       int      `json:"hop_count"`
	Sensitive      bool     `json:"sensitive"`
}

// UpsertFunction creates or updates a serverless function node
func (g *Graph) UpsertFunction(ctx context.Context, accountID uuid.UUID, fn *FunctionNode) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (f:Function {arn: $arn})
		SET f.id = $id,
			f.name = $name,
			f.runtime = $runtime,
			f.executionRole = $executionRole,
			f.accountId = $accountId
		WITH f
		MATCH (acc:CloudAccount {id: $accountId})
		MERGE (f)-[:BELONGS_TO]->(acc)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":            fn.ID.String(),
		"arn":           fn.ARN,
		"name":          fn.Name,
		"runtime":       fn.Runtime,
		"executionRole": fn.ExecutionRole,
		"accountId":     accountID.String(),
	})

	return err
}

// CreateLineageEdge creates a data flow relationship between resources
func (g *Graph) CreateLineageEdge(ctx context.Context, edge *LineageEdge) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	// Determine the relationship type based on FlowType
	relType := "DATA_FLOW"
	switch edge.FlowType {
	case "READS_FROM":
		relType = "READS_FROM"
	case "WRITES_TO":
		relType = "WRITES_TO"
	case "EXPORTS_TO":
		relType = "EXPORTS_TO"
	case "REPLICATES_TO":
		relType = "REPLICATES_TO"
	}

	// Create source and target nodes if they don't exist
	query := `
		MERGE (source {arn: $sourceArn})
		ON CREATE SET source:DataResource, source.resourceType = $sourceType
		MERGE (target {arn: $targetArn})
		ON CREATE SET target:DataResource, target.resourceType = $targetType
		WITH source, target
		MERGE (source)-[r:` + relType + `]->(target)
		SET r.id = $id,
			r.flowType = $flowType,
			r.accessMethod = $accessMethod,
			r.inferredFrom = $inferredFrom,
			r.confidenceScore = $confidenceScore
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"id":              edge.ID.String(),
		"sourceArn":       edge.SourceARN,
		"sourceType":      edge.SourceType,
		"targetArn":       edge.TargetARN,
		"targetType":      edge.TargetType,
		"flowType":        edge.FlowType,
		"accessMethod":    edge.AccessMethod,
		"inferredFrom":    edge.InferredFrom,
		"confidenceScore": edge.ConfidenceScore,
	})

	return err
}

// FindDataFlowPaths finds all data flow paths from a source
func (g *Graph) FindDataFlowPaths(ctx context.Context, sourceARN string, maxHops int) ([]LineagePathResult, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	if maxHops <= 0 {
		maxHops = 5
	}

	query := `
		MATCH path = (source {arn: $sourceArn})-[:READS_FROM|WRITES_TO|EXPORTS_TO|REPLICATES_TO*1..` + fmt.Sprintf("%d", maxHops) + `]->(target)
		RETURN source.arn as origin,
			   target.arn as destination,
			   [n in nodes(path) | n.arn] as pathNodes,
			   [r in relationships(path) | type(r)] as flowTypes,
			   length(path) as hops
		LIMIT 100
	`

	result, err := session.Run(ctx, query, map[string]interface{}{
		"sourceArn": sourceARN,
	})
	if err != nil {
		return nil, fmt.Errorf("executing lineage query: %w", err)
	}

	var paths []LineagePathResult
	for result.Next(ctx) {
		record := result.Record()
		origin, _ := record.Get("origin")
		destination, _ := record.Get("destination")
		pathNodes, _ := record.Get("pathNodes")
		flowTypes, _ := record.Get("flowTypes")
		hops, _ := record.Get("hops")

		path := LineagePathResult{
			OriginARN:      origin.(string),
			DestinationARN: destination.(string),
			HopCount:       int(hops.(int64)),
		}

		if nodes, ok := pathNodes.([]interface{}); ok {
			for _, n := range nodes {
				if s, ok := n.(string); ok {
					path.Path = append(path.Path, s)
				}
			}
		}

		if types, ok := flowTypes.([]interface{}); ok {
			for _, t := range types {
				if s, ok := t.(string); ok {
					path.FlowTypes = append(path.FlowTypes, s)
				}
			}
		}

		paths = append(paths, path)
	}

	return paths, nil
}

// FindSensitiveDataFlows finds all data flows involving sensitive data
func (g *Graph) FindSensitiveDataFlows(ctx context.Context, accountID *uuid.UUID) ([]LineagePathResult, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (source:DataAsset)-[:READS_FROM|WRITES_TO|EXPORTS_TO|REPLICATES_TO]->(target)
		WHERE source.sensitivityLevel IN ['CRITICAL', 'HIGH']
	`

	if accountID != nil {
		query += ` AND source.accountId = $accountId`
	}

	query += `
		RETURN source.arn as origin,
			   target.arn as destination,
			   source.sensitivityLevel as sensitivity
		LIMIT 100
	`

	params := map[string]interface{}{}
	if accountID != nil {
		params["accountId"] = accountID.String()
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("executing sensitive flow query: %w", err)
	}

	var paths []LineagePathResult
	for result.Next(ctx) {
		record := result.Record()
		origin, _ := record.Get("origin")
		destination, _ := record.Get("destination")

		paths = append(paths, LineagePathResult{
			OriginARN:      origin.(string),
			DestinationARN: destination.(string),
			Sensitive:      true,
			HopCount:       1,
		})
	}

	return paths, nil
}

// GetLineageForAsset returns the complete lineage graph for an asset
func (g *Graph) GetLineageForAsset(ctx context.Context, assetARN string) (*LineageGraph, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	graph := &LineageGraph{
		Nodes: []LineageNode{},
		Edges: []LineageGraphEdge{},
	}

	// Get upstream lineage (what feeds into this asset)
	upstreamQuery := `
		MATCH path = (source)-[r:READS_FROM|WRITES_TO|EXPORTS_TO|REPLICATES_TO*1..3]->(target {arn: $assetArn})
		UNWIND nodes(path) as n
		UNWIND relationships(path) as rel
		RETURN DISTINCT n.arn as arn, n.name as name, n.resourceType as type,
			   startNode(rel).arn as relSource, endNode(rel).arn as relTarget, type(rel) as relType
	`

	result, err := session.Run(ctx, upstreamQuery, map[string]interface{}{
		"assetArn": assetARN,
	})
	if err != nil {
		return nil, fmt.Errorf("executing upstream query: %w", err)
	}

	nodeSet := make(map[string]bool)
	edgeSet := make(map[string]bool)

	for result.Next(ctx) {
		record := result.Record()

		// Add node
		arn, _ := record.Get("arn")
		if arn != nil && !nodeSet[arn.(string)] {
			name, _ := record.Get("name")
			nodeType, _ := record.Get("type")
			node := LineageNode{
				ARN:  arn.(string),
				Name: fmt.Sprintf("%v", name),
				Type: fmt.Sprintf("%v", nodeType),
			}
			graph.Nodes = append(graph.Nodes, node)
			nodeSet[arn.(string)] = true
		}

		// Add edge
		relSource, _ := record.Get("relSource")
		relTarget, _ := record.Get("relTarget")
		relType, _ := record.Get("relType")
		if relSource != nil && relTarget != nil {
			edgeKey := fmt.Sprintf("%s-%s-%s", relSource, relTarget, relType)
			if !edgeSet[edgeKey] {
				edge := LineageGraphEdge{
					Source:   relSource.(string),
					Target:   relTarget.(string),
					FlowType: relType.(string),
				}
				graph.Edges = append(graph.Edges, edge)
				edgeSet[edgeKey] = true
			}
		}
	}

	return graph, nil
}

// FunctionNode represents a serverless function in the graph
type FunctionNode struct {
	ID            uuid.UUID
	ARN           string
	Name          string
	Runtime       string
	ExecutionRole string
}

// LineageGraph represents a complete data lineage graph
type LineageGraph struct {
	Nodes []LineageNode      `json:"nodes"`
	Edges []LineageGraphEdge `json:"edges"`
}

// LineageNode represents a node in the lineage graph
type LineageNode struct {
	ARN              string   `json:"arn"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	SensitivityLevel string   `json:"sensitivity_level,omitempty"`
	DataCategories   []string `json:"data_categories,omitempty"`
}

// LineageGraphEdge represents an edge in the lineage graph
type LineageGraphEdge struct {
	Source          string  `json:"source"`
	Target          string  `json:"target"`
	FlowType        string  `json:"flow_type"`
	ConfidenceScore float64 `json:"confidence_score,omitempty"`
}

// =====================================================
// Phase 2: AI Source Tracking Relationships
// =====================================================

// UpsertAIModel creates or updates an AI model node
func (g *Graph) UpsertAIModel(ctx context.Context, accountID uuid.UUID, modelARN, modelName, modelType string) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MERGE (m:AIModel {arn: $arn})
		SET m.name = $name,
			m.modelType = $modelType,
			m.accountId = $accountId
		WITH m
		MATCH (acc:CloudAccount {id: $accountId})
		MERGE (m)-[:BELONGS_TO]->(acc)
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"arn":       modelARN,
		"name":      modelName,
		"modelType": modelType,
		"accountId": accountID.String(),
	})

	return err
}

// CreateTrainingDataEdge creates a TRAINED_ON relationship between model and data
func (g *Graph) CreateTrainingDataEdge(ctx context.Context, modelARN, dataSourceARN string, sensitivityLevel string) error {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (m:AIModel {arn: $modelArn})
		MERGE (d:DataSource {arn: $dataSourceArn})
		MERGE (m)-[r:TRAINED_ON]->(d)
		SET r.sensitivityLevel = $sensitivityLevel
	`

	_, err := session.Run(ctx, query, map[string]interface{}{
		"modelArn":         modelARN,
		"dataSourceArn":    dataSourceARN,
		"sensitivityLevel": sensitivityLevel,
	})

	return err
}

// FindAIModelsAccessingSensitiveData finds AI models trained on sensitive data
func (g *Graph) FindAIModelsAccessingSensitiveData(ctx context.Context, accountID *uuid.UUID) ([]AIModelAccessRecord, error) {
	session := g.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	query := `
		MATCH (m:AIModel)-[r:TRAINED_ON]->(d)
		WHERE r.sensitivityLevel IN ['CRITICAL', 'HIGH']
	`

	if accountID != nil {
		query += ` AND m.accountId = $accountId`
	}

	query += `
		RETURN m.arn as modelArn,
			   m.name as modelName,
			   m.modelType as modelType,
			   d.arn as dataSourceArn,
			   r.sensitivityLevel as sensitivityLevel
	`

	params := map[string]interface{}{}
	if accountID != nil {
		params["accountId"] = accountID.String()
	}

	result, err := session.Run(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("executing AI model query: %w", err)
	}

	var records []AIModelAccessRecord
	for result.Next(ctx) {
		rec := result.Record()
		modelArn, _ := rec.Get("modelArn")
		modelName, _ := rec.Get("modelName")
		modelType, _ := rec.Get("modelType")
		dataSourceArn, _ := rec.Get("dataSourceArn")
		sensitivityLevel, _ := rec.Get("sensitivityLevel")

		records = append(records, AIModelAccessRecord{
			ModelARN:         modelArn.(string),
			ModelName:        modelName.(string),
			ModelType:        modelType.(string),
			DataSourceARN:    dataSourceArn.(string),
			SensitivityLevel: sensitivityLevel.(string),
		})
	}

	return records, nil
}

// AIModelAccessRecord represents an AI model's access to data
type AIModelAccessRecord struct {
	ModelARN         string `json:"model_arn"`
	ModelName        string `json:"model_name"`
	ModelType        string `json:"model_type"`
	DataSourceARN    string `json:"data_source_arn"`
	SensitivityLevel string `json:"sensitivity_level"`
}
