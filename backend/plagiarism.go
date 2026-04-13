package main

import (
	"math"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type assignmentPlagiarismRow struct {
	ReportID               uint     `json:"report_id"`
	SubmissionID           uint     `json:"submission_id"`
	StudentID              uint     `json:"student_id"`
	StudentName            string   `json:"student_name"`
	ComparedSubmissionID   *uint    `json:"compared_submission_id,omitempty"`
	ComparedStudentID      *uint    `json:"compared_student_id,omitempty"`
	ComparedStudentName    string   `json:"compared_student_name,omitempty"`
	SemanticSimilarity     float64  `json:"semantic_similarity"`
	StyleAnomalyScore      float64  `json:"style_anomaly_score"`
	AIUsageLikelihood      float64  `json:"ai_usage_likelihood"`
	PlagiarismLikelihood   float64  `json:"plagiarism_likelihood"`
	InterpretationBand     string   `json:"interpretation_band"`
	Flagged                bool     `json:"flagged"`
	ReasonSummary          string   `json:"reason_summary"`
	Reasons                []string `json:"reasons"`
}

func interpretationBand(score float64) string {
	switch {
	case score >= 0.9:
		return "0.9-1.0: Highly plagiarized or AI-generated"
	case score >= 0.7:
		return "0.7-0.9: Suspicious similarity"
	default:
		return "Below 0.7: Acceptable variation"
	}
}

func clampZeroOne(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func tokenizeForSimilarity(text string) map[string]float64 {
	tokens := make(map[string]float64)
	re := regexp.MustCompile(`[a-z0-9]+`)
	for _, token := range re.FindAllString(strings.ToLower(text), -1) {
		if len(token) < 3 {
			continue
		}
		tokens[token]++
	}
	return tokens
}

func cosineTokenSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	var dot float64
	var normA float64
	var normB float64

	for token, av := range a {
		bv := b[token]
		dot += av * bv
		normA += av * av
	}
	for _, bv := range b {
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return clampZeroOne(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func textStats(content string) (sentenceCount float64, avgWordsPerSentence float64, avgWordLength float64) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return 0, 0, 0
	}
	splitSentences := regexp.MustCompile(`[.!?]+`).Split(trimmed, -1)
	words := regexp.MustCompile(`[a-zA-Z0-9]+`).FindAllString(trimmed, -1)
	wordCount := float64(len(words))
	sentenceCounter := 0
	for _, sentence := range splitSentences {
		if strings.TrimSpace(sentence) != "" {
			sentenceCounter++
		}
	}
	if sentenceCounter == 0 {
		sentenceCounter = 1
	}
	var chars float64
	for _, word := range words {
		chars += float64(len(word))
	}
	avgWord := 0.0
	if wordCount > 0 {
		avgWord = chars / wordCount
	}
	return float64(sentenceCounter), wordCount / float64(sentenceCounter), avgWord
}

func styleAnomalyScore(current string, history []Submission) float64 {
	if len(history) == 0 {
		return 0
	}

	_, currAvgWords, currAvgWordLen := textStats(current)
	if currAvgWords == 0 {
		return 0
	}

	avgWordsList := make([]float64, 0, len(history))
	avgWordLenList := make([]float64, 0, len(history))
	for _, submission := range history {
		_, avgWords, avgWordLen := textStats(submission.Content)
		if avgWords <= 0 {
			continue
		}
		avgWordsList = append(avgWordsList, avgWords)
		avgWordLenList = append(avgWordLenList, avgWordLen)
	}
	if len(avgWordsList) == 0 {
		return 0
	}

	meanWords := 0.0
	meanWordLen := 0.0
	for _, v := range avgWordsList {
		meanWords += v
	}
	for _, v := range avgWordLenList {
		meanWordLen += v
	}
	meanWords /= float64(len(avgWordsList))
	meanWordLen /= float64(len(avgWordLenList))

	wordDelta := math.Abs(currAvgWords-meanWords) / math.Max(meanWords, 1)
	wordLenDelta := math.Abs(currAvgWordLen-meanWordLen) / math.Max(meanWordLen, 1)
	return clampZeroOne((0.7 * wordDelta) + (0.3 * wordLenDelta))
}

func aiUsageLikelihood(content string) float64 {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return 0
	}

	_, avgWords, avgWordLen := textStats(trimmed)
	tokenCount := len(tokenizeForSimilarity(trimmed))
	formalConnectors := 0
	for _, phrase := range []string{
		"therefore", "moreover", "furthermore", "in conclusion", "consequently",
		"additionally", "in contrast", "however", "notwithstanding",
	} {
		if strings.Contains(strings.ToLower(trimmed), phrase) {
			formalConnectors++
		}
	}

	lengthSignal := clampZeroOne((float64(tokenCount) - 120) / 200)
	formalitySignal := clampZeroOne(float64(formalConnectors) / 5)
	verbositySignal := clampZeroOne((avgWords - 18) / 18)
	wordLenSignal := clampZeroOne((avgWordLen - 4.8) / 2.2)

	return clampZeroOne((0.35 * lengthSignal) + (0.25 * formalitySignal) + (0.25 * verbositySignal) + (0.15 * wordLenSignal))
}

func evaluateSubmissionIntegrity(tx *gorm.DB, submission Submission, assignment Assignment) error {
	currentTokens := tokenizeForSimilarity(submission.Content)

	var peerSubmissions []Submission
	if err := tx.Where("assignment_id = ? AND id <> ?", submission.AssignmentID, submission.ID).Find(&peerSubmissions).Error; err != nil {
		return err
	}

	bestSimilarity := 0.0
	var bestPeerID *uint
	for _, peer := range peerSubmissions {
		sim := cosineTokenSimilarity(currentTokens, tokenizeForSimilarity(peer.Content))
		if sim > bestSimilarity {
			bestSimilarity = sim
			peerID := peer.ID
			bestPeerID = &peerID
		}
	}

	var history []Submission
	if err := tx.Table("submissions AS s").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("s.student_id = ? AND a.course_id = ? AND s.id <> ?", submission.StudentID, assignment.CourseID, submission.ID).
		Order("s.created_at DESC, s.id DESC").
		Limit(10).
		Find(&history).Error; err != nil {
		return err
	}

	styleScore := styleAnomalyScore(submission.Content, history)
	aiLikelihood := aiUsageLikelihood(submission.Content)
	plagiarismLikelihood := clampZeroOne((0.7 * bestSimilarity) + (0.2 * styleScore) + (0.1 * aiLikelihood))

	reasons := make([]string, 0, 4)
	if bestSimilarity >= 0.85 {
		reasons = append(reasons, "Very high semantic overlap with another submission in the same assignment.")
	} else if bestSimilarity >= 0.7 {
		reasons = append(reasons, "High semantic overlap with another submission in the same assignment.")
	}
	if styleScore >= 0.6 {
		reasons = append(reasons, "Writing style differs significantly from the student's historical submissions.")
	}
	if aiLikelihood >= 0.75 {
		reasons = append(reasons, "Text exhibits high AI-assisted writing likelihood.")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "No strong integrity anomaly detected.")
	}
	reasons = append(reasons, interpretationBand(plagiarismLikelihood))

	flagged := plagiarismLikelihood >= 0.7 || (aiLikelihood >= 0.9 && styleScore >= 0.6)

	report := PlagiarismReport{
		SubmissionID:         submission.ID,
		AssignmentID:         submission.AssignmentID,
		CourseID:             assignment.CourseID,
		StudentID:            submission.StudentID,
		ComparedSubmissionID: bestPeerID,
		SemanticSimilarity:   bestSimilarity,
		StyleAnomalyScore:    styleScore,
		AIUsageLikelihood:    aiLikelihood,
		PlagiarismLikelihood: plagiarismLikelihood,
		Flagged:              flagged,
		ReasonSummary:        reasons[0],
		Reasons:              reasons,
	}
	return tx.Create(&report).Error
}

func ListAssignmentPlagiarismFlagsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "professor" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only professors can access plagiarism flags"})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token context"})
		return
	}
	professorID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token context"})
		return
	}

	assignmentIDParam := c.Param("id")
	parsedAssignmentID, err := strconv.ParseUint(assignmentIDParam, 10, 64)
	if err != nil || parsedAssignmentID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignment id"})
		return
	}
	assignmentID := uint(parsedAssignmentID)

	var assignment Assignment
	if err := DB.Where("id = ?", assignmentID).First(&assignment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
		return
	}

	var course Course
	if err := DB.Where("id = ?", assignment.CourseID).First(&course).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify assignment ownership"})
		return
	}
	if course.ProfessorID != professorID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only access your own course assignments"})
		return
	}

	var reports []PlagiarismReport
	if err := DB.Where("assignment_id = ?", assignmentID).Order("plagiarism_likelihood DESC, id DESC").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch plagiarism reports"})
		return
	}

	userIDs := make([]uint, 0, len(reports)*2)
	comparedSubmissionIDs := make([]uint, 0, len(reports))
	for _, report := range reports {
		userIDs = append(userIDs, report.StudentID)
		if report.ComparedSubmissionID != nil {
			comparedSubmissionIDs = append(comparedSubmissionIDs, *report.ComparedSubmissionID)
		}
	}

	submissionToStudent := map[uint]uint{}
	if len(comparedSubmissionIDs) > 0 {
		var submissions []Submission
		if err := DB.Where("id IN ?", comparedSubmissionIDs).Find(&submissions).Error; err == nil {
			for _, row := range submissions {
				submissionToStudent[row.ID] = row.StudentID
				userIDs = append(userIDs, row.StudentID)
			}
		}
	}

	userMap := map[uint]string{}
	if len(userIDs) > 0 {
		var users []User
		if err := DB.Where("id IN ?", userIDs).Find(&users).Error; err == nil {
			for _, user := range users {
				userMap[user.ID] = strings.TrimSpace(user.Name)
			}
		}
	}

	rows := make([]assignmentPlagiarismRow, 0, len(reports))
	for _, report := range reports {
		var comparedStudentID *uint
		var comparedStudentName string
		if report.ComparedSubmissionID != nil {
			if sid, ok := submissionToStudent[*report.ComparedSubmissionID]; ok {
				comparedStudentID = &sid
				comparedStudentName = userMap[sid]
			}
		}

		rows = append(rows, assignmentPlagiarismRow{
			ReportID:             report.ID,
			SubmissionID:         report.SubmissionID,
			StudentID:            report.StudentID,
			StudentName:          userMap[report.StudentID],
			ComparedSubmissionID: report.ComparedSubmissionID,
			ComparedStudentID:    comparedStudentID,
			ComparedStudentName:  comparedStudentName,
			SemanticSimilarity:   report.SemanticSimilarity,
			StyleAnomalyScore:    report.StyleAnomalyScore,
			AIUsageLikelihood:    report.AIUsageLikelihood,
			PlagiarismLikelihood: report.PlagiarismLikelihood,
			InterpretationBand:   interpretationBand(report.PlagiarismLikelihood),
			Flagged:              report.Flagged,
			ReasonSummary:        report.ReasonSummary,
			Reasons:              report.Reasons,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Flagged == rows[j].Flagged {
			return rows[i].PlagiarismLikelihood > rows[j].PlagiarismLikelihood
		}
		return rows[i].Flagged && !rows[j].Flagged
	})

	c.JSON(http.StatusOK, gin.H{
		"assignment_id": assignmentID,
		"reports":       rows,
	})
}
