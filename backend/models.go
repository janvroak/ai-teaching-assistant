package main

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Email     string    `gorm:"unique;not null"`
	Password  string    `gorm:"not null"`
	Role      string    `gorm:"not null"` // "student" or "professor"
	CreatedAt time.Time `gorm:"not null"`
}

type Course struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"not null"`
	CourseCode  string    `gorm:"unique;not null"`
	ProfessorID uint      `gorm:"not null"`
	CreatedAt   time.Time `gorm:"not null"`
}

type Enrollment struct {
	ID        uint `gorm:"primaryKey"`
	StudentID uint `gorm:"not null"`
	CourseID  uint `gorm:"not null"`
}

type Assignment struct {
	ID                uint      `gorm:"primaryKey"`
	CourseID          uint      `gorm:"not null"`
	UnitID            *uint     `gorm:"index"`
	Title             string    `gorm:"not null"`
	Question          string    `gorm:"type:text;not null"`
	QuestionFilePath  string    `gorm:"type:text"`
	AnswerKey         string    `gorm:"type:text;not null"`
	RubricJSON        string    `gorm:"type:text;not null;default:'[]'"`
	AnswerKeyFilePath string    `gorm:"type:text"`
	CreatedAt         time.Time `gorm:"not null"`
}

type Submission struct {
	ID           uint      `gorm:"primaryKey"`
	StudentID    uint      `gorm:"not null"`
	AssignmentID uint      `gorm:"not null"`
	Content      string    `gorm:"type:text;not null"`
	FilePath     string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"not null"`
}

type Evaluation struct {
	ID                uint      `gorm:"primaryKey"`
	SubmissionID      uint      `gorm:"not null"`
	Marks             float64   `gorm:"not null"`
	Feedback          string    `gorm:"type:text;not null"`
	Confidence        float64   `gorm:"not null"`
	Mistakes          []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"mistakes"`
	CorrectPoints     []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"correct_points"`
	WrongPoints       []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"wrong_points"`
	MissingConcepts   []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"missing_concepts"`
	StrongTopics      []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"strong_topics"`
	WeakTopics        []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"weak_topics"`
	Topics            []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"topics"`
	IsFinal           bool      `gorm:"not null;default:false"`
	AIOriginalMarks   float64   `gorm:"not null;default:0"`
	AIOriginalFeedback string   `gorm:"type:text;not null;default:''"`
	FinalizedAt       *time.Time
	CreatedAt         time.Time `gorm:"not null"`
}

type EvaluationQuestion struct {
	ID                uint      `gorm:"primaryKey"`
	EvaluationID      uint      `gorm:"not null;index"`
	QuestionText      string    `gorm:"type:text;not null"`
	StudentAnswerText string    `gorm:"type:text;not null;default:''"`
	Marks             float64   `gorm:"not null"`
	Feedback          string    `gorm:"type:text;not null"`
	Confidence        float64   `gorm:"not null"`
	CorrectPoints     []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"correct_points"`
	WrongPoints       []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"wrong_points"`
	MissingConcepts   []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"missing_concepts"`
	StrongTopics      []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"strong_topics"`
	WeakTopics        []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"weak_topics"`
	Mistakes          []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"mistakes"`
	Topics            []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"topics"`
	AIOriginalMarks   float64   `gorm:"not null;default:0"`
	AIOriginalFeedback string   `gorm:"type:text;not null;default:''"`
	CreatedAt         time.Time `gorm:"not null"`
}

