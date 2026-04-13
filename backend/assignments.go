package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"encoding/hex"
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

type RubricCriterion struct {
	Criterion   string  `json:"criterion"`
	Description string  `json:"description"`
	MaxMarks    float64 `json:"max_marks"`
	Weight      float64 `json:"weight"`
}

type CreateAssignmentRequest struct {
	CourseID  uint   `json:"course_id"`
	UnitID    *uint  `json:"unit_id"`
	Title     string `json:"title"`
	Question  string `json:"question"`
	AnswerKey string `json:"answer_key"`
	Rubric    []RubricCriterion `json:"rubric"`
}

type ExtractPDFTextResponse struct {
	ExtractedText string `json:"extracted_text"`
}

type AssignmentListItem struct {
	ID              uint   `json:"id"`
	UnitID          *uint  `json:"unit_id,omitempty"`
	Title           string `json:"title"`
	Question        string `json:"question"`
	QuestionType    string `json:"question_type"`
	QuestionPDFURL  string `json:"question_pdf_url,omitempty"`
	AnswerKeyType   string `json:"answer_key_type,omitempty"`
	AnswerKeyPDFURL string `json:"answer_key_pdf_url,omitempty"`
	HasRubric       bool   `json:"has_rubric"`
	Rubric          []RubricCriterion `json:"rubric,omitempty"`
}

func CreateAssignmentHandler(c *gin.Context) {
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
			"error": "Only professors can create assignments",
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

	var req CreateAssignmentRequest
	questionFilePath := ""
	answerKeyFilePath := ""
	rubricJSONInput := ""
	contentType := strings.ToLower(strings.TrimSpace(c.ContentType()))

	if strings.HasPrefix(contentType, "multipart/form-data") {
		courseIDText := strings.TrimSpace(c.PostForm("course_id"))
		parsedCourseID, err := strconv.ParseUint(courseIDText, 10, 64)
		if err != nil || parsedCourseID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Valid course_id is required",
			})
			return
		}
		req.CourseID = uint(parsedCourseID)
		unitIDText := strings.TrimSpace(c.PostForm("unit_id"))
		if unitIDText != "" {
			parsedUnitID, parseErr := strconv.ParseUint(unitIDText, 10, 64)
			if parseErr != nil || parsedUnitID == 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "unit_id must be a positive integer",
				})
				return
			}
			parsedValue := uint(parsedUnitID)
			req.UnitID = &parsedValue
		}
		req.Title = strings.TrimSpace(c.PostForm("title"))
		rubricJSONInput = strings.TrimSpace(c.PostForm("rubric_json"))
		questionText := strings.TrimSpace(c.PostForm("question_text"))
		answerKeyText := strings.TrimSpace(c.PostForm("answer_key_text"))
		legacyAnswerKey := strings.TrimSpace(c.PostForm("answer_key"))
		if answerKeyText == "" && legacyAnswerKey != "" {
			answerKeyText = legacyAnswerKey
		}

		fileHeader, fileErr := c.FormFile("question_file")
		hasQuestionFile := fileErr == nil && fileHeader != nil
		hasQuestionText := questionText != ""
		answerKeyFileHeader, answerKeyFileErr := c.FormFile("answer_key_file")
		hasAnswerKeyFile := answerKeyFileErr == nil && answerKeyFileHeader != nil
		hasAnswerKeyText := answerKeyText != ""

		if hasQuestionText == hasQuestionFile {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Provide exactly one of question_text or question_file",
			})
			return
		}
		if hasAnswerKeyText == hasAnswerKeyFile {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Provide exactly one of answer_key_text or answer_key_file",
			})
			return
		}

		if hasQuestionFile {
			if strings.ToLower(filepath.Ext(fileHeader.Filename)) != ".pdf" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Only PDF files are supported for question_file",
				})
				return
			}

			savedPath, err := saveAssignmentPDFFile(fileHeader, "question")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Failed to save question PDF: %v", err),
				})
				return
			}
			questionFilePath = savedPath

			extractedQuestion, err := extractTextFromPDF(fileHeader)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{
					"error": fmt.Sprintf("Failed to process question PDF: %v", err),
				})
				return
			}

			req.Question = strings.TrimSpace(extractedQuestion)
		} else {
			req.Question = questionText
		}

		if hasAnswerKeyFile {
			if strings.ToLower(filepath.Ext(answerKeyFileHeader.Filename)) != ".pdf" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Only PDF files are supported for answer_key_file",
				})
				return
			}

			savedPath, err := saveAssignmentPDFFile(answerKeyFileHeader, "answer-key")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Failed to save answer key PDF: %v", err),
				})
				return
			}
			answerKeyFilePath = savedPath

			extractedAnswerKey, err := extractTextFromPDF(answerKeyFileHeader)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{
					"error": fmt.Sprintf("Failed to process answer key PDF: %v", err),
				})
				return
			}
			req.AnswerKey = strings.TrimSpace(extractedAnswerKey)
		} else {
			req.AnswerKey = answerKeyText
		}
	} else {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}
		req.Question = strings.TrimSpace(req.Question)
		if len(req.Rubric) > 0 {
			encoded, err := json.Marshal(req.Rubric)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "rubric must be valid JSON",
				})
				return
			}
			rubricJSONInput = string(encoded)
		}
	}

	req.Title = strings.TrimSpace(req.Title)
	req.AnswerKey = strings.TrimSpace(req.AnswerKey)
	if req.CourseID == 0 || req.Title == "" || req.Question == "" || req.AnswerKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "course_id, title, answer_key, and one question input are required",
		})
		return
	}
	if err := ensureAssignmentUnitBelongsToCourse(DB, req.CourseID, req.UnitID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "unit_id must belong to this course",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify unit",
		})
		return
	}

	var course Course
	err := DB.Where("id = ? AND professor_id = ?", req.CourseID, professorID).First(&course).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only create assignments for your own courses",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify course ownership",
		})
		return
	}
	rubricRows, normalizedRubricJSON, err := parseAndNormalizeRubric(rubricJSONInput)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	assignment := Assignment{
		CourseID:          req.CourseID,
		UnitID:            req.UnitID,
		Title:             req.Title,
		Question:          req.Question,
		QuestionFilePath:  questionFilePath,
		AnswerKey:         req.AnswerKey,
		RubricJSON:        normalizedRubricJSON,
		AnswerKeyFilePath: answerKeyFilePath,
	}

	if err := DB.Create(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create assignment",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":               assignment.ID,
		"title":            assignment.Title,
		"question":         assignment.Question,
		"course_id":        assignment.CourseID,
		"unit_id":          assignment.UnitID,
		"question_type":    questionTypeForAssignment(assignment),
		"question_pdf_url": questionPDFURLForAssignment(assignment),
		"answer_key_type":  answerKeyTypeForAssignment(assignment),
		"has_rubric":       len(rubricRows) > 0,
		"rubric":           rubricRows,
	})
}

