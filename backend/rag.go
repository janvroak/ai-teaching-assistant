package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateCourseMaterialResponse struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	HasFile  bool   `json:"has_file"`
	FileURL  string `json:"file_url,omitempty"`
	CourseID uint   `json:"course_id"`
}

type CourseMaterialListItem struct {
	ID      uint   `json:"id"`
	Title   string `json:"title"`
	HasFile bool   `json:"has_file"`
	FileURL string `json:"file_url,omitempty"`
}

type AskDoubtRequest struct {
	Question string `json:"question"`
}

type RAGAnswerRequest struct {
	Question string   `json:"question"`
	Contexts []string `json:"contexts"`
}

type RAGAnswerResponse struct {
	Answer     string   `json:"answer"`
	Confidence float64  `json:"confidence"`
	Citations  []string `json:"citations"`
}

func CreateCourseMaterialHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, professorID, ok := requireProfessorOwnsCourse(c)
	if !ok {
		return
	}
	_ = professorID

	title := strings.TrimSpace(c.PostForm("title"))
	contentText := strings.TrimSpace(c.PostForm("content"))

	fileHeader, fileErr := c.FormFile("file")
	hasFile := fileErr == nil && fileHeader != nil
	hasText := contentText != ""

	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if hasFile == hasText {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provide exactly one of content or file"})
		return
	}

	filePath := ""
	content := ""

	if hasFile {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		switch ext {
		case ".pdf":
			savedPath, err := saveCourseMaterialFile(fileHeader)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to store material file: %v", err)})
				return
			}
			filePath = savedPath

			extractedText, err := extractTextFromPDF(fileHeader)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Failed to process material PDF: %v", err)})
				return
			}
			content = strings.TrimSpace(extractedText)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only PDF files are currently supported for materials"})
			return
		}
	} else {
		content = contentText
	}

	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Material content cannot be empty"})
		return
	}

	material := CourseMaterial{
		CourseID: courseID,
		Title:    title,
		Content:  content,
		FilePath: filePath,
	}
	if err := DB.Create(&material).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create course material"})
		return
	}

	resp := CreateCourseMaterialResponse{
		ID:       material.ID,
		Title:    material.Title,
		HasFile:  strings.TrimSpace(material.FilePath) != "",
		CourseID: material.CourseID,
	}
	if resp.HasFile {
		resp.FileURL = fmt.Sprintf("/materials/%d/file", material.ID)
	}

	c.JSON(http.StatusCreated, resp)
}

func ListCourseMaterialsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, _, role, ok := requireCourseAccess(c)
	if !ok {
		return
	}

	var materials []CourseMaterial
	if err := DB.Where("course_id = ?", courseID).Order("id DESC").Find(&materials).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch materials"})
		return
	}

	response := make([]CourseMaterialListItem, 0, len(materials))
	for _, material := range materials {
		item := CourseMaterialListItem{
			ID:      material.ID,
			Title:   material.Title,
			HasFile: strings.TrimSpace(material.FilePath) != "",
		}
		if item.HasFile {
			item.FileURL = fmt.Sprintf("/materials/%d/file", material.ID)
		}
		response = append(response, item)
	}

	_ = role
	c.JSON(http.StatusOK, response)
}

func AskCourseDoubtHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, _, _, ok := requireCourseAccess(c)
	if !ok {
		return
	}

	var req AskDoubtRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	question := strings.TrimSpace(req.Question)
	if question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
		return
	}

	var materials []CourseMaterial
	if err := DB.Where("course_id = ?", courseID).Order("id DESC").Find(&materials).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch course materials"})
		return
	}
	if len(materials) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No course materials found. Ask the professor to upload materials first."})
		return
	}

	type scoredChunk struct {
		Score         int
		Chunk         string
		MaterialTitle string
	}

	questionTokens := tokenize(question)
	scored := make([]scoredChunk, 0)
	for _, material := range materials {
		chunks := splitIntoChunks(material.Content, 1200, 200)
		if len(chunks) == 0 {
			continue
		}
		for _, chunk := range chunks {
			score := tokenOverlapScore(questionTokens, tokenize(chunk))
			if score == 0 {
				continue
			}
			scored = append(scored, scoredChunk{
				Score:         score,
				Chunk:         chunk,
				MaterialTitle: material.Title,
			})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Strictly context-grounded mode:
	// if no chunk has lexical overlap with the student question,
	// do not ask the model to avoid out-of-context hallucinations.
	if len(scored) == 0 || scored[0].Score <= 0 {
		c.JSON(http.StatusOK, RAGAnswerResponse{
			Answer:     "I could not find this topic in the uploaded course materials. Please ask a question from course content or ask your professor to upload material for this topic.",
			Confidence: 0.2,
			Citations:  []string{},
		})
		return
	}

	topN := 6
	if len(scored) < topN {
		topN = len(scored)
	}
	contexts := make([]string, 0, topN)
	citations := make([]string, 0, topN)
	seenTitles := map[string]bool{}
	for i := 0; i < topN; i++ {
		contexts = append(contexts, scored[i].Chunk)
		if !seenTitles[scored[i].MaterialTitle] {
			citations = append(citations, scored[i].MaterialTitle)
			seenTitles[scored[i].MaterialTitle] = true
		}
	}

	answer, err := askAIWithContext(question, contexts)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	answer.Citations = citations
	c.JSON(http.StatusOK, answer)
}

func GetCourseMaterialFileHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	materialIDParam := c.Param("id")
	parsedMaterialID, err := strconv.ParseUint(materialIDParam, 10, 64)
	if err != nil || parsedMaterialID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid material id"})
		return
	}

	var material CourseMaterial
	if err := DB.Where("id = ?", uint(parsedMaterialID)).First(&material).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Material not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch material"})
		}
		return
	}

	_, _, _, ok := requireCourseAccessByID(c, material.CourseID)
	if !ok {
		return
	}

	if strings.TrimSpace(material.FilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Material file not found"})
		return
	}

	c.File(material.FilePath)
}

