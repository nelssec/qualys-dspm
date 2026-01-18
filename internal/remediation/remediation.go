package remediation

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Store defines the interface for remediation persistence
type Store interface {
	CreateAction(ctx context.Context, action *Action) error
	GetAction(ctx context.Context, id uuid.UUID) (*Action, error)
	UpdateAction(ctx context.Context, action *Action) error
	ListActions(ctx context.Context, accountID uuid.UUID, status *ActionStatus, limit, offset int) ([]Action, int, error)
	ListActionsForAsset(ctx context.Context, assetID uuid.UUID) ([]Action, error)
	GetActionSummary(ctx context.Context, accountID uuid.UUID) (*ActionSummary, error)
}

// Remediator defines the interface for executing remediation actions
type Remediator interface {
	Execute(ctx context.Context, action *Action) (*ExecuteResult, error)
	Rollback(ctx context.Context, action *Action) (*RollbackResult, error)
	ValidateAction(ctx context.Context, action *Action) error
}

// Service provides remediation management capabilities
type Service struct {
	store       Store
	remediators map[string]Remediator
	logger      *slog.Logger
}

// NewService creates a new remediation service
func NewService(store Store, logger *slog.Logger) *Service {
	return &Service{
		store:       store,
		remediators: make(map[string]Remediator),
		logger:      logger,
	}
}

// RegisterRemediator registers a remediator for a specific provider
func (s *Service) RegisterRemediator(provider string, remediator Remediator) {
	s.remediators[provider] = remediator
}

// CreateAction creates a new remediation action
func (s *Service) CreateAction(ctx context.Context, req CreateActionRequest) (*Action, error) {
	// Validate action type
	def := s.getActionDefinition(req.ActionType)
	if def == nil {
		return nil, fmt.Errorf("unknown action type: %s", req.ActionType)
	}

	// Validate required parameters
	if req.Parameters == nil {
		req.Parameters = make(map[string]interface{})
	}
	for _, param := range def.RequiredParams {
		if _, ok := req.Parameters[param]; !ok {
			return nil, fmt.Errorf("missing required parameter: %s", param)
		}
	}

	// Generate description if not provided
	description := req.Description
	if description == "" {
		description = def.Description
	}

	action := &Action{
		ID:                uuid.New(),
		AccountID:         req.AccountID,
		AssetID:           req.AssetID,
		FindingID:         req.FindingID,
		ActionType:        req.ActionType,
		Status:            StatusPending,
		RiskLevel:         def.RiskLevel,
		Description:       description,
		Parameters:        req.Parameters,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		RollbackAvailable: def.RollbackAvailable,
	}

	if err := s.store.CreateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("creating action: %w", err)
	}

	s.logger.Info("remediation action created",
		"action_id", action.ID,
		"action_type", action.ActionType,
		"asset_id", action.AssetID)

	return action, nil
}

// GetAction retrieves a remediation action by ID
func (s *Service) GetAction(ctx context.Context, id uuid.UUID) (*Action, error) {
	return s.store.GetAction(ctx, id)
}

// ListActions lists remediation actions with optional filtering
func (s *Service) ListActions(ctx context.Context, accountID uuid.UUID, status *ActionStatus, limit, offset int) ([]Action, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListActions(ctx, accountID, status, limit, offset)
}

// ListActionsForAsset lists all remediation actions for a specific asset
func (s *Service) ListActionsForAsset(ctx context.Context, assetID uuid.UUID) ([]Action, error) {
	return s.store.ListActionsForAsset(ctx, assetID)
}

// GetActionSummary returns a summary of remediation actions
func (s *Service) GetActionSummary(ctx context.Context, accountID uuid.UUID) (*ActionSummary, error) {
	return s.store.GetActionSummary(ctx, accountID)
}

// ApproveAction approves a pending remediation action
func (s *Service) ApproveAction(ctx context.Context, id uuid.UUID, req ApproveActionRequest) (*Action, error) {
	action, err := s.store.GetAction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting action: %w", err)
	}

	if action.Status != StatusPending {
		return nil, fmt.Errorf("action is not in pending status, current status: %s", action.Status)
	}

	now := time.Now()
	action.Status = StatusApproved
	action.ApprovedAt = &now
	action.ApprovedBy = req.ApprovedBy
	action.UpdatedAt = now

	if err := s.store.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("updating action: %w", err)
	}

	s.logger.Info("remediation action approved",
		"action_id", action.ID,
		"approved_by", req.ApprovedBy)

	return action, nil
}

// RejectAction rejects a pending remediation action
func (s *Service) RejectAction(ctx context.Context, id uuid.UUID, reason string) (*Action, error) {
	action, err := s.store.GetAction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting action: %w", err)
	}

	if action.Status != StatusPending {
		return nil, fmt.Errorf("action is not in pending status, current status: %s", action.Status)
	}

	action.Status = StatusRejected
	action.ErrorMessage = reason
	action.UpdatedAt = time.Now()

	if err := s.store.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("updating action: %w", err)
	}

	s.logger.Info("remediation action rejected",
		"action_id", action.ID,
		"reason", reason)

	return action, nil
}