func parseAndNormalizeRubric(raw string) ([]RubricCriterion, string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []RubricCriterion{}, "[]", nil
	}
	var rows []RubricCriterion
	if err := json.Unmarshal([]byte(trimmed), &rows); err != nil {
		return nil, "", errors.New("rubric_json must be a JSON array")
	}
	normalized := make([]RubricCriterion, 0, len(rows))
	for _, row := range rows {
		row.Criterion = strings.TrimSpace(row.Criterion)
		row.Description = strings.TrimSpace(row.Description)
		if row.Criterion == "" {
			return nil, "", errors.New("each rubric criterion must include criterion")
		}
		if row.MaxMarks <= 0 {
			return nil, "", errors.New("each rubric criterion must include max_marks > 0")
		}
		if row.Weight <= 0 {
			row.Weight = row.MaxMarks
		}
		normalized = append(normalized, row)
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, "", errors.New("failed to encode rubric")
	}
	return normalized, string(encoded), nil
}

func extractTextFromPDF(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", errors.New("could not open uploaded PDF file")
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", errors.New("could not read uploaded PDF file")
	}

	var multipartBody bytes.Buffer
	multipartWriter := multipart.NewWriter(&multipartBody)

	fileWriter, err := multipartWriter.CreateFormFile("file", fileHeader.Filename)
	if err != nil {
		return "", errors.New("failed to prepare question file payload")
	}
	if _, err := fileWriter.Write(fileBytes); err != nil {
		return "", errors.New("failed to prepare question file payload")
	}
	if err := multipartWriter.Close(); err != nil {
		return "", errors.New("failed to finalize question file payload")
	}

	request, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/extract-pdf-text", aiServiceBaseURL()),
		&multipartBody,
	)
	if err != nil {
		return "", errors.New("failed to prepare AI extraction request")
	}
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		return "", errors.New("could not connect to AI extraction service")
	}
	defer response.Body.Close()

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", errors.New("failed to read AI extraction response")
	}

	if response.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("AI extraction service returned status %d", response.StatusCode)
	}

	var extracted ExtractPDFTextResponse
	if err := json.Unmarshal(responseBytes, &extracted); err != nil {
		return "", errors.New("AI extraction response format is invalid")
	}

	extractedText := strings.TrimSpace(extracted.ExtractedText)
	if extractedText == "" {
		return "", errors.New("no text found in uploaded PDF")
	}

	return extractedText, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func saveAssignmentPDFFile(fileHeader *multipart.FileHeader, kind string) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", errors.New("could not open uploaded PDF file")
	}
	defer file.Close()

	if err := os.MkdirAll("uploads/assignments", 0o755); err != nil {
		return "", errors.New("could not prepare uploads directory")
	}

	suffix, err := randomHex(6)
	if err != nil {
		return "", errors.New("could not generate file identifier")
	}

	filename := fmt.Sprintf(
		"%s-%d-%s.pdf",
		kind,
		time.Now().UnixNano(),
		suffix,
	)
	fullPath := filepath.Join("uploads", "assignments", filename)

	out, err := os.Create(fullPath)
	if err != nil {
		return "", errors.New("could not create uploaded PDF file")
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return "", errors.New("could not save uploaded PDF file")
	}

	return filepath.ToSlash(fullPath), nil
}