func askAIWithContext(question string, contexts []string) (RAGAnswerResponse, error) {
	payload := RAGAnswerRequest{
		Question: question,
		Contexts: contexts,
	}
	requestBytes, err := json.Marshal(payload)
	if err != nil {
		return RAGAnswerResponse{}, errors.New("failed to prepare AI RAG payload")
	}

	response, err := http.Post(
		fmt.Sprintf("%s/answer-with-context", aiServiceBaseURL()),
		"application/json",
		bytes.NewBuffer(requestBytes),
	)
	if err != nil {
		return RAGAnswerResponse{}, errors.New("could not connect to AI RAG service")
	}
	defer response.Body.Close()

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return RAGAnswerResponse{}, errors.New("failed to read AI RAG response")
	}

	if response.StatusCode >= http.StatusBadRequest {
		return RAGAnswerResponse{}, fmt.Errorf("AI RAG service returned status %d", response.StatusCode)
	}

	var ragResponse RAGAnswerResponse
	if err := json.Unmarshal(responseBytes, &ragResponse); err != nil {
		return RAGAnswerResponse{}, errors.New("AI RAG response format is invalid")
	}
	return ragResponse, nil
}

func tokenize(text string) map[string]struct{} {
	tokens := map[string]struct{}{}
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, text)
	for _, token := range strings.Fields(cleaned) {
		if len(token) < 3 {
			continue
		}
		tokens[token] = struct{}{}
	}
	return tokens
}

func tokenOverlapScore(a, b map[string]struct{}) int {
	score := 0
	for token := range a {
		if _, ok := b[token]; ok {
			score++
		}
	}
	return score
}

func splitIntoChunks(text string, chunkSize, overlap int) []string {
	content := strings.TrimSpace(text)
	if content == "" {
		return nil
	}
	if len(content) <= chunkSize {
		return []string{content}
	}

	chunks := make([]string, 0)
	start := 0
	for start < len(content) {
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}
		chunk := strings.TrimSpace(content[start:end])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(content) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

func saveCourseMaterialFile(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", errors.New("could not open uploaded material file")
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", errors.New("could not read uploaded material file")
	}

	if err := os.MkdirAll("uploads/materials", 0o755); err != nil {
		return "", errors.New("could not create materials upload directory")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	filename := fmt.Sprintf("material-%d%s", parsedNowUnixNano(), ext)
	fullPath := filepath.Join("uploads", "materials", filename)

	if err := os.WriteFile(fullPath, fileBytes, 0o644); err != nil {
		return "", errors.New("could not save uploaded material file")
	}
	return filepath.ToSlash(fullPath), nil
}

func parsedNowUnixNano() int64 {
	return time.Now().UnixNano()
}

func requireProfessorOwnsCourse(c *gin.Context) (uint, uint, bool) {
	courseIDParam := c.Param("id")
	parsedCourseID, err := strconv.ParseUint(courseIDParam, 10, 64)
	if err != nil || parsedCourseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course id"})
		return 0, 0, false
	}
	courseID := uint(parsedCourseID)

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "professor" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only professors can perform this action"})
		return 0, 0, false
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token context"})
		return 0, 0, false
	}
	professorID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token context"})
		return 0, 0, false
	}

	var course Course
	if err := DB.Where("id = ? AND professor_id = ?", courseID, professorID).First(&course).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only manage materials for your own courses"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify course ownership"})
		}
		return 0, 0, false
	}

	return courseID, professorID, true
}

func requireCourseAccess(c *gin.Context) (uint, uint, string, bool) {
	courseIDParam := c.Param("id")
	parsedCourseID, err := strconv.ParseUint(courseIDParam, 10, 64)
	if err != nil || parsedCourseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course id"})
		return 0, 0, "", false
	}
	return requireCourseAccessByID(c, uint(parsedCourseID))
}

func requireCourseAccessByID(c *gin.Context, courseID uint) (uint, uint, string, bool) {
	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token context"})
		return 0, 0, "", false
	}
	role = strings.ToLower(strings.TrimSpace(role))

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token context"})
		return 0, 0, "", false
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token context"})
		return 0, 0, "", false
	}

	switch role {
	case "professor":
		var course Course
		if err := DB.Where("id = ? AND professor_id = ?", courseID, userID).First(&course).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, gin.H{"error": "You can only access your own courses"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify course ownership"})
			}
			return 0, 0, "", false
		}
	case "student":
		var enrollment Enrollment
		if err := DB.Where("student_id = ? AND course_id = ?", userID, courseID).First(&enrollment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, gin.H{"error": "You are not enrolled in this course"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify enrollment"})
			}
			return 0, 0, "", false
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid role"})
		return 0, 0, "", false
	}

	return courseID, userID, role, true
}