type EvaluationAuditLog struct {
	ID           uint      `gorm:"primaryKey"`
	EvaluationID uint      `gorm:"not null;index"`
	EditorID     uint      `gorm:"not null;index"`
	Action       string    `gorm:"not null"`
	BeforeState  string    `gorm:"type:text;not null"`
	AfterState   string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

type CourseMaterial struct {
	ID        uint      `gorm:"primaryKey"`
	CourseID  uint      `gorm:"not null;index"`
	Title     string    `gorm:"not null"`
	Content   string    `gorm:"type:text;not null"`
	FilePath  string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"not null"`
}

type StudentProfile struct {
	ID               uint      `gorm:"primaryKey"`
	StudentID        uint      `gorm:"not null;index:idx_student_course_profile,unique"`
	CourseID         uint      `gorm:"not null;index:idx_student_course_profile,unique"`
	ProficiencyLevel string    `gorm:"not null;default:beginner"`
	AvgScore         float64   `gorm:"not null;default:0"`
	TotalSubmissions int       `gorm:"not null;default:0"`
	WeakTopics       string    `gorm:"type:text;not null;default:'[]'"`
	StrongTopics     string    `gorm:"type:text;not null;default:'[]'"`
	LastUpdated      time.Time `gorm:"not null"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

type StudentInteraction struct {
	ID               uint      `gorm:"primaryKey"`
	StudentID        uint      `gorm:"not null;index"`
	CourseID         uint      `gorm:"not null;index"`
	InteractionType  string    `gorm:"not null"`
	Question         string    `gorm:"type:text;not null"`
	Response         string    `gorm:"type:text;not null"`
	ProficiencyLevel string    `gorm:"not null"`
	CreatedAt        time.Time `gorm:"not null"`
}

type StudentMistakeStat struct {
	ID        uint      `gorm:"primaryKey"`
	StudentID uint      `gorm:"not null;index:idx_student_course_mistake,unique"`
	CourseID  uint      `gorm:"not null;index:idx_student_course_mistake,unique"`
	Mistake   string    `gorm:"type:text;not null;index:idx_student_course_mistake,unique"`
	Count     int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

type CourseUnit struct {
	ID          uint      `gorm:"primaryKey"`
	CourseID    uint      `gorm:"not null;index:idx_course_unit_order,priority:1"`
	Name        string    `gorm:"not null"`
	Weight      float64   `gorm:"not null;default:1"`
	UnitOrder   int       `gorm:"not null;default:0;index:idx_course_unit_order,priority:2"`
	Description string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

type StudentUnitProgress struct {
	ID               uint      `gorm:"primaryKey"`
	StudentID        uint      `gorm:"not null;index:idx_student_course_unit_progress,priority:1"`
	CourseID         uint      `gorm:"not null;index:idx_student_course_unit_progress,priority:2"`
	UnitID           uint      `gorm:"not null;index:idx_student_course_unit_progress,priority:3"`
	Performance      float64   `gorm:"not null;default:0"` // 0..100
	Source           string    `gorm:"not null;default:assignment"` // assignment|inferred
	IsEstimated      bool      `gorm:"not null;default:false"`
	SignalSummary    string    `gorm:"type:text;not null;default:''"`
	LastEvaluatedAt  time.Time `gorm:"not null"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

type MaterialEngagement struct {
	ID           uint      `gorm:"primaryKey"`
	StudentID    uint      `gorm:"not null;index:idx_student_course_material,priority:1"`
	CourseID     uint      `gorm:"not null;index:idx_student_course_material,priority:2"`
	MaterialID   uint      `gorm:"not null;index:idx_student_course_material,priority:3"`
	AccessCount  int       `gorm:"not null;default:1"`
	LastAccessed time.Time `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null"`
}

type PlagiarismReport struct {
	ID                     uint      `gorm:"primaryKey"`
	SubmissionID           uint      `gorm:"not null;index"`
	AssignmentID           uint      `gorm:"not null;index"`
	CourseID               uint      `gorm:"not null;index"`
	StudentID              uint      `gorm:"not null;index"`
	ComparedSubmissionID   *uint     `gorm:"index"`
	SemanticSimilarity     float64   `gorm:"not null;default:0"` // 0..1
	StyleAnomalyScore      float64   `gorm:"not null;default:0"` // 0..1
	AIUsageLikelihood      float64   `gorm:"not null;default:0"` // 0..1
	PlagiarismLikelihood   float64   `gorm:"not null;default:0"` // 0..1
	Flagged                bool      `gorm:"not null;default:false"`
	ReasonSummary          string    `gorm:"type:text;not null;default:''"`
	Reasons                []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'"`
	CreatedAt              time.Time `gorm:"not null"`
}

type StudentStrengthStat struct {
	ID        uint      `gorm:"primaryKey"`
	StudentID uint      `gorm:"not null;index:idx_student_course_strength,unique"`
	CourseID  uint      `gorm:"not null;index:idx_student_course_strength,unique"`
	Topic     string    `gorm:"type:text;not null;index:idx_student_course_strength,unique"`
	Count     int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

type StudentChatTopicStat struct {
	ID        uint      `gorm:"primaryKey"`
	StudentID uint      `gorm:"not null;index:idx_student_course_chat_topic,unique"`
	CourseID  uint      `gorm:"not null;index:idx_student_course_chat_topic,unique"`
	Topic     string    `gorm:"type:text;not null;index:idx_student_course_chat_topic,unique"`
	Count     int       `gorm:"not null;default:1"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}
