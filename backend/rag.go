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
	ID         uint      `json:"id"`
	Title      string    `json:"title"`
	HasFile    bool      `json:"has_file"`
	FileURL    string    `json:"file_url,omitempty"`
	CourseID   uint      `json:"course_id"`
	UploadedBy string    `json:"uploaded_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type CourseMaterialListItem struct {
	ID         uint      `json:"id"`
	Title      string    `json:"title"`
	HasFile    bool      `json:"has_file"`
	FileURL    string    `json:"file_url,omitempty"`
	UploadedBy string    `json:"uploaded_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AskDoubtRequest struct {
	Question string `json:"question"`
}

type RAGAnswerRequest struct {
	Question         string   `json:"question"`
	Contexts         []string `json:"contexts"`
	ProficiencyLevel string   `json:"proficiency_level,omitempty"`
	WeakTopics       []string `json:"weak_topics,omitempty"`
	RecentMistakes   []string `json:"recent_mistakes,omitempty"`
}

type RAGAnswerResponse struct {
	Answer     string                 `json:"answer"`
	Confidence float64                `json:"confidence"`
	Citations  []string               `json:"citations"`
	AdaptedFor string                 `json:"adapted_for,omitempty"`
	Profile    *StudentProfileSummary `json:"profile,omitempty"`
}

type StudentProfileSummary struct {
	ProficiencyLevel string   `json:"proficiency_level"`
	AvgScore         float64  `json:"avg_score"`
	TotalSubmissions int      `json:"total_submissions"`
	TotalAssignments int      `json:"total_assignments"`
	WeakTopics       []string `json:"weak_topics"`
	StrongTopics     []string `json:"strong_topics"`
	Progress         float64  `json:"progress"`
	ProgressLabel    string   `json:"progress_label"`
	Trend            string   `json:"trend"`
}

type ChatHistoryItem struct {
	ID        uint      `json:"id"`
	Question  string    `json:"question"`
	Answer    string    `json:"answer"`
	StudentID uint      `json:"student_id"`
	CourseID  uint      `json:"course_id"`
	Timestamp time.Time `json:"timestamp"`
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
		ID:         material.ID,
		Title:      material.Title,
		HasFile:    strings.TrimSpace(material.FilePath) != "",
		CourseID:   material.CourseID,
		UploadedBy: "Professor",
		CreatedAt:  material.CreatedAt,
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

	courseID, _, _, ok := requireCourseAccess(c)
	if !ok {
		return
	}

	var materials []CourseMaterial
	if err := DB.Where("course_id = ?", courseID).Order("id DESC").Find(&materials).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch materials"})
		return
	}

	uploadedBy := "Professor"
	var course Course
	if err := DB.Where("id = ?", courseID).First(&course).Error; err == nil {
		var professor User
		if err := DB.Where("id = ?", course.ProfessorID).First(&professor).Error; err == nil {
			name := strings.TrimSpace(professor.Name)
			if name != "" {
				uploadedBy = name
			}
		}
	}

	response := make([]CourseMaterialListItem, 0, len(materials))
	for _, material := range materials {
		item := CourseMaterialListItem{
			ID:         material.ID,
			Title:      material.Title,
			HasFile:    strings.TrimSpace(material.FilePath) != "",
			UploadedBy: uploadedBy,
			CreatedAt:  material.CreatedAt,
		}
		if item.HasFile {
			item.FileURL = fmt.Sprintf("/materials/%d/file", material.ID)
		}
		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

func AskCourseDoubtHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, userID, role, ok := requireCourseAccess(c)
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

	proficiencyLevel := "intermediate"
	avgScore := 0.0
	totalSubmissions := 0
	weakTopics := []string{}
	strongTopics := []string{}
	recentMistakes := []string{}
	if role == "student" {
		profile, err := getOrBuildAdaptiveProfile(userID, courseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build student profile"})
			return
		}
		proficiencyLevel = profile.ProficiencyLevel
		avgScore = profile.AvgScore
		totalSubmissions = profile.TotalSubmissions
		weakTopics = parseTopicsJSON(profile.WeakTopics)
		strongTopics = parseTopicsJSON(profile.StrongTopics)

		mistakes, err := listRecentMistakes(userID, courseID, 5)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load recent mistakes"})
			return
		}
		recentMistakes = mistakes
	}

	trend := "improving"
	progress := 0.0
	progressLabelValue := "Beginner"
	totalAssignments := 0
	if role == "student" {
		value, err := learningTrend(userID, courseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate learning trend"})
			return
		}
		trend = value

		completion, completionLabel, assignmentCount, err := courseProgressForStudent(userID, courseID, totalSubmissions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate course progress"})
			return
		}
		progress = completion
		progressLabelValue = completionLabel
		totalAssignments = assignmentCount
	}

	answer, err := askAIWithContext(question, contexts, proficiencyLevel, weakTopics, recentMistakes)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	answer.Citations = citations
	answer.AdaptedFor = proficiencyLevel
	if role == "student" {
		answer.Profile = &StudentProfileSummary{
			ProficiencyLevel: proficiencyLevel,
			AvgScore:         avgScore,
			TotalSubmissions: totalSubmissions,
			TotalAssignments: totalAssignments,
			WeakTopics:       weakTopics,
			StrongTopics:     strongTopics,
			Progress:         progress,
			ProgressLabel:    progressLabelValue,
			Trend:            trend,
		}
		if err := saveStudentInteraction(userID, courseID, question, answer.Answer, proficiencyLevel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record student interaction"})
			return
		}
	}
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

