package main

import (
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"gorm.io/gorm"
)

type adaptiveAggregate struct {
	AvgScore float64
}

type recentMarkRow struct {
	Marks float64
}

func updateAdaptiveProfileOnEvaluation(tx *gorm.DB, studentID uint, courseID uint, marks float64, question string, mistakes []string) error {
	if tx == nil {
		return errors.New("database transaction is nil")
	}
	_ = question

	for _, mistake := range normalizeStringList(mistakes, 30) {
		normalized := strings.TrimSpace(strings.ToLower(mistake))
		if normalized == "" {
			continue
		}
		if err := incrementMistakeStat(tx, studentID, courseID, normalized); err != nil {
			return err
		}
	}

	var profile StudentProfile
	err := tx.Where("student_id = ? AND course_id = ?", studentID, courseID).First(&profile).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		profile = StudentProfile{
			StudentID:        studentID,
			CourseID:         courseID,
			AvgScore:         marks,
			TotalSubmissions: 1,
		}
	} else {
		newTotal := profile.TotalSubmissions + 1
		profile.AvgScore = ((profile.AvgScore * float64(profile.TotalSubmissions)) + marks) / float64(newTotal)
		profile.TotalSubmissions = newTotal
	}

	level, err := calculateProficiencyLevel(tx, studentID, courseID, profile.TotalSubmissions, profile.AvgScore)
	if err != nil {
		return err
	}
	profile.ProficiencyLevel = level
	weakTopics, strongTopics, err := buildTopicSnapshots(tx, studentID, courseID)
	if err != nil {
		return err
	}
	profile.WeakTopics = weakTopics
	profile.StrongTopics = strongTopics
	profile.LastUpdated = time.Now()

	if profile.ID == 0 {
		return tx.Create(&profile).Error
	}
	return tx.Save(&profile).Error
}

func recomputeAdaptiveProfile(tx *gorm.DB, studentID uint, courseID uint) (StudentProfile, error) {
	if tx == nil {
		return StudentProfile{}, errors.New("database transaction is nil")
	}

	var aggregate adaptiveAggregate
	if err := tx.Table("evaluations AS e").
		Select("COALESCE(AVG(e.marks), 0) AS avg_score").
		Joins("JOIN submissions AS s ON s.id = e.submission_id").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("s.student_id = ? AND a.course_id = ?", studentID, courseID).
		Scan(&aggregate).Error; err != nil {
		return StudentProfile{}, err
	}

	var submissionCount int64
	if err := tx.Table("submissions AS s").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("s.student_id = ? AND a.course_id = ?", studentID, courseID).
		Count(&submissionCount).Error; err != nil {
		return StudentProfile{}, err
	}

	weakTopics, strongTopics, err := buildTopicSnapshots(tx, studentID, courseID)
	if err != nil {
		return StudentProfile{}, err
	}

	var profile StudentProfile
	err = tx.Where("student_id = ? AND course_id = ?", studentID, courseID).First(&profile).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return StudentProfile{}, err
		}
		level, levelErr := calculateProficiencyLevel(tx, studentID, courseID, int(submissionCount), aggregate.AvgScore)
		if levelErr != nil {
			return StudentProfile{}, levelErr
		}
		profile = StudentProfile{
			StudentID:        studentID,
			CourseID:         courseID,
			AvgScore:         aggregate.AvgScore,
			TotalSubmissions: int(submissionCount),
			ProficiencyLevel: level,
			WeakTopics:       weakTopics,
			StrongTopics:     strongTopics,
			LastUpdated:      time.Now(),
		}
		if err := tx.Create(&profile).Error; err != nil {
			return StudentProfile{}, err
		}
		return profile, nil
	}

	profile.AvgScore = aggregate.AvgScore
	profile.TotalSubmissions = int(submissionCount)
	level, levelErr := calculateProficiencyLevel(tx, studentID, courseID, int(submissionCount), aggregate.AvgScore)
	if levelErr != nil {
		return StudentProfile{}, levelErr
	}
	profile.ProficiencyLevel = level
	profile.WeakTopics = weakTopics
	profile.StrongTopics = strongTopics
	profile.LastUpdated = time.Now()
	if err := tx.Save(&profile).Error; err != nil {
		return StudentProfile{}, err
	}
	return profile, nil
}

