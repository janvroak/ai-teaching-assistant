package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SubmitRequest struct {
	AssignmentID uint   `json:"assignment_id"`
	Content      string `json:"content"`
}

type AIEvaluateRequest struct {
	Question        string `json:"question"`
	StudentAnswer   string `json:"student_answer"`
	ReferenceAnswer string `json:"reference_answer"`
	EvaluationRubric []RubricCriterion `json:"evaluation_rubric"`
}

type AIEvaluateResponse struct {
	Marks           float64  `json:"marks"`
	Score           float64  `json:"score"`
	Feedback        string   `json:"feedback"`
	Confidence      float64  `json:"confidence"`
	Mistakes        []string `json:"mistakes"`
	CorrectPoints   []string `json:"correct_points"`
	WrongPoints     []string `json:"wrong_points"`
	MissingConcepts []string `json:"missing_concepts"`
	StrongTopics    []string `json:"strong_topics"`
	WeakTopics      []string `json:"weak_topics"`
	Topics          []string `json:"topics"`
}

type AIEvaluateFileResult struct {
	Question        string   `json:"question"`
	StudentAnswer   string   `json:"student_answer"`
	Marks           float64  `json:"marks"`
	Score           float64  `json:"score"`
	Feedback        string   `json:"feedback"`
	Confidence      float64  `json:"confidence"`
	Mistakes        []string `json:"mistakes"`
	CorrectPoints   []string `json:"correct_points"`
	WrongPoints     []string `json:"wrong_points"`
	MissingConcepts []string `json:"missing_concepts"`
	StrongTopics    []string `json:"strong_topics"`
	WeakTopics      []string `json:"weak_topics"`
	Topics          []string `json:"topics"`
}

type AIEvaluateFileResponse struct {
	ExtractedText   string                 `json:"extracted_text"`
	Results         []AIEvaluateFileResult `json:"results"`
	TotalMarks      float64                `json:"total_marks"`
	OverallFeedback string                 `json:"overall_feedback"`
	CommonMistakes  []string               `json:"common_mistakes"`
}

func assignmentRubric(assignment Assignment) []RubricCriterion {
	trimmed := strings.TrimSpace(assignment.RubricJSON)
	if trimmed == "" {
		return []RubricCriterion{}
	}
	var rubric []RubricCriterion
	if err := json.Unmarshal([]byte(trimmed), &rubric); err != nil {
		return []RubricCriterion{}
	}
	return rubric
}

func normalizedTopics(topics []string) []string {
	return normalizeStringList(topics, 20)
}

func collectTopicsFromFileResults(results []AIEvaluateFileResult) []string {
	topics := make([]string, 0)
	for _, result := range results {
		topics = append(topics, result.Topics...)
	}
	return normalizeStringList(topics, 20)
}

func collectMistakesFromFileResults(results []AIEvaluateFileResult) []string {
	mistakes := make([]string, 0)
	for _, result := range results {
		mistakes = append(mistakes, result.Mistakes...)
	}
	return normalizeStringList(mistakes, 40)
}

func collectCorrectPointsFromFileResults(results []AIEvaluateFileResult) []string {
	correctPoints := make([]string, 0)
	for _, result := range results {
		correctPoints = append(correctPoints, result.CorrectPoints...)
	}
	return normalizeStringList(correctPoints, 40)
}

func collectWrongPointsFromFileResults(results []AIEvaluateFileResult) []string {
	wrongPoints := make([]string, 0)
	for _, result := range results {
		wrongPoints = append(wrongPoints, result.WrongPoints...)
	}
	return normalizeStringList(wrongPoints, 40)
}

func collectMissingConceptsFromFileResults(results []AIEvaluateFileResult) []string {
	missingConcepts := make([]string, 0)
	for _, result := range results {
		missingConcepts = append(missingConcepts, result.MissingConcepts...)
	}
	return normalizeStringList(missingConcepts, 40)
}

func collectStrongTopicsFromFileResults(results []AIEvaluateFileResult) []string {
	topics := make([]string, 0)
	for _, result := range results {
		topics = append(topics, result.StrongTopics...)
	}
	return normalizeStringList(topics, 20)
}

func collectWeakTopicsFromFileResults(results []AIEvaluateFileResult) []string {
	topics := make([]string, 0)
	for _, result := range results {
		topics = append(topics, result.WeakTopics...)
	}
	return normalizeStringList(topics, 20)
}

type AssignmentSubmissionResponse struct {
	SubmissionID   uint   `json:"submission_id"`
	StudentID      uint   `json:"student_id"`
	Content        string `json:"content"`
	SubmissionType string `json:"submission_type"`
	SubmissionPDF  string `json:"submission_pdf_url,omitempty"`
	SubmissionFile string `json:"submission_file_url,omitempty"`
	Evaluation     gin.H  `json:"evaluation"`
}