func askAIWithContext(question string, contexts []string, proficiencyLevel string, weakTopics []string, recentMistakes []string) (RAGAnswerResponse, error) {
	payload := RAGAnswerRequest{
		Question:         question,
		Contexts:         contexts,
		ProficiencyLevel: proficiencyLevel,
		WeakTopics:       weakTopics,
		RecentMistakes:   recentMistakes,
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

func GetStudentProfileHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, userID, role, ok := requireCourseAccess(c)
	if !ok {
		return
	}
	if role != "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only students can access adaptive profile"})
		return
	}

	profile, err := recomputeAdaptiveProfile(DB, userID, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch profile"})
		return
	}
	trend, err := learningTrend(userID, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate trend"})
		return
	}
	progress, label, totalAssignments, err := courseProgressForStudent(userID, courseID, profile.TotalSubmissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate course progress"})
		return
	}

	c.JSON(http.StatusOK, StudentProfileSummary{
		ProficiencyLevel: profile.ProficiencyLevel,
		AvgScore:         profile.AvgScore,
		TotalSubmissions: profile.TotalSubmissions,
		TotalAssignments: totalAssignments,
		WeakTopics:       parseTopicsJSON(profile.WeakTopics),
		StrongTopics:     parseTopicsJSON(profile.StrongTopics),
		Progress:         progress,
		ProgressLabel:    label,
		Trend:            trend,
	})
}

func GetCourseChatHistoryHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, userID, role, ok := requireCourseAccess(c)
	if !ok {
		return
	}
	if role != "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only students can access chat history"})
		return
	}

	var interactions []StudentInteraction
	if err := DB.Where("student_id = ? AND course_id = ? AND interaction_type = ?", userID, courseID, "doubt").
		Order("id ASC").
		Find(&interactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat history"})
		return
	}

	items := make([]ChatHistoryItem, 0, len(interactions))
	for _, interaction := range interactions {
		items = append(items, ChatHistoryItem{
			ID:        interaction.ID,
			Question:  interaction.Question,
			Answer:    interaction.Response,
			StudentID: interaction.StudentID,
			CourseID:  interaction.CourseID,
			Timestamp: interaction.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, items)
}

func listRecentInteractionQuestions(studentID uint, courseID uint, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}
	var interactions []StudentInteraction
	if err := DB.Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("id DESC").
		Limit(limit).
		Find(&interactions).Error; err != nil {
		return nil, err
	}

	history := make([]string, 0, len(interactions))
	for _, item := range interactions {
		question := strings.TrimSpace(item.Question)
		if question == "" {
			continue
		}
		history = append(history, question)
	}
	return history, nil
}

func saveStudentInteraction(studentID uint, courseID uint, question string, response string, proficiencyLevel string) error {
	interaction := StudentInteraction{
		StudentID:        studentID,
		CourseID:         courseID,
		InteractionType:  "doubt",
		Question:         strings.TrimSpace(question),
		Response:         strings.TrimSpace(response),
		ProficiencyLevel: strings.TrimSpace(proficiencyLevel),
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&interaction).Error; err != nil {
			return err
		}
		for _, topic := range extractTopicKeywords(question, 5) {
			if err := incrementChatTopicStat(tx, studentID, courseID, topic); err != nil {
				return err
			}
		}
		return nil
	})
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
