package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/qualys/dspm/internal/models"
)

// =====================================================
// ML Models Store Methods
// =====================================================

func (s *Store) CreateMLModel(ctx context.Context, model *models.MLModel) error {
	query := `
		INSERT INTO ml_models (
			id, name, model_type, version, description, framework, model_path,
			config, accuracy, precision_score, recall_score, f1_score,
			status, is_default, trained_on_samples, training_data_version,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (name, version) DO UPDATE SET
			status = EXCLUDED.status,
			accuracy = EXCLUDED.accuracy,
			precision_score = EXCLUDED.precision_score,
			recall_score = EXCLUDED.recall_score,
			f1_score = EXCLUDED.f1_score,
			updated_at = EXCLUDED.updated_at
	`

	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
	}
	model.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		model.ID, model.Name, model.ModelType, model.Version, model.Description,
		model.Framework, model.ModelPath, model.Config, model.Accuracy,
		model.PrecisionScore, model.RecallScore, model.F1Score, model.Status,
		model.IsDefault, model.TrainedOnSamples, model.TrainingDataVersion,
		model.CreatedAt, model.UpdatedAt,
	)
	return err
}

func (s *Store) UpdateMLModel(ctx context.Context, model *models.MLModel) error {
	query := `
		UPDATE ml_models SET
			status = $1, accuracy = $2, precision_score = $3, recall_score = $4,
			f1_score = $5, is_default = $6, updated_at = $7
		WHERE id = $8
	`
	_, err := s.db.ExecContext(ctx, query,
		model.Status, model.Accuracy, model.PrecisionScore, model.RecallScore,
		model.F1Score, model.IsDefault, time.Now(), model.ID,
	)
	return err
}