type MySubmissionResponse struct {
	SubmissionID    uint   `json:"submission_id"`
	AssignmentID    uint   `json:"assignment_id"`
	AssignmentTitle string `json:"assignment_title"`
	Content         string `json:"content"`
	SubmissionType  string `json:"submission_type"`
	SubmissionPDF   string `json:"submission_pdf_url,omitempty"`
	SubmissionFile  string `json:"submission_file_url,omitempty"`
	Evaluation      gin.H  `json:"evaluation"`
}

func SubmitHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "student" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only students can submit assignments",
		})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return
	}
	studentID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return
	}

	var req SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.Content = strings.TrimSpace(req.Content)
	if req.AssignmentID == 0 || req.Content == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "assignment_id and content are required",
		})
		return
	}

	var assignment Assignment
	if err := DB.Where("id = ?", req.AssignmentID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Assignment not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch assignment",
		})
		return
	}

	var enrollment Enrollment
	if err := DB.Where("student_id = ? AND course_id = ?", studentID, assignment.CourseID).First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You are not enrolled in this course",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify enrollment",
		})
		return
	}

	var existingSubmission Submission
	if err := DB.Where("student_id = ? AND assignment_id = ?", studentID, req.AssignmentID).First(&existingSubmission).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "You have already submitted this assignment",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify existing submission",
		})
		return
	}

	evalReq := AIEvaluateRequest{
		Question:        strings.TrimSpace(assignment.Question),
		StudentAnswer:   req.Content,
		ReferenceAnswer: assignment.AnswerKey,
		EvaluationRubric: assignmentRubric(assignment),
	}
	if evalReq.Question == "" {
		evalReq.Question = assignment.Title
	}
	requestBytes, err := json.Marshal(evalReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to prepare evaluation payload",
		})
		return
	}

	fastAPIResponse, err := http.Post(
		fmt.Sprintf("%s/evaluate", aiServiceBaseURL()),
		"application/json",
		bytes.NewBuffer(requestBytes),
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Could not connect to AI evaluation service",
		})
		return
	}
	defer fastAPIResponse.Body.Close()

	responseBytes, err := io.ReadAll(fastAPIResponse.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read AI evaluation response",
		})
		return
	}

	if fastAPIResponse.StatusCode >= http.StatusBadRequest {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "AI evaluation service returned an error",
		})
		return
	}

	var evalResp AIEvaluateResponse
	if err := json.Unmarshal(responseBytes, &evalResp); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "AI evaluation response format is invalid",
		})
		return
	}

	evaluation := Evaluation{}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		submission := Submission{
			StudentID:    studentID,
			AssignmentID: req.AssignmentID,
			Content:      req.Content,
		}
		if err := tx.Create(&submission).Error; err != nil {
			return err
		}

		evaluation = Evaluation{
			SubmissionID:    submission.ID,
			Marks:           evalResp.Marks,
			Feedback:        evalResp.Feedback,
			Confidence:      evalResp.Confidence,
			Mistakes:        normalizeStringList(evalResp.Mistakes, 40),
			CorrectPoints:   normalizeStringList(evalResp.CorrectPoints, 40),
			WrongPoints:     normalizeStringList(evalResp.WrongPoints, 40),
			MissingConcepts: normalizeStringList(evalResp.MissingConcepts, 40),
			StrongTopics:    normalizeStringList(evalResp.StrongTopics, 20),
			WeakTopics:      normalizeStringList(evalResp.WeakTopics, 20),
			Topics: normalizedTopics(func() []string {
				if len(evalResp.Topics) > 0 {
					return evalResp.Topics
				}
				return append(append([]string{}, evalResp.StrongTopics...), evalResp.WeakTopics...)
			}()),
			IsFinal:            false,
			AIOriginalMarks:    evalResp.Marks,
			AIOriginalFeedback: evalResp.Feedback,
		}
		if err := tx.Create(&evaluation).Error; err != nil {
			return err
		}

		questionText := strings.TrimSpace(evalReq.Question)
		if questionText == "" {
			questionText = "Question 1"
		}
		questionRow := EvaluationQuestion{
			EvaluationID:       evaluation.ID,
			QuestionText:       questionText,
			StudentAnswerText:  strings.TrimSpace(req.Content),
			Marks:              evalResp.Marks,
			Feedback:           evalResp.Feedback,
			Confidence:         evalResp.Confidence,
			CorrectPoints:      normalizeStringList(evalResp.CorrectPoints, 40),
			WrongPoints:        normalizeStringList(evalResp.WrongPoints, 40),
			MissingConcepts:    normalizeStringList(evalResp.MissingConcepts, 40),
			StrongTopics:       normalizeStringList(evalResp.StrongTopics, 20),
			WeakTopics:         normalizeStringList(evalResp.WeakTopics, 20),
			Mistakes:           normalizeStringList(evalResp.Mistakes, 40),
			Topics:             normalizedTopics(evalResp.Topics),
			AIOriginalMarks:    evalResp.Marks,
			AIOriginalFeedback: evalResp.Feedback,
		}
		if err := tx.Create(&questionRow).Error; err != nil {
			return err
		}

		if err := updateAdaptiveProfileOnEvaluation(
			tx,
			studentID,
			assignment.CourseID,
			evaluation.Marks,
			evalReq.Question,
			evaluation.Mistakes,
		); err != nil {
			return err
		}

		return evaluateSubmissionIntegrity(tx, submission, assignment)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save submission evaluation",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"marks":            evaluation.Marks,
		"score":            evaluation.Marks,
		"feedback":         evaluation.Feedback,
		"confidence":       evaluation.Confidence,
		"correct_points":   evaluation.CorrectPoints,
		"wrong_points":     evaluation.WrongPoints,
		"missing_concepts": evaluation.MissingConcepts,
		"strong_topics":    evaluation.StrongTopics,
		"weak_topics":      evaluation.WeakTopics,
		"mistakes":         evaluation.Mistakes,
		"topics":           evaluation.Topics,
		"is_final":         evaluation.IsFinal,
	})
}

func SubmitFileHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "student" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only students can submit assignments",
		})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return
	}
	studentID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return
	}

	assignmentIDText := strings.TrimSpace(c.PostForm("assignment_id"))
	parsedAssignmentID, err := strconv.ParseUint(assignmentIDText, 10, 64)
	if err != nil || parsedAssignmentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "assignment_id is required in multipart form",
		})
		return
	}
	assignmentID := uint(parsedAssignmentID)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "file is required",
		})
		return
	}
	if !isSupportedSubmissionExtension(fileHeader.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Supported file formats: PDF, PNG, JPG, JPEG, WEBP",
		})
		return
	}

	var assignment Assignment
	if err := DB.Where("id = ?", assignmentID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Assignment not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch assignment",
		})
		return
	}

	var enrollment Enrollment
	if err := DB.Where("student_id = ? AND course_id = ?", studentID, assignment.CourseID).First(&enrollment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You are not enrolled in this course",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify enrollment",
		})
		return
	}

	var existingSubmission Submission
	if err := DB.Where("student_id = ? AND assignment_id = ?", studentID, assignmentID).First(&existingSubmission).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "You have already submitted this assignment",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify existing submission",
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Could not open uploaded file",
		})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Could not read uploaded file",
		})
		return
	}

	savedFilePath, err := saveSubmissionFile(fileHeader, fileBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to store uploaded submission file: %v", err),
		})
		return
	}

	var multipartBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBody)

	fileWriter, err := multipartWriter.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to prepare file upload payload",
		})
		return
	}
	if _, err := fileWriter.Write(fileBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to prepare file upload payload",
		})
		return
	}
	referenceAnswer := strings.TrimSpace(assignment.AnswerKey)
	if referenceAnswer == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Assignment answer key is missing",
		})
		return
	}
	if err := multipartWriter.WriteField("reference_answer", referenceAnswer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to prepare file upload payload",
		})
		return
	}
	rubricJSON := strings.TrimSpace(assignment.RubricJSON)
	if rubricJSON == "" {
		rubricJSON = "[]"
	}
	if err := multipartWriter.WriteField("rubric_json", rubricJSON); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to prepare file upload payload",
		})
		return
	}
	if err := multipartWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to finalize file upload payload",
		})
		return
	}

	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/evaluate-file", aiServiceBaseURL()), &multipartBody)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to prepare AI evaluation request",
		})
		return
	}
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	httpClient := &http.Client{}
	fastAPIResponse, err := httpClient.Do(request)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Could not connect to AI evaluation service",
		})
		return
	}
	defer fastAPIResponse.Body.Close()

	responseBytes, err := io.ReadAll(fastAPIResponse.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read AI evaluation response",
		})
		return
	}

	if fastAPIResponse.StatusCode >= http.StatusBadRequest {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "AI evaluation service returned an error",
		})
		return
	}

	var evalResp AIEvaluateFileResponse
	if err := json.Unmarshal(responseBytes, &evalResp); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "AI file evaluation response format is invalid",
		})
		return
	}

	content := strings.TrimSpace(evalResp.ExtractedText)
	if content == "" {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "AI evaluation service returned empty extracted text",
		})
		return
	}

	confidence := 0.0
	if len(evalResp.Results) > 0 {
		var sum float64
		for _, result := range evalResp.Results {
			sum += result.Confidence
		}
		confidence = sum / float64(len(evalResp.Results))
	}

	feedback := strings.TrimSpace(evalResp.OverallFeedback)
	if feedback == "" {
		feedback = "Evaluation completed from uploaded submission file."
	}

	evaluation := Evaluation{}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		submission := Submission{
			StudentID:    studentID,
			AssignmentID: assignmentID,
			Content:      content,
			FilePath:     savedFilePath,
		}
		if err := tx.Create(&submission).Error; err != nil {
			return err
		}

		evaluation = Evaluation{
			SubmissionID:    submission.ID,
			Marks:           evalResp.TotalMarks,
			Feedback:        feedback,
			Confidence:      confidence,
			Mistakes:        collectMistakesFromFileResults(evalResp.Results),
			CorrectPoints:   collectCorrectPointsFromFileResults(evalResp.Results),
			WrongPoints:     collectWrongPointsFromFileResults(evalResp.Results),
			MissingConcepts: collectMissingConceptsFromFileResults(evalResp.Results),
			StrongTopics:    collectStrongTopicsFromFileResults(evalResp.Results),
			WeakTopics:      collectWeakTopicsFromFileResults(evalResp.Results),
			Topics: normalizedTopics(func() []string {
				collected := collectTopicsFromFileResults(evalResp.Results)
				if len(collected) > 0 {
					return collected
				}
				return append(
					append([]string{}, collectStrongTopicsFromFileResults(evalResp.Results)...),
					collectWeakTopicsFromFileResults(evalResp.Results)...,
				)
			}()),
			IsFinal:            false,
			AIOriginalMarks:    evalResp.TotalMarks,
			AIOriginalFeedback: feedback,
		}
		if err := tx.Create(&evaluation).Error; err != nil {
			return err
		}

		for index, result := range evalResp.Results {
			questionText := strings.TrimSpace(result.Question)
			if questionText == "" {
				questionText = fmt.Sprintf("Question %d", index+1)
			}
			questionRow := EvaluationQuestion{
				EvaluationID:       evaluation.ID,
				QuestionText:       questionText,
				StudentAnswerText:  strings.TrimSpace(result.StudentAnswer),
				Marks:              result.Marks,
				Feedback:           strings.TrimSpace(result.Feedback),
				Confidence:         result.Confidence,
				CorrectPoints:      normalizeStringList(result.CorrectPoints, 40),
				WrongPoints:        normalizeStringList(result.WrongPoints, 40),
				MissingConcepts:    normalizeStringList(result.MissingConcepts, 40),
				StrongTopics:       normalizeStringList(result.StrongTopics, 20),
				WeakTopics:         normalizeStringList(result.WeakTopics, 20),
				Mistakes:           normalizeStringList(result.Mistakes, 40),
				Topics:             normalizedTopics(result.Topics),
				AIOriginalMarks:    result.Marks,
				AIOriginalFeedback: strings.TrimSpace(result.Feedback),
			}
			if questionRow.Feedback == "" {
				questionRow.Feedback = "No detailed feedback provided."
			}
			if err := tx.Create(&questionRow).Error; err != nil {
				return err
			}
		}

		adaptiveQuestion := strings.TrimSpace(assignment.Question)
		if adaptiveQuestion == "" {
			adaptiveQuestion = assignment.Title
		}
		if err := updateAdaptiveProfileOnEvaluation(
			tx,
			studentID,
			assignment.CourseID,
			evaluation.Marks,
			adaptiveQuestion,
			evaluation.Mistakes,
		); err != nil {
			return err
		}

		return evaluateSubmissionIntegrity(tx, submission, assignment)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save submission evaluation",
		})
		return
	}

	responseSubmission := Submission{
		ID:       evaluation.SubmissionID,
		FilePath: savedFilePath,
	}

	c.JSON(http.StatusCreated, gin.H{
		"submission_id":       evaluation.SubmissionID,
		"marks":               evaluation.Marks,
		"score":               evaluation.Marks,
		"feedback":            evaluation.Feedback,
		"confidence":          evaluation.Confidence,
		"correct_points":      evaluation.CorrectPoints,
		"wrong_points":        evaluation.WrongPoints,
		"missing_concepts":    evaluation.MissingConcepts,
		"strong_topics":       evaluation.StrongTopics,
		"weak_topics":         evaluation.WeakTopics,
		"mistakes":            evaluation.Mistakes,
		"topics":              evaluation.Topics,
		"submission_pdf_url":  submissionPDFURL(responseSubmission),
		"submission_file_url": submissionFileURL(responseSubmission),
		"is_final":            evaluation.IsFinal,
	})
}

func GetSubmissionsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "professor" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only professors can access assignment submissions",
		})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return
	}
	professorID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return
	}

	assignmentIDParam := c.Param("id")
	parsedAssignmentID, err := strconv.ParseUint(assignmentIDParam, 10, 64)
	if err != nil || parsedAssignmentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid assignment id",
		})
		return
	}
	assignmentID := uint(parsedAssignmentID)

	var assignment Assignment
	if err := DB.Where("id = ?", assignmentID).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Assignment not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch assignment",
		})
		return
	}

	var course Course
	if err := DB.Where("id = ?", assignment.CourseID).First(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify assignment ownership",
		})
		return
	}

	if course.ProfessorID != professorID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You can only view submissions for assignments in your own courses",
		})
		return
	}

	var submissions []Submission
	if err := DB.Model(&Submission{}).
		Where("assignment_id = ?", assignmentID).
		Distinct("id", "student_id", "content", "file_path", "created_at").
		Order("id ASC").
		Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch submissions",
		})
		return
	}

	submissionIDs := make([]uint, 0, len(submissions))
	for _, submission := range submissions {
		submissionIDs = append(submissionIDs, submission.ID)
	}

	evaluationBySubmissionID := make(map[uint]Evaluation, len(submissionIDs))

	if len(submissionIDs) > 0 {
		var evaluations []Evaluation
		if err := DB.
			Where("submission_id IN ?", submissionIDs).
			Order("id DESC").
			Find(&evaluations).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch evaluation data",
			})
			return
		}

		for _, evaluation := range evaluations {
			// Keep the latest evaluation for each submission.
			if _, exists := evaluationBySubmissionID[evaluation.SubmissionID]; !exists {
				evaluationBySubmissionID[evaluation.SubmissionID] = evaluation
			}
		}
	}

	evaluationQuestionsByEvaluationID := make(map[uint][]EvaluationQuestion)
	if len(evaluationBySubmissionID) > 0 {
		evaluationIDs := make([]uint, 0, len(evaluationBySubmissionID))
		for _, evaluation := range evaluationBySubmissionID {
			evaluationIDs = append(evaluationIDs, evaluation.ID)
		}
		var questionRows []EvaluationQuestion
		if err := DB.Where("evaluation_id IN ?", evaluationIDs).Order("id ASC").Find(&questionRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch question-wise evaluation data",
			})
			return
		}
		for _, row := range questionRows {
			evaluationQuestionsByEvaluationID[row.EvaluationID] = append(evaluationQuestionsByEvaluationID[row.EvaluationID], row)
		}
	}

	plagiarismBySubmissionID := make(map[uint]PlagiarismReport, len(submissionIDs))
	if len(submissionIDs) > 0 {
		var reports []PlagiarismReport
		if err := DB.Where("submission_id IN ?", submissionIDs).Order("id DESC").Find(&reports).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch plagiarism reports",
			})
			return
		}
		for _, row := range reports {
			if _, exists := plagiarismBySubmissionID[row.SubmissionID]; !exists {
				plagiarismBySubmissionID[row.SubmissionID] = row
			}
		}
	}

	response := make([]AssignmentSubmissionResponse, 0, len(submissions))
	for _, submission := range submissions {
		evaluationData := gin.H{
			"id":                   nil,
			"marks":                nil,
			"score":                nil,
			"feedback":             nil,
			"confidence":           nil,
			"is_final":             false,
			"ai_original_marks":    nil,
			"ai_original_feedback": nil,
			"question_results":     []EvaluationQuestion{},
			"reference_answer":     assignment.AnswerKey,
			"correct_points":       []string{},
			"wrong_points":         []string{},
			"missing_concepts":     []string{},
			"strong_topics":        []string{},
			"weak_topics":          []string{},
			"mistakes":             []string{},
			"topics":               []string{},
			"plagiarism":           nil,
		}
		if evaluation, exists := evaluationBySubmissionID[submission.ID]; exists {
			evaluationData["id"] = evaluation.ID
			evaluationData["marks"] = evaluation.Marks
			evaluationData["score"] = evaluation.Marks
			evaluationData["feedback"] = evaluation.Feedback
			evaluationData["confidence"] = evaluation.Confidence
			evaluationData["correct_points"] = evaluation.CorrectPoints
			evaluationData["wrong_points"] = evaluation.WrongPoints
			evaluationData["missing_concepts"] = evaluation.MissingConcepts
			evaluationData["strong_topics"] = evaluation.StrongTopics
			evaluationData["weak_topics"] = evaluation.WeakTopics
			evaluationData["mistakes"] = evaluation.Mistakes
			evaluationData["topics"] = evaluation.Topics
			evaluationData["is_final"] = evaluation.IsFinal
			evaluationData["ai_original_marks"] = evaluation.AIOriginalMarks
			evaluationData["ai_original_feedback"] = evaluation.AIOriginalFeedback
			evaluationData["question_results"] = evaluationQuestionsByEvaluationID[evaluation.ID]
			evaluationData["reference_answer"] = assignment.AnswerKey
		}
		if report, exists := plagiarismBySubmissionID[submission.ID]; exists {
			evaluationData["plagiarism"] = gin.H{
				"id":                   report.ID,
				"flagged":              report.Flagged,
				"semantic_similarity":  report.SemanticSimilarity,
				"style_anomaly_score":  report.StyleAnomalyScore,
				"ai_usage_likelihood":  report.AIUsageLikelihood,
				"plagiarism_likelihood": report.PlagiarismLikelihood,
				"interpretation_band":  interpretationBand(report.PlagiarismLikelihood),
				"reason_summary":       report.ReasonSummary,
				"reasons":              report.Reasons,
			}
		}

		response = append(response, AssignmentSubmissionResponse{
			SubmissionID:   submission.ID,
			StudentID:      submission.StudentID,
			Content:        submission.Content,
			SubmissionType: submissionType(submission),
			SubmissionPDF:  submissionPDFURL(submission),
			SubmissionFile: submissionFileURL(submission),
			Evaluation:     evaluationData,
		})
	}

	c.JSON(http.StatusOK, response)
}

func GetSubmissionHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	submissionIDParam := c.Param("id")
	parsedSubmissionID, err := strconv.ParseUint(submissionIDParam, 10, 64)
	if err != nil || parsedSubmissionID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid submission id",
		})
		return
	}
	submissionID := uint(parsedSubmissionID)

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return
	}
	role = strings.ToLower(strings.TrimSpace(role))

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return
	}

	var submission Submission
	if err := DB.Where("id = ?", submissionID).First(&submission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Submission not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch submission",
		})
		return
	}

	switch role {
	case "student":
		if submission.StudentID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only access your own submissions",
			})
			return
		}
	case "professor":
		var assignment Assignment
		if err := DB.Where("id = ?", submission.AssignmentID).First(&assignment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify submission access",
			})
			return
		}

		var course Course
		if err := DB.Where("id = ?", assignment.CourseID).First(&course).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify submission access",
			})
			return
		}

		if course.ProfessorID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only access submissions from your own courses",
			})
			return
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid role",
		})
		return
	}

	var evaluation Evaluation
	if err := DB.Where("submission_id = ?", submission.ID).First(&evaluation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Evaluation not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch evaluation",
		})
		return
	}

	var assignment Assignment
	if err := DB.Where("id = ?", submission.AssignmentID).First(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch assignment details",
		})
		return
	}

	var questionRows []EvaluationQuestion
	if err := DB.Where("evaluation_id = ?", evaluation.ID).Order("id ASC").Find(&questionRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch question-wise evaluation data",
		})
		return
	}

	isStudentViewer := role == "student"
	visibleToStudent := evaluation.IsFinal

	c.JSON(http.StatusOK, gin.H{
		"submission": gin.H{
			"id":                  submission.ID,
			"student_id":          submission.StudentID,
			"assignment_id":       submission.AssignmentID,
			"content":             submission.Content,
			"submission_type":     submissionType(submission),
			"submission_pdf":      submissionPDFURL(submission),
			"submission_pdf_url":  submissionPDFURL(submission),
			"submission_file":     submissionFileURL(submission),
			"submission_file_url": submissionFileURL(submission),
			"created_at":          submission.CreatedAt,
		},
		"evaluation": gin.H{
			"marks": func() interface{} {
				if isStudentViewer && !visibleToStudent {
					return nil
				}
				return evaluation.Marks
			}(),
			"score": func() interface{} {
				if isStudentViewer && !visibleToStudent {
					return nil
				}
				return evaluation.Marks
			}(),
			"feedback": func() interface{} {
				if isStudentViewer && !visibleToStudent {
					return nil
				}
				return evaluation.Feedback
			}(),
			"confidence": func() interface{} {
				if isStudentViewer && !visibleToStudent {
					return nil
				}
				return evaluation.Confidence
			}(),
			"correct_points": func() []string {
				if isStudentViewer && !visibleToStudent {
					return []string{}
				}
				return evaluation.CorrectPoints
			}(),
			"wrong_points": func() []string {
				if isStudentViewer && !visibleToStudent {
					return []string{}
				}
				return evaluation.WrongPoints
			}(),
			"missing_concepts": func() []string {
				if isStudentViewer && !visibleToStudent {
					return []string{}
				}
				return evaluation.MissingConcepts
			}(),
			"strong_topics": func() []string {
				if isStudentViewer && !visibleToStudent {
					return []string{}
				}
				return evaluation.StrongTopics
			}(),
			"weak_topics": func() []string {
				if isStudentViewer && !visibleToStudent {
					return []string{}
				}
				return evaluation.WeakTopics
			}(),
			"mistakes": func() []string {
				if isStudentViewer && !visibleToStudent {
					return []string{}
				}
				return evaluation.Mistakes
			}(),
			"topics": func() []string {
				if isStudentViewer && !visibleToStudent {
					return []string{}
				}
				return evaluation.Topics
			}(),
			"is_final": evaluation.IsFinal,
			"question_results": func() []EvaluationQuestion {
				if isStudentViewer && !visibleToStudent {
					return []EvaluationQuestion{}
				}
				return questionRows
			}(),
			"reference_answer": func() string {
				if isStudentViewer && !visibleToStudent {
					return ""
				}
				return assignment.AnswerKey
			}(),
			"ai_original_marks": func() interface{} {
				if isStudentViewer {
					return nil
				}
				return evaluation.AIOriginalMarks
			}(),
			"ai_original_feedback": func() interface{} {
				if isStudentViewer {
					return nil
				}
				return evaluation.AIOriginalFeedback
			}(),
		},
	})
}

func GetMySubmissionsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "student" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only students can access their submissions",
		})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return
	}
	studentID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return
	}

	var submissions []Submission
	if err := DB.Where("student_id = ?", studentID).Order("id DESC").Find(&submissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch submissions",
		})
		return
	}

	submissionIDs := make([]uint, 0, len(submissions))
	assignmentIDs := make([]uint, 0, len(submissions))
	for _, submission := range submissions {
		submissionIDs = append(submissionIDs, submission.ID)
		assignmentIDs = append(assignmentIDs, submission.AssignmentID)
	}

	evaluationBySubmissionID := make(map[uint]Evaluation, len(submissionIDs))
	assignmentTitleByID := make(map[uint]string, len(assignmentIDs))
	assignmentByID := make(map[uint]Assignment, len(assignmentIDs))

	if len(assignmentIDs) > 0 {
		var assignments []Assignment
		if err := DB.Model(&Assignment{}).Where("id IN ?", assignmentIDs).Find(&assignments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch assignment data",
			})
			return
		}

		for _, assignment := range assignments {
			assignmentTitleByID[assignment.ID] = assignment.Title
			assignmentByID[assignment.ID] = assignment
		}
	}

	if len(submissionIDs) > 0 {
		var evaluations []Evaluation
		if err := DB.Where("submission_id IN ?", submissionIDs).Order("id DESC").Find(&evaluations).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch evaluation data",
			})
			return
		}

		for _, evaluation := range evaluations {
			if _, exists := evaluationBySubmissionID[evaluation.SubmissionID]; !exists {
				evaluationBySubmissionID[evaluation.SubmissionID] = evaluation
			}
		}
	}

	evaluationQuestionsByEvaluationID := make(map[uint][]EvaluationQuestion)
	if len(evaluationBySubmissionID) > 0 {
		evaluationIDs := make([]uint, 0, len(evaluationBySubmissionID))
		for _, evaluation := range evaluationBySubmissionID {
			evaluationIDs = append(evaluationIDs, evaluation.ID)
		}
		var questionRows []EvaluationQuestion
		if err := DB.Where("evaluation_id IN ?", evaluationIDs).Order("id ASC").Find(&questionRows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch question-wise evaluation data",
			})
			return
		}
		for _, row := range questionRows {
			evaluationQuestionsByEvaluationID[row.EvaluationID] = append(evaluationQuestionsByEvaluationID[row.EvaluationID], row)
		}
	}

	response := make([]MySubmissionResponse, 0, len(submissions))
	for _, submission := range submissions {
		evaluationData := gin.H{
			"marks":            nil,
			"score":            nil,
			"feedback":         nil,
			"confidence":       nil,
			"is_final":         false,
			"question_results": []EvaluationQuestion{},
			"reference_answer": "",
			"correct_points":   []string{},
			"wrong_points":     []string{},
			"missing_concepts": []string{},
			"strong_topics":    []string{},
			"weak_topics":      []string{},
			"mistakes":         []string{},
			"topics":           []string{},
		}

		if evaluation, exists := evaluationBySubmissionID[submission.ID]; exists {
			evaluationData["is_final"] = evaluation.IsFinal
			if evaluation.IsFinal {
				evaluationData["marks"] = evaluation.Marks
				evaluationData["score"] = evaluation.Marks
				evaluationData["feedback"] = evaluation.Feedback
				evaluationData["confidence"] = evaluation.Confidence
				evaluationData["correct_points"] = evaluation.CorrectPoints
				evaluationData["wrong_points"] = evaluation.WrongPoints
				evaluationData["missing_concepts"] = evaluation.MissingConcepts
				evaluationData["strong_topics"] = evaluation.StrongTopics
				evaluationData["weak_topics"] = evaluation.WeakTopics
				evaluationData["mistakes"] = evaluation.Mistakes
				evaluationData["topics"] = evaluation.Topics
				evaluationData["question_results"] = evaluationQuestionsByEvaluationID[evaluation.ID]
				if assignment, ok := assignmentByID[submission.AssignmentID]; ok {
					evaluationData["reference_answer"] = assignment.AnswerKey
				}
			}
		}

		response = append(response, MySubmissionResponse{
			SubmissionID:    submission.ID,
			AssignmentID:    submission.AssignmentID,
			AssignmentTitle: assignmentTitleByID[submission.AssignmentID],
			Content:         submission.Content,
			SubmissionType:  submissionType(submission),
			SubmissionPDF:   submissionPDFURL(submission),
			SubmissionFile:  submissionFileURL(submission),
			Evaluation:      evaluationData,
		})
	}

	c.JSON(http.StatusOK, response)
}

