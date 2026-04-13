package main

import (
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

type StudentAnalytics struct {
	UserID           uint     `json:"user_id"`
	Name             string   `json:"name"`
	CourseID         uint     `json:"course_id"`
	AvgScore         float64  `json:"avg_score"`
	TotalSubmissions int      `json:"total_submissions"`
	Progress         float64  `json:"progress"`
	ProgressStatus   string   `json:"progress_status"`
	ProgressEstimated bool    `json:"progress_estimated"`
	ProgressNote     string   `json:"progress_note,omitempty"`
	Trend            string   `json:"trend"`
	Rank             int      `json:"rank"`
	Percentile       float64  `json:"percentile"`
	StrongTopics     []string `json:"strong_topics"`
	WeakTopics       []string `json:"weak_topics"`
}

type AtRiskStudent struct {
	UserID         uint    `json:"user_id"`
	Name           string  `json:"name"`
	AvgScore       float64 `json:"avg_score"`
	Progress       float64 `json:"progress"`
	ProgressStatus string  `json:"progress_status"`
	Trend          string  `json:"trend"`
}

type CourseAnalytics struct {
	CourseID                uint                 `json:"course_id"`
	AvgScore                float64              `json:"avg_score"`
	AverageProgress         float64              `json:"average_progress"`
	TotalStudents           int                  `json:"total_students"`
	TopPerformer            *TopPerformerSummary `json:"top_performer,omitempty"`
	WeakTopics              []string             `json:"weak_topics"`
	StrongTopics            []string             `json:"strong_topics"`
	PerformanceDistribution map[string]int       `json:"performance_distribution"`
	ProgressDistribution    map[string]int       `json:"progress_distribution"`
	AtRiskStudents          []AtRiskStudent      `json:"at_risk_students"`
}

type TopPerformerSummary struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type studentAggregateRow struct {
	AvgScore         float64
	TotalSubmissions int64
}

type studentRankResponse struct {
	StudentID     uint    `json:"student_id"`
	Rank          int     `json:"rank"`
	TotalStudents int     `json:"total_students"`
	Percentile    float64 `json:"percentile"`
	TopPercentile float64 `json:"top_percentile"`
	AvgScore      float64 `json:"avg_score"`
	CourseID      uint    `json:"course_id"`
}

func computeCourseAnalyticsSnapshot(courseID uint) ([]StudentAnalytics, CourseAnalytics, error) {
	var enrollments []Enrollment
	if err := DB.Where("course_id = ?", courseID).Find(&enrollments).Error; err != nil {
		return nil, CourseAnalytics{}, err
	}

	totalStudents := len(enrollments)
	studentAnalytics := make([]StudentAnalytics, 0, totalStudents)
	studentIDs := make([]uint, 0, totalStudents)
	for _, enrollment := range enrollments {
		studentIDs = append(studentIDs, enrollment.StudentID)
	}

	var users []User
	if len(studentIDs) > 0 {
		if err := DB.Where("id IN ?", studentIDs).Find(&users).Error; err != nil {
			return nil, CourseAnalytics{}, err
		}
	}

	userMap := map[uint]string{}
	for _, u := range users {
		userMap[u.ID] = u.Name
	}

	for _, enrollment := range enrollments {
		studentID := enrollment.StudentID

		var aggregate studentAggregateRow
		if err := DB.Table("submissions AS s").
			Select("COALESCE(AVG(e.marks), 0) AS avg_score, COUNT(DISTINCT s.id) AS total_submissions").
			Joins("LEFT JOIN evaluations AS e ON e.submission_id = s.id").
			Joins("JOIN assignments AS a ON a.id = s.assignment_id").
			Where("s.student_id = ? AND a.course_id = ?", studentID, courseID).
			Scan(&aggregate).Error; err != nil {
			return nil, CourseAnalytics{}, err
		}
		if aggregate.TotalSubmissions == 0 {
			aggregate.AvgScore = 0
		}

		profile, err := recomputeAdaptiveProfile(DB, studentID, courseID)
		if err != nil {
			return nil, CourseAnalytics{}, err
		}

		weakTopics := parseTopicsJSON(profile.WeakTopics)
		strongTopics := parseTopicsJSON(profile.StrongTopics)
		progress, status, _, estimated, note, err := courseProgressForStudent(studentID, courseID, int(aggregate.TotalSubmissions))
		if err != nil {
			return nil, CourseAnalytics{}, err
		}
		trend, err := learningTrend(studentID, courseID)
		if err != nil {
			return nil, CourseAnalytics{}, err
		}

		studentAnalytics = append(studentAnalytics, StudentAnalytics{
			UserID:           studentID,
			Name:             userMap[studentID],
			CourseID:         courseID,
			AvgScore:         aggregate.AvgScore,
			TotalSubmissions: int(aggregate.TotalSubmissions),
			Progress:         progress,
			ProgressStatus:   status,
			ProgressEstimated: estimated,
			ProgressNote:     note,
			Trend:            trend,
			StrongTopics:     strongTopics,
			WeakTopics:       weakTopics,
		})
	}

	// Ranking logic: sort by average score descending and assign tie-aware ranks.
	sort.Slice(studentAnalytics, func(i, j int) bool {
		if studentAnalytics[i].AvgScore == studentAnalytics[j].AvgScore {
			return studentAnalytics[i].UserID < studentAnalytics[j].UserID
		}
		return studentAnalytics[i].AvgScore > studentAnalytics[j].AvgScore
	})

	topPerformerID := uint(0)
	topPerformerName := ""
	if len(studentAnalytics) > 0 {
		topPerformerID = studentAnalytics[0].UserID
		topPerformerName = studentAnalytics[0].Name
	}
	if topPerformerID != 0 && strings.TrimSpace(topPerformerName) == "" {
		var user User
		if err := DB.First(&user, topPerformerID).Error; err == nil {
			topPerformerName = user.Name
		}
	}

	totalStudentsFloat := float64(totalStudents)
	if totalStudentsFloat <= 0 {
		totalStudentsFloat = 1
	}
	lastScore := math.NaN()
	lastRank := 0
	for index := range studentAnalytics {
		currentScore := studentAnalytics[index].AvgScore
		if index == 0 {
			lastRank = 1
			lastScore = currentScore
		} else if currentScore != lastScore {
			lastRank = index + 1
			lastScore = currentScore
		}
		studentAnalytics[index].Rank = lastRank
		studentAnalytics[index].Percentile = (float64(studentAnalytics[index].Rank) / totalStudentsFloat) * 100.0
	}

	var courseAverage float64
	if err := DB.Table("evaluations AS e").
		Select("COALESCE(AVG(e.marks), 0)").
		Joins("JOIN submissions AS s ON s.id = e.submission_id").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("a.course_id = ?", courseID).
		Scan(&courseAverage).Error; err != nil {
		return nil, CourseAnalytics{}, err
	}

	type evaluationTopicRow struct {
		StrongTopicsJSON    string
		WeakTopicsJSON      string
		CorrectPointsJSON   string
		WrongPointsJSON     string
		MissingConceptsJSON string
	}

	var topicRows []evaluationTopicRow
	if err := DB.Table("evaluations AS e").
		Select(`
			COALESCE(e.strong_topics::text, '[]') AS strong_topics_json,
			COALESCE(e.weak_topics::text, '[]') AS weak_topics_json,
			COALESCE(e.correct_points::text, '[]') AS correct_points_json,
			COALESCE(e.wrong_points::text, '[]') AS wrong_points_json,
			COALESCE(e.missing_concepts::text, '[]') AS missing_concepts_json
		`).
		Joins("JOIN submissions AS s ON s.id = e.submission_id").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("a.course_id = ?", courseID).
		Scan(&topicRows).Error; err != nil {
		return nil, CourseAnalytics{}, err
	}

	weakTopicCounts := map[string]int{}
	strongTopicCounts := map[string]int{}
	for _, row := range topicRows {
		strongTopics := parseTopicsJSON(row.StrongTopicsJSON)
		if len(strongTopics) == 0 {
			strongTopics = parseTopicsJSON(row.CorrectPointsJSON)
		}

		for _, topic := range normalizeStringList(strongTopics, 12) {
			normalized := strings.TrimSpace(strings.ToLower(topic))
			if normalized == "" {
				continue
			}
			strongTopicCounts[normalized]++
		}

		weakTopics := parseTopicsJSON(row.WeakTopicsJSON)
		if len(weakTopics) == 0 {
			weakTopics = append(weakTopics, parseTopicsJSON(row.WrongPointsJSON)...)
			weakTopics = append(weakTopics, parseTopicsJSON(row.MissingConceptsJSON)...)
		}
		for _, topic := range normalizeStringList(weakTopics, 12) {
			normalized := strings.TrimSpace(strings.ToLower(topic))
			if normalized == "" {
				continue
			}
			weakTopicCounts[normalized]++
		}
	}

	var topPerformer *TopPerformerSummary
	if topPerformerID != 0 {
		topPerformer = &TopPerformerSummary{
			ID:   topPerformerID,
			Name: topPerformerName,
		}
	}

	const topicMinCount = 1
	atRisk := make([]AtRiskStudent, 0)
	performanceDistribution := map[string]int{
		"high":   0,
		"medium": 0,
		"low":    0,
	}
	progressDistribution := map[string]int{
		"Not Started": 0,
		"In Progress": 0,
		"Evaluated":   0,
		"Mastered":    0,
	}
	totalProgress := 0.0
	for _, student := range studentAnalytics {
		switch {
		case student.AvgScore >= 7:
			performanceDistribution["high"]++
		case student.AvgScore >= 4:
			performanceDistribution["medium"]++
		default:
			performanceDistribution["low"]++
		}
		progressDistribution[student.ProgressStatus]++
		totalProgress += student.Progress

		if student.Progress < 40 || student.AvgScore < 4 || student.Trend == "declining" {
			atRisk = append(atRisk, AtRiskStudent{
				UserID:         student.UserID,
				Name:           student.Name,
				AvgScore:       student.AvgScore,
				Progress:       student.Progress,
				ProgressStatus: student.ProgressStatus,
				Trend:          student.Trend,
			})
		}
	}
	sort.Slice(atRisk, func(i, j int) bool {
		if atRisk[i].Progress == atRisk[j].Progress {
			return atRisk[i].AvgScore < atRisk[j].AvgScore
		}
		return atRisk[i].Progress < atRisk[j].Progress
	})

	avgProgress := 0.0
	if len(studentAnalytics) > 0 {
		avgProgress = totalProgress / float64(len(studentAnalytics))
	}

	courseAnalytics := CourseAnalytics{
		CourseID:                courseID,
		AvgScore:                courseAverage,
		AverageProgress:         avgProgress,
		TotalStudents:           totalStudents,
		TopPerformer:            topPerformer,
		WeakTopics:              topTopicsByFrequency(weakTopicCounts, 5, topicMinCount),
		StrongTopics:            topTopicsByFrequency(strongTopicCounts, 5, topicMinCount),
		PerformanceDistribution: performanceDistribution,
		ProgressDistribution:    progressDistribution,
		AtRiskStudents:          atRisk,
	}

	return studentAnalytics, courseAnalytics, nil
}

func GetCourseAnalyticsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, _, _, ok := requireCourseAccess(c)
	if !ok {
		return
	}

	studentAnalytics, courseAnalytics, err := computeCourseAnalyticsSnapshot(courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"students": studentAnalytics,
		"course":   courseAnalytics,
	})
}

func GetCourseLeaderboardHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, _, _, ok := requireCourseAccess(c)
	if !ok {
		return
	}

	studentAnalytics, _, err := computeCourseAnalyticsSnapshot(courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute leaderboard"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"course_id":      courseID,
		"total_students": len(studentAnalytics),
		"leaderboard":    studentAnalytics,
	})
}

func GetStudentRankHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, userID, role, ok := requireCourseAccess(c)
	if !ok {
		return
	}
	if role != "student" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only students can access their rank"})
		return
	}

	studentAnalytics, _, err := computeCourseAnalyticsSnapshot(courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute student rank"})
		return
	}

	var current *StudentAnalytics
	for index := range studentAnalytics {
		if studentAnalytics[index].UserID == userID {
			current = &studentAnalytics[index]
			break
		}
	}
	if current == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student rank not found"})
		return
	}

	topPercentile := math.Ceil(current.Percentile/10.0) * 10.0
	if topPercentile < 1 {
		topPercentile = 1
	}
	if topPercentile > 100 {
		topPercentile = 100
	}

	c.JSON(http.StatusOK, studentRankResponse{
		StudentID:     current.UserID,
		Rank:          current.Rank,
		TotalStudents: len(studentAnalytics),
		Percentile:    current.Percentile,
		TopPercentile: topPercentile,
		AvgScore:      current.AvgScore,
		CourseID:      courseID,
	})
}