func questionTypeForAssignment(assignment Assignment) string {
	if strings.TrimSpace(assignment.QuestionFilePath) != "" {
		return "pdf"
	}
	return "text"
}

func answerKeyTypeForAssignment(assignment Assignment) string {
	if strings.TrimSpace(assignment.AnswerKeyFilePath) != "" {
		return "pdf"
	}
	return "text"
}

func questionPDFURLForAssignment(assignment Assignment) string {
	if strings.TrimSpace(assignment.QuestionFilePath) == "" {
		return ""
	}
	return fmt.Sprintf("/assignments/%d/question-file", assignment.ID)
}

func answerKeyPDFURLForAssignment(assignment Assignment) string {
	if strings.TrimSpace(assignment.AnswerKeyFilePath) == "" {
		return ""
	}
	return fmt.Sprintf("/assignments/%d/answer-key-file", assignment.ID)
}

func GetAssignmentQuestionPDFHandler(c *gin.Context) {
	assignment, role, userID, err := getAccessibleAssignmentForFile(c)
	if err != nil {
		return
	}

	if role == "student" && strings.TrimSpace(assignment.QuestionFilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question PDF not found"})
		return
	}
	if role == "professor" && strings.TrimSpace(assignment.QuestionFilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Question PDF not found"})
		return
	}

	_ = userID
	c.File(assignment.QuestionFilePath)
}

func GetAssignmentAnswerKeyPDFHandler(c *gin.Context) {
	assignment, role, _, err := getAccessibleAssignmentForFile(c)
	if err != nil {
		return
	}

	if role != "professor" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only professors can access answer key PDF"})
		return
	}
	if strings.TrimSpace(assignment.AnswerKeyFilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Answer key PDF not found"})
		return
	}

	c.File(assignment.AnswerKeyFilePath)
}

func getAccessibleAssignmentForFile(c *gin.Context) (Assignment, string, uint, error) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return Assignment{}, "", 0, errors.New("db not initialized")
	}

	assignmentIDParam := c.Param("id")
	parsedAssignmentID, err := strconv.ParseUint(assignmentIDParam, 10, 64)
	if err != nil || parsedAssignmentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment id"})
		return Assignment{}, "", 0, errors.New("invalid assignment id")
	}

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token context"})
		return Assignment{}, "", 0, errors.New("invalid role context")
	}
	role = strings.ToLower(strings.TrimSpace(role))

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token context"})
		return Assignment{}, "", 0, errors.New("missing user id")
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token context"})
		return Assignment{}, "", 0, errors.New("invalid user id")
	}

	var assignment Assignment
	if err := DB.Where("id = ?", uint(parsedAssignmentID)).First(&assignment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignment"})
		}
		return Assignment{}, "", 0, err
	}

	switch role {
	case "professor":
		var course Course
		if err := DB.Where("id = ?", assignment.CourseID).First(&course).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assignment access"})
			return Assignment{}, "", 0, err
		}
		if course.ProfessorID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only access assignment files for your own courses"})
			return Assignment{}, "", 0, errors.New("forbidden")
		}
	case "student":
		var enrollment Enrollment
		if err := DB.Where("student_id = ? AND course_id = ?", userID, assignment.CourseID).First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, gin.H{"error": "You are not enrolled in this course"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify enrollment"})
			}
			return Assignment{}, "", 0, err
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid role"})
		return Assignment{}, "", 0, errors.New("invalid role")
	}

	return assignment, role, userID, nil
}

func ListCourseAssignmentsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	courseIDParam := c.Param("id")
	parsedCourseID, err := strconv.ParseUint(courseIDParam, 10, 64)
	if err != nil || parsedCourseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid course id",
		})
		return
	}
	courseID := uint(parsedCourseID)

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

	switch role {
	case "professor":
		var course Course
		err := DB.Where("id = ? AND professor_id = ?", courseID, userID).First(&course).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "You can only view assignments for your own courses",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify course ownership",
			})
			return
		}
	case "student":
		var enrollment Enrollment
		err := DB.Where("student_id = ? AND course_id = ?", userID, courseID).First(&enrollment).Error
		if err != nil {
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
	default:
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid role",
		})
		return
	}

	var assignments []Assignment
	if err := DB.
		Where("course_id = ?", courseID).
		Order("id ASC").
		Find(&assignments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch assignments",
		})
		return
	}

	response := make([]AssignmentListItem, 0, len(assignments))
	for _, assignment := range assignments {
		item := AssignmentListItem{
			ID:           assignment.ID,
			UnitID:       assignment.UnitID,
			Title:        assignment.Title,
			QuestionType: questionTypeForAssignment(assignment),
			HasRubric:    strings.TrimSpace(assignment.RubricJSON) != "" && strings.TrimSpace(assignment.RubricJSON) != "[]",
		}
		if assignment.QuestionFilePath != "" {
			item.QuestionPDFURL = questionPDFURLForAssignment(assignment)
		} else {
			item.Question = assignment.Question
		}

		if role == "professor" {
			item.AnswerKeyType = answerKeyTypeForAssignment(assignment)
			if assignment.AnswerKeyFilePath != "" {
				item.AnswerKeyPDFURL = answerKeyPDFURLForAssignment(assignment)
			}
			if item.HasRubric {
				var rubric []RubricCriterion
				if err := json.Unmarshal([]byte(assignment.RubricJSON), &rubric); err == nil {
					item.Rubric = rubric
				}
			}
		}

		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

func DeleteAssignmentHandler(c *gin.Context) {
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
			"error": "Only professors can delete assignments",
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
	if err := DB.Where("id = ? AND professor_id = ?", assignment.CourseID, professorID).First(&course).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only delete assignments in your own courses",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify assignment ownership",
		})
		return
	}

	filePaths := make([]string, 0, 16)
	if path := strings.TrimSpace(assignment.QuestionFilePath); path != "" {
		filePaths = append(filePaths, path)
	}
	if path := strings.TrimSpace(assignment.AnswerKeyFilePath); path != "" {
		filePaths = append(filePaths, path)
	}

	tx := DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start delete transaction",
		})
		return
	}

	rollbackWithError := func(message string) {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": message,
		})
	}

	var submissions []Submission
	if err := tx.Where("assignment_id = ?", assignmentID).Find(&submissions).Error; err != nil {
		rollbackWithError("Failed to fetch submissions")
		return
	}

	submissionIDs := make([]uint, 0, len(submissions))
	for _, submission := range submissions {
		submissionIDs = append(submissionIDs, submission.ID)
		if path := strings.TrimSpace(submission.FilePath); path != "" {
			filePaths = append(filePaths, path)
		}
	}

	if len(submissionIDs) > 0 {
		var evaluationIDs []uint
		if err := tx.Model(&Evaluation{}).Where("submission_id IN ?", submissionIDs).Pluck("id", &evaluationIDs).Error; err != nil {
			rollbackWithError("Failed to fetch evaluation ids")
			return
		}

		if len(evaluationIDs) > 0 {
			if err := tx.Where("evaluation_id IN ?", evaluationIDs).Delete(&EvaluationQuestion{}).Error; err != nil {
				rollbackWithError("Failed to delete evaluation questions")
				return
			}
			if err := tx.Where("evaluation_id IN ?", evaluationIDs).Delete(&EvaluationAuditLog{}).Error; err != nil {
				rollbackWithError("Failed to delete evaluation audit logs")
				return
			}
		}

		if err := tx.Where("submission_id IN ?", submissionIDs).Delete(&Evaluation{}).Error; err != nil {
			rollbackWithError("Failed to delete evaluations")
			return
		}
	}

	if err := tx.Where("assignment_id = ?", assignmentID).Delete(&PlagiarismReport{}).Error; err != nil {
		rollbackWithError("Failed to delete plagiarism reports")
		return
	}

	if err := tx.Where("assignment_id = ?", assignmentID).Delete(&Submission{}).Error; err != nil {
		rollbackWithError("Failed to delete submissions")
		return
	}

	if err := tx.Where("id = ?", assignmentID).Delete(&Assignment{}).Error; err != nil {
		rollbackWithError("Failed to delete assignment")
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to finalize assignment deletion",
		})
		return
	}

	for _, path := range filePaths {
		_ = os.Remove(path)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Assignment deleted successfully",
	})
}