func getOrBuildAdaptiveProfile(studentID uint, courseID uint) (StudentProfile, error) {
	var profile StudentProfile
	err := DB.Where("student_id = ? AND course_id = ?", studentID, courseID).First(&profile).Error
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return StudentProfile{}, err
	}
	return recomputeAdaptiveProfile(DB, studentID, courseID)
}

func calculateProficiencyLevel(tx *gorm.DB, studentID uint, courseID uint, totalSubmissions int, fallbackAvgScore float64) (string, error) {
	// Conservative rule: do not classify aggressively with sparse history.
	if totalSubmissions < 3 {
		return "beginner", nil
	}

	var recent []recentMarkRow
	if err := tx.Table("evaluations AS e").
		Select("e.marks").
		Joins("JOIN submissions AS s ON s.id = e.submission_id").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("s.student_id = ? AND a.course_id = ?", studentID, courseID).
		Order("e.created_at DESC, e.id DESC").
		Limit(3).
		Scan(&recent).Error; err != nil {
		return "", err
	}

	smoothed := fallbackAvgScore
	if len(recent) > 0 {
		sum := 0.0
		for _, row := range recent {
			sum += row.Marks
		}
		smoothed = sum / float64(len(recent))
	}

	switch {
	case smoothed < 4:
		return "beginner", nil
	case smoothed <= 7:
		return "intermediate", nil
	default:
		return "advanced", nil
	}
}

func incrementMistakeStat(tx *gorm.DB, studentID uint, courseID uint, mistake string) error {
	var row StudentMistakeStat
	err := tx.Where("student_id = ? AND course_id = ? AND mistake = ?", studentID, courseID, mistake).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = StudentMistakeStat{
			StudentID: studentID,
			CourseID:  courseID,
			Mistake:   mistake,
			Count:     1,
		}
		return tx.Create(&row).Error
	}

	row.Count++
	return tx.Save(&row).Error
}

func incrementStrengthStat(tx *gorm.DB, studentID uint, courseID uint, topic string) error {
	var row StudentStrengthStat
	err := tx.Where("student_id = ? AND course_id = ? AND topic = ?", studentID, courseID, topic).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = StudentStrengthStat{
			StudentID: studentID,
			CourseID:  courseID,
			Topic:     topic,
			Count:     1,
		}
		return tx.Create(&row).Error
	}

	row.Count++
	return tx.Save(&row).Error
}

func incrementChatTopicStat(tx *gorm.DB, studentID uint, courseID uint, topic string) error {
	var row StudentChatTopicStat
	err := tx.Where("student_id = ? AND course_id = ? AND topic = ?", studentID, courseID, topic).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = StudentChatTopicStat{
			StudentID: studentID,
			CourseID:  courseID,
			Topic:     topic,
			Count:     1,
		}
		return tx.Create(&row).Error
	}

	row.Count++
	return tx.Save(&row).Error
}