// ExecuteAction executes an approved remediation action
func (s *Service) ExecuteAction(ctx context.Context, id uuid.UUID, provider string) (*Action, error) {
	action, err := s.store.GetAction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting action: %w", err)
	}

	if action.Status != StatusApproved {
		return nil, fmt.Errorf("action is not in approved status, current status: %s", action.Status)
	}

	remediator, ok := s.remediators[provider]
	if !ok {
		return nil, fmt.Errorf("no remediator registered for provider: %s", provider)
	}

	// Validate before execution
	if err := remediator.ValidateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("validating action: %w", err)
	}

	// Update status to executing
	action.Status = StatusExecuting
	now := time.Now()
	action.ExecutedAt = &now
	action.UpdatedAt = now
	if err := s.store.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("updating action status: %w", err)
	}

	// Execute the action
	s.logger.Info("executing remediation action",
		"action_id", action.ID,
		"action_type", action.ActionType)

	result, err := remediator.Execute(ctx, action)
	if err != nil {
		action.Status = StatusFailed
		action.ErrorMessage = err.Error()
		action.UpdatedAt = time.Now()
		s.store.UpdateAction(ctx, action)
		return action, fmt.Errorf("executing action: %w", err)
	}

	if !result.Success {
		action.Status = StatusFailed
		action.ErrorMessage = result.ErrorMessage
		action.UpdatedAt = time.Now()
		s.store.UpdateAction(ctx, action)
		return action, fmt.Errorf("action execution failed: %s", result.ErrorMessage)
	}

	// Update action with results
	completedAt := time.Now()
	action.Status = StatusCompleted
	action.PreviousState = result.PreviousState
	action.NewState = result.NewState
	action.CompletedAt = &completedAt
	action.UpdatedAt = completedAt

	if err := s.store.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("updating action: %w", err)
	}

	s.logger.Info("remediation action completed",
		"action_id", action.ID,
		"action_type", action.ActionType)

	return action, nil
}

// RollbackAction rolls back a completed remediation action
func (s *Service) RollbackAction(ctx context.Context, id uuid.UUID, provider string) (*Action, error) {
	action, err := s.store.GetAction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting action: %w", err)
	}

	if action.Status != StatusCompleted {
		return nil, fmt.Errorf("action is not in completed status, current status: %s", action.Status)
	}

	if !action.RollbackAvailable {
		return nil, fmt.Errorf("rollback is not available for this action type")
	}

	if action.PreviousState == nil {
		return nil, fmt.Errorf("no previous state available for rollback")
	}

	remediator, ok := s.remediators[provider]
	if !ok {
		return nil, fmt.Errorf("no remediator registered for provider: %s", provider)
	}

	s.logger.Info("rolling back remediation action",
		"action_id", action.ID,
		"action_type", action.ActionType)

	result, err := remediator.Rollback(ctx, action)
	if err != nil {
		return nil, fmt.Errorf("rolling back action: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("rollback failed: %s", result.ErrorMessage)
	}

	action.Status = StatusRolledBack
	action.UpdatedAt = time.Now()

	if err := s.store.UpdateAction(ctx, action); err != nil {
		return nil, fmt.Errorf("updating action: %w", err)
	}

	s.logger.Info("remediation action rolled back",
		"action_id", action.ID,
		"action_type", action.ActionType)

	return action, nil
}

// GetActionDefinitions returns all available action definitions
func (s *Service) GetActionDefinitions() []ActionDefinition {
	return GetActionDefinitions()
}

// GetPlaybooks returns predefined remediation playbooks
func (s *Service) GetPlaybooks() []Playbook {
	return []Playbook{
		{
			ID:          "secure-s3-bucket",
			Name:        "Secure S3 Bucket",
			Description: "Apply standard security controls to an S3 bucket",
			Actions: []ActionType{
				ActionEnableBucketEncryption,
				ActionBlockPublicAccess,
				ActionEnableVersioning,
				ActionEnableLogging,
			},
			RiskLevel:   RiskMedium,
			AutoApprove: false,
		},
		{
			ID:          "encryption-compliance",
			Name:        "Encryption Compliance",
			Description: "Ensure encryption best practices are followed",
			Actions: []ActionType{
				ActionEnableBucketEncryption,
				ActionEnableKMSRotation,
				ActionUpgradeTLS,
			},
			RiskLevel:   RiskLow,
			AutoApprove: false,
		},
		{
			ID:          "public-exposure-fix",
			Name:        "Fix Public Exposure",
			Description: "Remove public access from exposed resources",
			Actions: []ActionType{
				ActionBlockPublicAccess,
				ActionRevokePublicACL,
			},
			RiskLevel:   RiskMedium,
			AutoApprove: false,
		},
	}
}

func (s *Service) getActionDefinition(actionType ActionType) *ActionDefinition {
	for _, def := range GetActionDefinitions() {
		if def.ActionType == actionType {
			return &def
		}
	}
	return nil
}
