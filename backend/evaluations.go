package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateEvaluationQuestionRequest struct {
	QuestionID *uint   `json:"question_id"`
	Question   string  `json:"question"`
	Answer     string  `json:"answer"`
	Marks      float64 `json:"marks"`
	Feedback   string  `json:"feedback"`
}

type UpdateEvaluationRequest struct {
	Marks     *float64                          `json:"marks"`
	Feedback  *string                           `json:"feedback"`
	Questions []UpdateEvaluationQuestionRequest `json:"questions"`
}

type FinalizeEvaluationRequest struct {
	IsFinal bool `json:"is_final"`
}

type EvaluationQuestionSnapshot struct {
	ID                 uint      `json:"id"`
	QuestionText       string    `json:"question_text"`
	StudentAnswerText  string    `json:"student_answer_text"`
	Marks              float64   `json:"marks"`
	Feedback           string    `json:"feedback"`
	Confidence         float64   `json:"confidence"`
	CorrectPoints      []string  `json:"correct_points"`
	WrongPoints        []string  `json:"wrong_points"`
	MissingConcepts    []string  `json:"missing_concepts"`
	StrongTopics       []string  `json:"strong_topics"`
	WeakTopics         []string  `json:"weak_topics"`
	Mistakes           []string  `json:"mistakes"`
	Topics             []string  `json:"topics"`
	AIOriginalMarks    float64   `json:"ai_original_marks"`
	AIOriginalFeedback string    `json:"ai_original_feedback"`
	CreatedAt          time.Time `json:"created_at"`
}

type EvaluationSnapshot struct {
	ID                 uint                         `json:"id"`
	SubmissionID       uint                         `json:"submission_id"`
	Marks              float64                      `json:"marks"`
	Feedback           string                       `json:"feedback"`
	Confidence         float64                      `json:"confidence"`
	Mistakes           []string                     `json:"mistakes"`
	CorrectPoints      []string                     `json:"correct_points"`
	WrongPoints        []string                     `json:"wrong_points"`
	MissingConcepts    []string                     `json:"missing_concepts"`
	StrongTopics       []string                     `json:"strong_topics"`
	WeakTopics         []string                     `json:"weak_topics"`
	Topics             []string                     `json:"topics"`
	IsFinal            bool                         `json:"is_final"`
	AIOriginalMarks    float64                      `json:"ai_original_marks"`
	AIOriginalFeedback string                       `json:"ai_original_feedback"`
	FinalizedAt        *time.Time                   `json:"finalized_at,omitempty"`
	Questions          []EvaluationQuestionSnapshot `json:"questions"`
}

func UpdateEvaluationHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	professorID, evaluation, submission, assignment, err := getProfessorAndOwnedEvaluation(c)
	if err != nil {
		return
	}

	var req UpdateEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	if req.Feedback != nil {
		trimmed := strings.TrimSpace(*req.Feedback)
		req.Feedback = &trimmed
	}
	for i := range req.Questions {
		req.Questions[i].Question = strings.TrimSpace(req.Questions[i].Question)
		req.Questions[i].Answer = strings.TrimSpace(req.Questions[i].Answer)
		req.Questions[i].Feedback = strings.TrimSpace(req.Questions[i].Feedback)
	}

	if req.Marks == nil && req.Feedback == nil && len(req.Questions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Provide marks, feedback, or question overrides",
		})
		return
	}

	beforeSnapshot, err := buildEvaluationSnapshot(DB, evaluation.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to capture evaluation state",
		})
		return
	}

	if err := DB.Transaction(func(tx *gorm.DB) error {
		if len(req.Questions) > 0 {
			if err := applyQuestionOverrides(tx, evaluation.ID, req.Questions); err != nil {
				return err
			}
		}

		if req.Marks != nil {
			evaluation.Marks = *req.Marks
		} else if len(req.Questions) > 0 {
			var questionRows []EvaluationQuestion
			if err := tx.Where("evaluation_id = ?", evaluation.ID).Find(&questionRows).Error; err != nil {
				return err
			}
			var total float64
			for _, row := range questionRows {
				total += row.Marks
			}
			evaluation.Marks = total
		}

		if req.Feedback != nil {
			if *req.Feedback == "" {
				return errors.New("feedback is required")
			}
			evaluation.Feedback = *req.Feedback
		}

		// Mark as draft whenever overrides happen. Professor explicitly finalizes.
		evaluation.IsFinal = false
		evaluation.FinalizedAt = nil

		if err := tx.Save(&evaluation).Error; err != nil {
			return err
		}

		if _, err := recomputeAdaptiveProfile(tx, submission.StudentID, assignment.CourseID); err != nil {
			return err
		}

		afterSnapshot, err := buildEvaluationSnapshot(tx, evaluation.ID)
		if err != nil {
			return err
		}

		return insertEvaluationAuditLog(
			tx,
			evaluation.ID,
			professorID,
			"override_update",
			beforeSnapshot,
			afterSnapshot,
		)
	}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Evaluation updated",
	})
}

func FinalizeEvaluationHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	professorID, evaluation, submission, assignment, err := getProfessorAndOwnedEvaluation(c)
	if err != nil {
		return
	}

	var req FinalizeEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	beforeSnapshot, err := buildEvaluationSnapshot(DB, evaluation.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to capture evaluation state",
		})
		return
	}

	now := time.Now()
	if req.IsFinal {
		evaluation.IsFinal = true
		evaluation.FinalizedAt = &now
	} else {
		evaluation.IsFinal = false
		evaluation.FinalizedAt = nil
	}

	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&evaluation).Error; err != nil {
			return err
		}
		if _, err := recomputeAdaptiveProfile(tx, submission.StudentID, assignment.CourseID); err != nil {
			return err
		}
		afterSnapshot, err := buildEvaluationSnapshot(tx, evaluation.ID)
		if err != nil {
			return err
		}
		return insertEvaluationAuditLog(
			tx,
			evaluation.ID,
			professorID,
			"finalize_toggle",
			beforeSnapshot,
			afterSnapshot,
		)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update evaluation finalization",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Evaluation finalization updated",
		"is_final": evaluation.IsFinal,
	})
}

func ListEvaluationAuditLogsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	_, evaluation, _, _, err := getProfessorAndOwnedEvaluation(c)
	if err != nil {
		return
	}

	var logs []EvaluationAuditLog
	if err := DB.Where("evaluation_id = ?", evaluation.ID).Order("id DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch evaluation audit logs",
		})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func applyQuestionOverrides(tx *gorm.DB, evaluationID uint, updates []UpdateEvaluationQuestionRequest) error {
	for _, update := range updates {
		if update.Feedback == "" {
			return errors.New("question feedback is required")
		}

		if update.QuestionID != nil && *update.QuestionID != 0 {
			var existing EvaluationQuestion
			if err := tx.Where("id = ? AND evaluation_id = ?", *update.QuestionID, evaluationID).First(&existing).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("question override target not found")
				}
				return err
			}

			existing.Marks = update.Marks
			existing.Feedback = update.Feedback
			if update.Question != "" {
				existing.QuestionText = update.Question
			}
			if update.Answer != "" {
				existing.StudentAnswerText = update.Answer
			}
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			continue
		}

		if update.Question == "" {
			return errors.New("question text is required when adding a question override")
		}

		newQuestion := EvaluationQuestion{
			EvaluationID:       evaluationID,
			QuestionText:       update.Question,
			StudentAnswerText:  update.Answer,
			Marks:              update.Marks,
			Feedback:           update.Feedback,
			Confidence:         0.7,
			AIOriginalMarks:    update.Marks,
			AIOriginalFeedback: update.Feedback,
		}
		if err := tx.Create(&newQuestion).Error; err != nil {
			return err
		}
	}

	return nil
}