func buildTopicSnapshots(tx *gorm.DB, studentID uint, courseID uint) (string, string, error) {
	type evaluationTopicRow struct {
		StrongTopicsJSON    string
		WeakTopicsJSON      string
		CorrectPointsJSON   string
		WrongPointsJSON     string
		MissingConceptsJSON string
	}

	var rows []evaluationTopicRow
	if err := tx.Table("evaluations AS e").
		Select(`
			COALESCE(e.strong_topics::text, '[]') AS strong_topics_json,
			COALESCE(e.weak_topics::text, '[]') AS weak_topics_json,
			COALESCE(e.correct_points::text, '[]') AS correct_points_json,
			COALESCE(e.wrong_points::text, '[]') AS wrong_points_json,
			COALESCE(e.missing_concepts::text, '[]') AS missing_concepts_json
		`).
		Joins("JOIN submissions AS s ON s.id = e.submission_id").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("s.student_id = ? AND a.course_id = ?", studentID, courseID).
		Order("e.created_at DESC, e.id DESC").
		Scan(&rows).Error; err != nil {
		return "", "", err
	}

	weakScores := map[string]int{}
	strongScores := map[string]int{}
	for _, row := range rows {
		strongTopics := parseTopicsJSON(row.StrongTopicsJSON)
		if len(strongTopics) == 0 {
			strongTopics = parseTopicsJSON(row.CorrectPointsJSON)
		}
		for _, topic := range normalizeStringList(strongTopics, 12) {
			normalized := strings.TrimSpace(strings.ToLower(topic))
			if normalized == "" {
				continue
			}
			strongScores[normalized]++
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
			weakScores[normalized]++
		}
	}

	weakTopics := topTopicsByFrequency(weakScores, 5, 1)
	strongTopics := topTopicsByFrequency(strongScores, 5, 1)

	weakJSON, err := json.Marshal(weakTopics)
	if err != nil {
		return "", "", err
	}
	strongJSON, err := json.Marshal(strongTopics)
	if err != nil {
		return "", "", err
	}
	return string(weakJSON), string(strongJSON), nil
}

func extractTopicsFromTextItems(items []string, limit int) []string {
	if limit <= 0 {
		limit = 8
	}
	phraseMap := []struct {
		Topic   string
		Phrases []string
	}{
		{Topic: "self-attention", Phrases: []string{"self attention", "self-attention", "selfattention"}},
		{Topic: "cross-attention", Phrases: []string{"cross attention", "cross-attention"}},
		{Topic: "multi-head attention", Phrases: []string{"multi head attention", "multi-head attention"}},
		{Topic: "bert", Phrases: []string{"bert"}},
		{Topic: "gpt", Phrases: []string{"gpt"}},
		{Topic: "transformer", Phrases: []string{"transformer"}},
		{Topic: "tokenization", Phrases: []string{"tokenization", "tokenizer"}},
		{Topic: "positional encoding", Phrases: []string{"positional encoding", "position encoding"}},
		{Topic: "encoder", Phrases: []string{"encoder"}},
		{Topic: "decoder", Phrases: []string{"decoder"}},
		{Topic: "embedding", Phrases: []string{"embedding", "embeddings"}},
		{Topic: "fine-tuning", Phrases: []string{"fine tuning", "fine-tuning", "finetuning"}},
		{Topic: "gradient descent", Phrases: []string{"gradient descent"}},
		{Topic: "backpropagation", Phrases: []string{"backpropagation", "back propagation"}},
	}

	seen := map[string]struct{}{}
	topics := make([]string, 0, limit)
	for _, item := range normalizeStringList(items, 50) {
		text := strings.TrimSpace(strings.ToLower(item))
		if text == "" {
			continue
		}
		text = strings.Join(strings.Fields(text), " ")

		// Explicit required mappings.
		if strings.Contains(text, "bert") && strings.Contains(text, "direction") {
			if _, exists := seen["bert"]; !exists {
				seen["bert"] = struct{}{}
				topics = append(topics, "bert")
				if len(topics) >= limit {
					return topics
				}
			}
		}
		if (strings.Contains(text, "missing") || strings.Contains(text, "missed")) &&
			(strings.Contains(text, "self attention") || strings.Contains(text, "self-attention")) {
			if _, exists := seen["self-attention"]; !exists {
				seen["self-attention"] = struct{}{}
				topics = append(topics, "self-attention")
				if len(topics) >= limit {
					return topics
				}
			}
		}

		for _, mapping := range phraseMap {
			if _, exists := seen[mapping.Topic]; exists {
				continue
			}
			matched := false
			for _, phrase := range mapping.Phrases {
				if strings.Contains(text, phrase) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
			seen[mapping.Topic] = struct{}{}
			topics = append(topics, mapping.Topic)
			if len(topics) >= limit {
				return topics
			}
		}
	}

	return topics
}

func fallbackLegacyTopicStats(tx *gorm.DB, studentID uint, courseID uint) ([]string, []string, error) {
	var mistakeRows []StudentMistakeStat
	if err := tx.Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("count DESC, updated_at DESC").
		Limit(20).
		Find(&mistakeRows).Error; err != nil {
		return nil, nil, err
	}

	var strengthRows []StudentStrengthStat
	if err := tx.Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("count DESC, updated_at DESC").
		Limit(20).
		Find(&strengthRows).Error; err != nil {
		return nil, nil, err
	}

	var chatRows []StudentChatTopicStat
	if err := tx.Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("count DESC, updated_at DESC").
		Limit(20).
		Find(&chatRows).Error; err != nil {
		return nil, nil, err
	}

	weakScores := map[string]int{}
	for _, row := range mistakeRows {
		topic := strings.TrimSpace(strings.ToLower(row.Mistake))
		if topic == "" {
			continue
		}
		weakScores[topic] += row.Count
	}
	for _, row := range chatRows {
		topic := strings.TrimSpace(strings.ToLower(row.Topic))
		if topic == "" {
			continue
		}
		weakScores[topic] += row.Count
	}

	strongScores := map[string]int{}
	for _, row := range strengthRows {
		topic := strings.TrimSpace(strings.ToLower(row.Topic))
		if topic == "" {
			continue
		}
		strongScores[topic] += row.Count
	}

	return topTopicsByFrequency(weakScores, 5, 1), topTopicsByFrequency(strongScores, 5, 1), nil
}

func normalizeStringList(items []string, maxItems int) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
		if maxItems > 0 && len(result) >= maxItems {
			break
		}
	}
	return result
}

