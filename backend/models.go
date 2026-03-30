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
	ID        uint      `gorm:"primaryKey"`
	CourseID  uint      `gorm:"not null"`
	Title     string    `gorm:"not null"`
	Question  string    `gorm:"type:text;not null"`
	AnswerKey string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null"`
}

type Submission struct {
	ID           uint      `gorm:"primaryKey"`
	StudentID    uint      `gorm:"not null"`
	AssignmentID uint      `gorm:"not null"`
	Content      string    `gorm:"type:text;not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

type Evaluation struct {
	ID           uint      `gorm:"primaryKey"`
	SubmissionID uint      `gorm:"not null"`
	Marks        float64   `gorm:"not null"`
	Feedback     string    `gorm:"type:text;not null"`
	Confidence   float64   `gorm:"not null"`
	IsFinal      bool      `gorm:"not null;default:false"`
	CreatedAt    time.Time `gorm:"not null"`
}