func getProfessorAndOwnedEvaluation(c *gin.Context) (uint, Evaluation, Submission, Assignment, error) {
	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "professor" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only professors can update evaluations",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, errors.New("forbidden")
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, errors.New("invalid token context")
	}
	professorID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, errors.New("invalid user id")
	}

	evaluationIDParam := c.Param("id")
	parsedEvaluationID, err := strconv.ParseUint(evaluationIDParam, 10, 64)
	if err != nil || parsedEvaluationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid evaluation id",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, errors.New("invalid evaluation id")
	}
	evaluationID := uint(parsedEvaluationID)

	var evaluation Evaluation
	if err := DB.Where("id = ?", evaluationID).First(&evaluation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Evaluation not found",
			})
			return 0, Evaluation{}, Submission{}, Assignment{}, err
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch evaluation",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, err
	}

	var submission Submission
	if err := DB.Where("id = ?", evaluation.SubmissionID).First(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify evaluation ownership",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, err
	}

	var assignment Assignment
	if err := DB.Where("id = ?", submission.AssignmentID).First(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify evaluation ownership",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, err
	}

	var course Course
	if err := DB.Where("id = ?", assignment.CourseID).First(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify evaluation ownership",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, err
	}

	if course.ProfessorID != professorID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You can only update evaluations in your own courses",
		})
		return 0, Evaluation{}, Submission{}, Assignment{}, errors.New("forbidden")
	}

	return professorID, evaluation, submission, assignment, nil
}

func buildEvaluationSnapshot(tx *gorm.DB, evaluationID uint) (EvaluationSnapshot, error) {
	var evaluation Evaluation
	if err := tx.Where("id = ?", evaluationID).First(&evaluation).Error; err != nil {
		return EvaluationSnapshot{}, err
	}

	var questions []EvaluationQuestion
	if err := tx.Where("evaluation_id = ?", evaluationID).Order("id ASC").Find(&questions).Error; err != nil {
		return EvaluationSnapshot{}, err
	}

	questionSnapshots := make([]EvaluationQuestionSnapshot, 0, len(questions))
	for _, question := range questions {
		questionSnapshots = append(questionSnapshots, EvaluationQuestionSnapshot{
			ID:                 question.ID,
			QuestionText:       question.QuestionText,
			StudentAnswerText:  question.StudentAnswerText,
			Marks:              question.Marks,
			Feedback:           question.Feedback,
			Confidence:         question.Confidence,
			CorrectPoints:      question.CorrectPoints,
			WrongPoints:        question.WrongPoints,
			MissingConcepts:    question.MissingConcepts,
			StrongTopics:       question.StrongTopics,
			WeakTopics:         question.WeakTopics,
			Mistakes:           question.Mistakes,
			Topics:             question.Topics,
			AIOriginalMarks:    question.AIOriginalMarks,
			AIOriginalFeedback: question.AIOriginalFeedback,
			CreatedAt:          question.CreatedAt,
		})
	}

	return EvaluationSnapshot{
		ID:                 evaluation.ID,
		SubmissionID:       evaluation.SubmissionID,
		Marks:              evaluation.Marks,
		Feedback:           evaluation.Feedback,
		Confidence:         evaluation.Confidence,
		Mistakes:           evaluation.Mistakes,
		CorrectPoints:      evaluation.CorrectPoints,
		WrongPoints:        evaluation.WrongPoints,
		MissingConcepts:    evaluation.MissingConcepts,
		StrongTopics:       evaluation.StrongTopics,
		WeakTopics:         evaluation.WeakTopics,
		Topics:             evaluation.Topics,
		IsFinal:            evaluation.IsFinal,
		AIOriginalMarks:    evaluation.AIOriginalMarks,
		AIOriginalFeedback: evaluation.AIOriginalFeedback,
		FinalizedAt:        evaluation.FinalizedAt,
		Questions:          questionSnapshots,
	}, nil
}

func insertEvaluationAuditLog(
	tx *gorm.DB,
	evaluationID uint,
	editorID uint,
	action string,
	before EvaluationSnapshot,
	after EvaluationSnapshot,
) error {
	beforeBytes, err := json.Marshal(before)
	if err != nil {
		return err
	}
	afterBytes, err := json.Marshal(after)
	if err != nil {
		return err
	}

	auditLog := EvaluationAuditLog{
		EvaluationID: evaluationID,
		EditorID:     editorID,
		Action:       action,
		BeforeState:  string(beforeBytes),
		AfterState:   string(afterBytes),
	}
	return tx.Create(&auditLog).Error
}
