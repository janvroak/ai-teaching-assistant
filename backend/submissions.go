package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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
}

type AIEvaluateResponse struct {
	Marks      float64 `json:"marks"`
	Feedback   string  `json:"feedback"`
	Confidence float64 `json:"confidence"`
}

type AIEvaluateFileResult struct {
	Confidence float64 `json:"confidence"`
}

type AIEvaluateFileResponse struct {
	ExtractedText   string                 `json:"extracted_text"`
	Results         []AIEvaluateFileResult `json:"results"`
	TotalMarks      float64                `json:"total_marks"`
	OverallFeedback string                 `json:"overall_feedback"`
}

type AssignmentSubmissionResponse struct {
	SubmissionID uint   `json:"submission_id"`
	StudentID    uint   `json:"student_id"`
	Content      string `json:"content"`
	Evaluation   gin.H  `json:"evaluation"`
}

type MySubmissionResponse struct {
	SubmissionID    uint   `json:"submission_id"`
	AssignmentID    uint   `json:"assignment_id"`
	AssignmentTitle string `json:"assignment_title"`
	Content         string `json:"content"`
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

	submission := Submission{
		StudentID:    studentID,
		AssignmentID: req.AssignmentID,
		Content:      req.Content,
	}
	if err := DB.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create submission",
		})
		return
	}

	evalReq := AIEvaluateRequest{
		Question:        strings.TrimSpace(assignment.Question),
		StudentAnswer:   req.Content,
		ReferenceAnswer: assignment.AnswerKey,
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
		"http://localhost:8000/evaluate",
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

	evaluation := Evaluation{
		SubmissionID: submission.ID,
		Marks:        evalResp.Marks,
		Feedback:     evalResp.Feedback,
		Confidence:   evalResp.Confidence,
		IsFinal:      false,
	}
	if err := DB.Create(&evaluation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store evaluation",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"marks":      evaluation.Marks,
		"feedback":   evaluation.Feedback,
		"confidence": evaluation.Confidence,
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
	if strings.ToLower(filepath.Ext(fileHeader.Filename)) != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Only PDF files are supported",
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
	questionText := strings.TrimSpace(assignment.Question)
	if questionText == "" {
		questionText = strings.TrimSpace(assignment.Title)
	}
	if err := multipartWriter.WriteField("reference_answer", referenceAnswer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to prepare file upload payload",
		})
		return
	}
	if questionText != "" {
		if err := multipartWriter.WriteField("question", questionText); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to prepare file upload payload",
			})
			return
		}
	}
	if err := multipartWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to finalize file upload payload",
		})
		return
	}

	request, err := http.NewRequest(http.MethodPost, "http://localhost:8000/evaluate-file", &multipartBody)
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
		content = "Submitted via PDF file upload"
	}

	submission := Submission{
		StudentID:    studentID,
		AssignmentID: assignmentID,
		Content:      content,
	}
	if err := DB.Create(&submission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create submission",
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
		feedback = "Evaluation completed from uploaded PDF."
	}

	evaluation := Evaluation{
		SubmissionID: submission.ID,
		Marks:        evalResp.TotalMarks,
		Feedback:     feedback,
		Confidence:   confidence,
		IsFinal:      false,
	}
	if err := DB.Create(&evaluation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store evaluation",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"marks":    evaluation.Marks,
		"feedback": evaluation.Feedback,
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
		Distinct("id", "student_id", "content", "created_at").
		Order("id ASC").
		Find(&submissions).Error; err != nil {
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

	if len(assignmentIDs) > 0 {
		var assignments []Assignment
		if err := DB.Model(&Assignment{}).Where("id IN ?", assignmentIDs).Select("id", "title").Find(&assignments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch assignment data",
			})
			return
		}

		for _, assignment := range assignments {
			assignmentTitleByID[assignment.ID] = assignment.Title
		}
	}

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

	response := make([]AssignmentSubmissionResponse, 0, len(submissions))
	for _, submission := range submissions {
		evaluationData := gin.H{
			"id":         nil,
			"marks":      nil,
			"feedback":   nil,
			"confidence": nil,
		}
		if evaluation, exists := evaluationBySubmissionID[submission.ID]; exists {
			evaluationData["id"] = evaluation.ID
			evaluationData["marks"] = evaluation.Marks
			evaluationData["feedback"] = evaluation.Feedback
			evaluationData["confidence"] = evaluation.Confidence
		}

		response = append(response, AssignmentSubmissionResponse{
			SubmissionID: submission.ID,
			StudentID:    submission.StudentID,
			Content:      submission.Content,
			Evaluation:   evaluationData,
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

	c.JSON(http.StatusOK, gin.H{
		"submission": gin.H{
			"id":            submission.ID,
			"student_id":    submission.StudentID,
			"assignment_id": submission.AssignmentID,
			"content":       submission.Content,
			"created_at":    submission.CreatedAt,
		},
		"evaluation": gin.H{
			"marks":      evaluation.Marks,
			"feedback":   evaluation.Feedback,
			"confidence": evaluation.Confidence,
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

	if len(assignmentIDs) > 0 {
		var assignments []Assignment
		if err := DB.Model(&Assignment{}).Where("id IN ?", assignmentIDs).Select("id", "title").Find(&assignments).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch assignment data",
			})
			return
		}

		for _, assignment := range assignments {
			assignmentTitleByID[assignment.ID] = assignment.Title
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

	response := make([]MySubmissionResponse, 0, len(submissions))
	for _, submission := range submissions {
		evaluationData := gin.H{
			"marks":      nil,
			"feedback":   nil,
			"confidence": nil,
		}

		if evaluation, exists := evaluationBySubmissionID[submission.ID]; exists {
			evaluationData["marks"] = evaluation.Marks
			evaluationData["feedback"] = evaluation.Feedback
			evaluationData["confidence"] = evaluation.Confidence
		}

		response = append(response, MySubmissionResponse{
			SubmissionID:    submission.ID,
			AssignmentID:    submission.AssignmentID,
			AssignmentTitle: assignmentTitleByID[submission.AssignmentID],
			Content:         submission.Content,
			Evaluation:      evaluationData,
		})
	}

	c.JSON(http.StatusOK, response)
}
