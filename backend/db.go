package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() *gorm.DB {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		log.Fatal("missing required database environment variables")
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost,
		dbPort,
		dbUser,
		dbPassword,
		dbName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get underlying DB: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("database is not reachable: %v", err)
	}

	// Backward-compatible schema fix:
	// Existing deployments may already have assignments rows, so adding a NOT NULL
	// question column directly can fail. Add column if missing, backfill from title,
	// then enforce NOT NULL before AutoMigrate.
if err := db.Exec(`
DO $$
BEGIN
	IF to_regclass('public.assignments') IS NOT NULL THEN
		ALTER TABLE assignments ADD COLUMN IF NOT EXISTS question text;
		ALTER TABLE assignments ADD COLUMN IF NOT EXISTS rubric_json text NOT NULL DEFAULT '[]';
		UPDATE assignments
		SET question = title
		WHERE question IS NULL OR btrim(question) = '';
		UPDATE assignments
		SET rubric_json = '[]'
		WHERE rubric_json IS NULL OR btrim(rubric_json) = '';
		ALTER TABLE assignments ALTER COLUMN question SET NOT NULL;
	END IF;
END $$;
`).Error; err != nil {
		log.Fatalf("failed to prepare assignments.question migration: %v", err)
	}

	if err := db.Exec(`
DO $$
BEGIN
	IF to_regclass('public.evaluations') IS NOT NULL THEN
		ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS ai_original_marks double precision NOT NULL DEFAULT 0;
		ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS ai_original_feedback text NOT NULL DEFAULT '';
		ALTER TABLE evaluations ADD COLUMN IF NOT EXISTS finalized_at timestamp with time zone;
		UPDATE evaluations
		SET ai_original_marks = marks,
		    ai_original_feedback = feedback
		WHERE ai_original_marks = 0 OR btrim(ai_original_feedback) = '';
	END IF;
END $$;
`).Error; err != nil {
		log.Fatalf("failed to prepare evaluations migration: %v", err)
	}

	if err := db.AutoMigrate(
		&User{},
		&Course{},
		&Enrollment{},
		&CourseUnit{},
		&Assignment{},
		&Submission{},
		&Evaluation{},
		&EvaluationQuestion{},
		&EvaluationAuditLog{},
		&CourseMaterial{},
		&MaterialEngagement{},
		&StudentProfile{},
		&StudentUnitProgress{},
		&StudentInteraction{},
		&StudentMistakeStat{},
		&StudentStrengthStat{},
		&StudentChatTopicStat{},
		&PlagiarismReport{},
	); err != nil {
		log.Fatalf("failed to run automigrate: %v", err)
	}

	log.Println("Connected to PostgreSQL successfully")

	return db
}