func (s *Store) GetMLModel(ctx context.Context, id uuid.UUID) (*models.MLModel, error) {
	var model models.MLModel
	query := `SELECT * FROM ml_models WHERE id = $1`
	err := s.db.GetContext(ctx, &model, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &model, err
}

func (s *Store) GetDefaultMLModel(ctx context.Context, modelType models.MLModelType) (*models.MLModel, error) {
	var model models.MLModel
	query := `SELECT * FROM ml_models WHERE model_type = $1 AND is_default = true AND status = 'active' LIMIT 1`
	err := s.db.GetContext(ctx, &model, query, modelType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &model, err
}

func (s *Store) ListMLModels(ctx context.Context) ([]*models.MLModel, error) {
	var models []*models.MLModel
	query := `SELECT * FROM ml_models ORDER BY created_at DESC`
	err := s.db.SelectContext(ctx, &models, query)
	return models, err
}

// =====================================================
// ML Predictions Store Methods
// =====================================================

func (s *Store) CreateMLPrediction(ctx context.Context, prediction *models.MLPrediction) error {
	query := `
		INSERT INTO ml_predictions (
			id, classification_id, model_id, prediction_type, predicted_label,
			confidence_score, entity_text, entity_start_offset, entity_end_offset,
			raw_output, review_status, reviewed_by, reviewed_at, review_notes,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	if prediction.ID == uuid.Nil {
		prediction.ID = uuid.New()
	}
	if prediction.CreatedAt.IsZero() {
		prediction.CreatedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, query,
		prediction.ID, prediction.ClassificationID, prediction.ModelID,
		prediction.PredictionType, prediction.PredictedLabel, prediction.ConfidenceScore,
		prediction.EntityText, prediction.EntityStartOffset, prediction.EntityEndOffset,
		prediction.RawOutput, prediction.ReviewStatus, prediction.ReviewedBy,
		prediction.ReviewedAt, prediction.ReviewNotes, prediction.CreatedAt,
	)
	return err
}

func (s *Store) UpdateMLPrediction(ctx context.Context, prediction *models.MLPrediction) error {
	query := `
		UPDATE ml_predictions SET
			review_status = $1, reviewed_by = $2, reviewed_at = $3, review_notes = $4
		WHERE id = $5
	`
	_, err := s.db.ExecContext(ctx, query,
		prediction.ReviewStatus, prediction.ReviewedBy, prediction.ReviewedAt,
		prediction.ReviewNotes, prediction.ID,
	)
	return err
}

func (s *Store) GetMLPrediction(ctx context.Context, id uuid.UUID) (*models.MLPrediction, error) {
	var prediction models.MLPrediction
	query := `SELECT * FROM ml_predictions WHERE id = $1`
	err := s.db.GetContext(ctx, &prediction, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &prediction, err
}

func (s *Store) ListMLPredictionsByClassification(ctx context.Context, classificationID uuid.UUID) ([]*models.MLPrediction, error) {
	var predictions []*models.MLPrediction
	query := `SELECT * FROM ml_predictions WHERE classification_id = $1 ORDER BY confidence_score DESC`
	err := s.db.SelectContext(ctx, &predictions, query, classificationID)
	return predictions, err
}

// =====================================================
// Classification Review Queue Store Methods
// =====================================================

func (s *Store) CreateReviewQueueItem(ctx context.Context, item *models.ClassificationReviewQueue) error {
	query := `
		INSERT INTO classification_review_queue (
			id, classification_id, prediction_id, priority, reason,
			original_confidence, assigned_to, assigned_at, due_by,
			status, resolved_at, resolution, final_label, final_confidence,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, query,
		item.ID, item.ClassificationID, item.PredictionID, item.Priority,
		item.Reason, item.OriginalConfidence, item.AssignedTo, item.AssignedAt,
		item.DueBy, item.Status, item.ResolvedAt, item.Resolution,
		item.FinalLabel, item.FinalConfidence, item.CreatedAt,
	)
	return err
}

func (s *Store) UpdateReviewQueueItem(ctx context.Context, item *models.ClassificationReviewQueue) error {
	query := `
		UPDATE classification_review_queue SET
			assigned_to = $1, assigned_at = $2, status = $3, resolved_at = $4,
			resolution = $5, final_label = $6, final_confidence = $7
		WHERE id = $8
	`
	_, err := s.db.ExecContext(ctx, query,
		item.AssignedTo, item.AssignedAt, item.Status, item.ResolvedAt,
		item.Resolution, item.FinalLabel, item.FinalConfidence, item.ID,
	)
	return err
}

func (s *Store) GetReviewQueueItem(ctx context.Context, id uuid.UUID) (*models.ClassificationReviewQueue, error) {
	var item models.ClassificationReviewQueue
	query := `SELECT * FROM classification_review_queue WHERE id = $1`
	err := s.db.GetContext(ctx, &item, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &item, err
}

func (s *Store) ListReviewQueue(ctx context.Context, status models.ReviewQueueStatus, limit int) ([]*models.ClassificationReviewQueue, error) {
	var items []*models.ClassificationReviewQueue
	query := `SELECT * FROM classification_review_queue WHERE status = $1 ORDER BY priority DESC, created_at ASC LIMIT $2`
	err := s.db.SelectContext(ctx, &items, query, status, limit)
	return items, err
}

func (s *Store) GetReviewQueueStats(ctx context.Context) (map[string]int, error) {
	query := `SELECT status, COUNT(*) as count FROM classification_review_queue GROUP BY status`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}
	return stats, nil
}

// =====================================================
// Training Feedback Store Methods
// =====================================================

func (s *Store) CreateTrainingFeedback(ctx context.Context, feedback *models.TrainingFeedback) error {
	query := `
		INSERT INTO training_feedback (
			id, model_id, prediction_id, original_prediction, corrected_label,
			feedback_type, sample_content, sample_hash, context_window,
			incorporated_in_training, training_run_id, submitted_by, submitted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	if feedback.ID == uuid.Nil {
		feedback.ID = uuid.New()
	}
	if feedback.SubmittedAt.IsZero() {
		feedback.SubmittedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, query,
		feedback.ID, feedback.ModelID, feedback.PredictionID,
		feedback.OriginalPrediction, feedback.CorrectedLabel, feedback.FeedbackType,
		feedback.SampleContent, feedback.SampleHash, feedback.ContextWindow,
		feedback.IncorporatedInTraining, feedback.TrainingRunID,
		feedback.SubmittedBy, feedback.SubmittedAt,
	)
	return err
}

func (s *Store) ListTrainingFeedback(ctx context.Context, modelID uuid.UUID, incorporated bool) ([]*models.TrainingFeedback, error) {
	var feedbacks []*models.TrainingFeedback
	query := `SELECT * FROM training_feedback WHERE model_id = $1 AND incorporated_in_training = $2 ORDER BY submitted_at DESC`
	err := s.db.SelectContext(ctx, &feedbacks, query, modelID, incorporated)
	return feedbacks, err
}

func (s *Store) MarkFeedbackIncorporated(ctx context.Context, feedbackIDs []uuid.UUID, trainingRunID string) error {
	query := `
		UPDATE training_feedback SET
			incorporated_in_training = true, training_run_id = $1
		WHERE id = ANY($2)
	`
	_, err := s.db.ExecContext(ctx, query, trainingRunID, pq.Array(feedbackIDs))
	return err
}

// =====================================================
// Classification Confidence Update
// =====================================================

func (s *Store) UpdateClassificationConfidence(ctx context.Context, id uuid.UUID, confidence float64, validated bool) error {
	query := `UPDATE classifications SET confidence_score = $1, validated = $2 WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, confidence, validated, id)
	return err
}

