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
	Title             string    `gorm:"not null"`
	Question          string    `gorm:"type:text;not null"`
	QuestionFilePath  string    `gorm:"type:text"`
	AnswerKey         string    `gorm:"type:text;not null"`
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
	ID              uint      `gorm:"primaryKey"`
	SubmissionID    uint      `gorm:"not null"`
	Marks           float64   `gorm:"not null"`
	Feedback        string    `gorm:"type:text;not null"`
	Confidence      float64   `gorm:"not null"`
	Mistakes        []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"mistakes"`
	CorrectPoints   []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"correct_points"`
	WrongPoints     []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"wrong_points"`
	MissingConcepts []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"missing_concepts"`
	StrongTopics    []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"strong_topics"`
	WeakTopics      []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"weak_topics"`
	Topics          []string  `gorm:"serializer:json;type:jsonb;not null;default:'[]'" json:"topics"`
	IsFinal         bool      `gorm:"not null;default:false"`
	CreatedAt       time.Time `gorm:"not null"`
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