func extractTopicKeywords(text string, limit int) []string {
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, text)

	stopwords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "from": {}, "that": {}, "this": {},
		"have": {}, "has": {}, "were": {}, "what": {}, "when": {}, "where": {}, "which": {},
		"why": {}, "how": {}, "into": {}, "about": {}, "define": {}, "explain": {}, "list": {},
		"state": {}, "write": {}, "question": {}, "answer": {}, "course": {}, "student": {},
	}

	freq := map[string]int{}
	for _, token := range strings.Fields(cleaned) {
		if len(token) < 4 {
			continue
		}
		if _, blocked := stopwords[token]; blocked {
			continue
		}
		freq[token]++
	}

	type pair struct {
		Token string
		Count int
	}
	pairs := make([]pair, 0, len(freq))
	for token, count := range freq {
		pairs = append(pairs, pair{Token: token, Count: count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count == pairs[j].Count {
			return pairs[i].Token < pairs[j].Token
		}
		return pairs[i].Count > pairs[j].Count
	})

	if limit <= 0 {
		limit = 5
	}
	if len(pairs) < limit {
		limit = len(pairs)
	}

	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, pairs[i].Token)
	}
	return result
}

func extractMistakeTopicKeywords(mistakes []string, limit int) []string {
	if limit <= 0 {
		limit = 12
	}
	seen := map[string]struct{}{}
	topics := make([]string, 0, limit)
	for _, mistake := range normalizeStringList(mistakes, 30) {
		keywords := extractTopicKeywords(mistake, 5)
		if len(keywords) == 0 {
			keywords = []string{strings.TrimSpace(mistake)}
		}
		for _, keyword := range keywords {
			item := strings.TrimSpace(keyword)
			if item == "" {
				continue
			}
			if _, exists := seen[item]; exists {
				continue
			}
			seen[item] = struct{}{}
			topics = append(topics, item)
			if len(topics) >= limit {
				return topics
			}
		}
	}
	return topics
}

func topTopicsByFrequency(scores map[string]int, limit int, minCount int) []string {
	type pair struct {
		Topic string
		Score int
	}
	pairs := make([]pair, 0, len(scores))
	for topic, score := range scores {
		if strings.TrimSpace(topic) == "" {
			continue
		}
		if score < minCount {
			continue
		}
		pairs = append(pairs, pair{Topic: topic, Score: score})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Score == pairs[j].Score {
			return pairs[i].Topic < pairs[j].Topic
		}
		return pairs[i].Score > pairs[j].Score
	})

	if limit <= 0 {
		limit = 5
	}
	if len(pairs) < limit {
		limit = len(pairs)
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, pairs[i].Topic)
	}
	return result
}