func GetSubmissionFileHandler(c *gin.Context) {
	submission, err := getAccessibleSubmission(c)
	if err != nil {
		return
	}

	if strings.TrimSpace(submission.FilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Submission file not found",
		})
		return
	}

	c.File(submission.FilePath)
}

func getAccessibleSubmission(c *gin.Context) (Submission, error) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return Submission{}, errors.New("db not initialized")
	}

	submissionIDParam := c.Param("id")
	parsedSubmissionID, err := strconv.ParseUint(submissionIDParam, 10, 64)
	if err != nil || parsedSubmissionID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid submission id",
		})
		return Submission{}, errors.New("invalid submission id")
	}
	submissionID := uint(parsedSubmissionID)

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return Submission{}, errors.New("invalid role")
	}
	role = strings.ToLower(strings.TrimSpace(role))

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token context",
		})
		return Submission{}, errors.New("missing user id")
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return Submission{}, errors.New("invalid user id")
	}

	var submission Submission
	if err := DB.Where("id = ?", submissionID).First(&submission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Submission not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch submission",
			})
		}
		return Submission{}, err
	}

	switch role {
	case "student":
		if submission.StudentID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only access your own submissions",
			})
			return Submission{}, errors.New("forbidden")
		}
	case "professor":
		var assignment Assignment
		if err := DB.Where("id = ?", submission.AssignmentID).First(&assignment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify submission access",
			})
			return Submission{}, err
		}

		var course Course
		if err := DB.Where("id = ?", assignment.CourseID).First(&course).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify submission access",
			})
			return Submission{}, err
		}

		if course.ProfessorID != userID {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only access submissions from your own courses",
			})
			return Submission{}, errors.New("forbidden")
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid role",
		})
		return Submission{}, errors.New("invalid role")
	}

	return submission, nil
}