func parseTopicsJSON(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return []string{}
	}
	var parsed []string
	if err := json.Unmarshal([]byte(value), &parsed); err != nil {
		return []string{}
	}
	return parsed
}

func listRecentMistakes(studentID uint, courseID uint, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5
	}

	var rows []StudentMistakeStat
	if err := DB.Where("student_id = ? AND course_id = ?", studentID, courseID).
		Order("updated_at DESC, count DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]string, 0, len(rows))
	for _, row := range rows {
		item := strings.TrimSpace(row.Mistake)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func countCourseAssignments(courseID uint) (int, error) {
	var count int64
	if err := DB.Model(&Assignment{}).Where("course_id = ?", courseID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func computeProgress(totalSubmissions int, totalAssignments int) float64 {
	if totalAssignments <= 0 || totalSubmissions <= 0 {
		return 0
	}
	progress := (float64(totalSubmissions) / float64(totalAssignments)) * 100.0
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

type unitProgressComputation struct {
	UnitID         uint
	Performance    float64
	Source         string
	IsEstimated    bool
	SignalSummary  string
	Weight         float64
}

func estimateSignalPerformance(studentID uint, courseID uint) (float64, string, error) {
	var interactionCount int64
	if err := DB.Model(&StudentInteraction{}).
		Where("student_id = ? AND course_id = ?", studentID, courseID).
		Count(&interactionCount).Error; err != nil {
		return 0, "", err
	}

	var materialRows []MaterialEngagement
	if err := DB.Where("student_id = ? AND course_id = ?", studentID, courseID).Find(&materialRows).Error; err != nil {
		return 0, "", err
	}
	materialTouches := 0
	for _, row := range materialRows {
		materialTouches += row.AccessCount
	}

	score := (float64(interactionCount) * 9.0) + (float64(materialTouches) * 4.0)
	score = math.Max(0, math.Min(100, score))
	summary := "estimated from interaction and engagement signals"
	if interactionCount == 0 && materialTouches == 0 {
		summary = "no interaction/engagement signals detected"
	}
	return score, summary, nil
}

func upsertStudentUnitProgress(studentID uint, courseID uint, unit unitProgressComputation) error {
	now := time.Now()
	var row StudentUnitProgress
	err := DB.Where("student_id = ? AND course_id = ? AND unit_id = ?", studentID, courseID, unit.UnitID).First(&row).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row = StudentUnitProgress{
			StudentID:       studentID,
			CourseID:        courseID,
			UnitID:          unit.UnitID,
			Performance:     unit.Performance,
			Source:          unit.Source,
			IsEstimated:     unit.IsEstimated,
			SignalSummary:   unit.SignalSummary,
			LastEvaluatedAt: now,
		}
		return DB.Create(&row).Error
	}

	row.Performance = unit.Performance
	row.Source = unit.Source
	row.IsEstimated = unit.IsEstimated
	row.SignalSummary = unit.SignalSummary
	row.LastEvaluatedAt = now
	return DB.Save(&row).Error
}

func progressStatus(progress float64, hasAssignmentEvidence bool, hasAnySignal bool) string {
	if !hasAnySignal || progress <= 0.01 {
		return "Not Started"
	}
	if hasAssignmentEvidence {
		if progress >= 75 {
			return "Mastered"
		}
		return "Evaluated"
	}
	return "In Progress"
}

func courseProgressForStudent(studentID uint, courseID uint, totalSubmissions int) (float64, string, int, bool, string, error) {
	totalAssignments, err := countCourseAssignments(courseID)
	if err != nil {
		return 0, "", 0, false, "", err
	}

	var units []CourseUnit
	if err := DB.Where("course_id = ?", courseID).Order("unit_order ASC, id ASC").Find(&units).Error; err != nil {
		return 0, "", 0, false, "", err
	}

	// Backward-compatible fallback when no unit plan exists.
	if len(units) == 0 {
		progress := computeProgress(totalSubmissions, totalAssignments)
		status := progressStatus(progress, totalSubmissions > 0, totalSubmissions > 0)
		return progress, status, totalAssignments, false, "", nil
	}

	unitProgresses := make([]unitProgressComputation, 0, len(units))
	hasAssignmentEvidence := false
	hasAnySignal := false

	for _, unit := range units {
		weight := unit.Weight
		if weight <= 0 {
			weight = 1
		}

		var assignmentCount int64
		if err := DB.Model(&Assignment{}).Where("course_id = ? AND unit_id = ?", courseID, unit.ID).Count(&assignmentCount).Error; err != nil {
			return 0, "", 0, false, "", err
		}

		if assignmentCount > 0 {
			type avgRow struct {
				AvgMarks float64
			}
			var row avgRow
			if err := DB.Table("evaluations AS e").
				Select("COALESCE(AVG(e.marks), 0) AS avg_marks").
				Joins("JOIN submissions AS s ON s.id = e.submission_id").
				Joins("JOIN assignments AS a ON a.id = s.assignment_id").
				Where("s.student_id = ? AND a.course_id = ? AND a.unit_id = ?", studentID, courseID, unit.ID).
				Scan(&row).Error; err != nil {
				return 0, "", 0, false, "", err
			}
			performance := math.Max(0, math.Min(100, row.AvgMarks*10.0))
			if performance > 0 {
				hasAnySignal = true
				hasAssignmentEvidence = true
			}
			unitProgress := unitProgressComputation{
				UnitID:        unit.ID,
				Performance:   performance,
				Source:        "assignment",
				IsEstimated:   false,
				SignalSummary: "assignment-based evaluation",
				Weight:        weight,
			}
			unitProgresses = append(unitProgresses, unitProgress)
			if err := upsertStudentUnitProgress(studentID, courseID, unitProgress); err != nil {
				return 0, "", 0, false, "", err
			}
			continue
		}

		estimatedPerformance, summary, estimateErr := estimateSignalPerformance(studentID, courseID)
		if estimateErr != nil {
			return 0, "", 0, false, "", estimateErr
		}
		if estimatedPerformance > 0 {
			hasAnySignal = true
		}
		unitProgress := unitProgressComputation{
			UnitID:        unit.ID,
			Performance:   estimatedPerformance,
			Source:        "inferred",
			IsEstimated:   true,
			SignalSummary: summary,
			Weight:        weight,
		}
		unitProgresses = append(unitProgresses, unitProgress)
		if err := upsertStudentUnitProgress(studentID, courseID, unitProgress); err != nil {
			return 0, "", 0, false, "", err
		}
	}

	totalWeight := 0.0
	weightedSum := 0.0
	for _, row := range unitProgresses {
		totalWeight += row.Weight
		weightedSum += row.Weight * row.Performance
	}
	progress := 0.0
	if totalWeight > 0 {
		progress = weightedSum / totalWeight
	}
	progress = math.Max(0, math.Min(100, progress))

	estimated := !hasAssignmentEvidence && hasAnySignal
	note := ""
	if estimated {
		note = "Progress estimated based on interaction and engagement due to absence of assignment-based evaluation."
	}

	return progress, progressStatus(progress, hasAssignmentEvidence, hasAnySignal), totalAssignments, estimated, note, nil
}

func learningTrend(studentID uint, courseID uint) (string, error) {
	type row struct {
		Marks float64
	}
	var rows []row
	if err := DB.Table("evaluations AS e").
		Select("e.marks").
		Joins("JOIN submissions AS s ON s.id = e.submission_id").
		Joins("JOIN assignments AS a ON a.id = s.assignment_id").
		Where("s.student_id = ? AND a.course_id = ?", studentID, courseID).
		Order("e.created_at DESC, e.id DESC").
		Limit(6).
		Scan(&rows).Error; err != nil {
		return "", err
	}

	if len(rows) < 2 {
		return "improving", nil
	}

	latest := rows[0].Marks
	previous := rows[1].Marks
	if latest-previous >= 0.25 {
		return "improving", nil
	}
	if previous-latest >= 0.25 {
		return "declining", nil
	}
	return "improving", nil
}