func submissionType(submission Submission) string {
	filePath := strings.TrimSpace(submission.FilePath)
	if filePath != "" {
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == ".pdf" {
			return "pdf"
		}
		return "image"
	}
	return "text"
}

func submissionPDFURL(submission Submission) string {
	if submissionType(submission) != "pdf" {
		return ""
	}
	return fmt.Sprintf("/submissions/%d/file", submission.ID)
}

func submissionFileURL(submission Submission) string {
	if strings.TrimSpace(submission.FilePath) == "" {
		return ""
	}
	return fmt.Sprintf("/submissions/%d/file", submission.ID)
}

func randomHexToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func saveSubmissionPDFFile(fileHeader *multipart.FileHeader, fileBytes []byte) (string, error) {
	if err := os.MkdirAll("uploads/submissions", 0o755); err != nil {
		return "", errors.New("could not prepare submissions uploads directory")
	}

	suffix, err := randomHexToken(6)
	if err != nil {
		return "", errors.New("could not generate submission file identifier")
	}

	filename := fmt.Sprintf("submission-%d-%s%s", time.Now().UnixNano(), suffix, strings.ToLower(filepath.Ext(fileHeader.Filename)))
	fullPath := filepath.Join("uploads", "submissions", filename)

	if err := os.WriteFile(fullPath, fileBytes, 0o644); err != nil {
		return "", errors.New("could not write submission PDF file")
	}

	return filepath.ToSlash(fullPath), nil
}

func saveSubmissionFile(fileHeader *multipart.FileHeader, fileBytes []byte) (string, error) {
	return saveSubmissionPDFFile(fileHeader, fileBytes)
}

func isSupportedSubmissionExtension(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf", ".png", ".jpg", ".jpeg", ".webp":
		return true
	default:
		return false
	}
}
