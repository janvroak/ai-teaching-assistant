package main

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateCourseRequest struct {
	Name string `json:"name"`
}

type JoinCourseRequest struct {
	CourseCode string `json:"course_code"`
}

type CourseResponse struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	CourseCode string `json:"course_code"`
}

func CreateCourseHandler(c *gin.Context) {
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
			"error": "Only professors can create courses",
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

	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "name is required",
		})
		return
	}

	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		courseCode := GenerateCourseCode()

		course := Course{
			Name:        req.Name,
			CourseCode:  courseCode,
			ProfessorID: professorID,
		}

		if err := DB.Create(&course).Error; err != nil {
			if isDuplicateCourseCodeError(err) {
				continue
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create course",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":          course.ID,
			"name":        course.Name,
			"course_code": course.CourseCode,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "Failed to generate unique course code",
	})
}

func JoinCourseHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	roleValue, ok := c.Get("role")
	role, roleOK := roleValue.(string)
	if !ok || !roleOK || strings.ToLower(strings.TrimSpace(role)) != "student" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Only students can join courses",
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

	studentID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid user ID in token context",
		})
		return
	}

	var req JoinCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.CourseCode = strings.TrimSpace(strings.ToUpper(req.CourseCode))
	if req.CourseCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "course_code is required",
		})
		return
	}

	var course Course
	if err := DB.Where("course_code = ?", req.CourseCode).First(&course).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Course not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch course",
		})
		return
	}

	var existing Enrollment
	err := DB.Where("student_id = ? AND course_id = ?", studentID, course.ID).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "You are already enrolled in this course",
		})
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check enrollment",
		})
		return
	}

	enrollment := Enrollment{
		StudentID: studentID,
		CourseID:  course.ID,
	}
	if err := DB.Create(&enrollment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to join course",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Joined course successfully",
	})
}

func ListCoursesHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

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

	var courses []CourseResponse

	switch role {
	case "professor":
		if err := DB.Model(&Course{}).
			Where("professor_id = ?", userID).
			Select("id", "name", "course_code").
			Find(&courses).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch courses",
			})
			return
		}
	case "student":
		if err := DB.Table("courses").
			Select("courses.id, courses.name, courses.course_code").
			Joins("JOIN enrollments ON enrollments.course_id = courses.id").
			Where("enrollments.student_id = ?", userID).
			Find(&courses).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch courses",
			})
			return
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid role",
		})
		return
	}

	c.JSON(http.StatusOK, courses)
}

func DeleteCourseHandler(c *gin.Context) {
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
			"error": "Only professors can delete courses",
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

	courseIDParam := c.Param("id")
	parsedCourseID, err := strconv.ParseUint(courseIDParam, 10, 64)
	if err != nil || parsedCourseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid course id",
		})
		return
	}
	courseID := uint(parsedCourseID)

	var course Course
	if err := DB.Where("id = ? AND professor_id = ?", courseID, professorID).First(&course).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only delete your own courses",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify course ownership",
		})
		return
	}

	filePaths := make([]string, 0)

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

	var materials []CourseMaterial
	if err := tx.Where("course_id = ?", courseID).Find(&materials).Error; err != nil {
		rollbackWithError("Failed to fetch course materials")
		return
	}
	for _, material := range materials {
		path := strings.TrimSpace(material.FilePath)
		if path != "" {
			filePaths = append(filePaths, path)
		}
	}

	var assignments []Assignment
	if err := tx.Where("course_id = ?", courseID).Find(&assignments).Error; err != nil {
		rollbackWithError("Failed to fetch assignments")
		return
	}

	assignmentIDs := make([]uint, 0, len(assignments))
	for _, assignment := range assignments {
		assignmentIDs = append(assignmentIDs, assignment.ID)
		if path := strings.TrimSpace(assignment.QuestionFilePath); path != "" {
			filePaths = append(filePaths, path)
		}
		if path := strings.TrimSpace(assignment.AnswerKeyFilePath); path != "" {
			filePaths = append(filePaths, path)
		}
	}

	if len(assignmentIDs) > 0 {
		var submissions []Submission
		if err := tx.Where("assignment_id IN ?", assignmentIDs).Find(&submissions).Error; err != nil {
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
			if err := tx.Model(&Evaluation{}).
				Where("submission_id IN ?", submissionIDs).
				Pluck("id", &evaluationIDs).Error; err != nil {
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

		if err := tx.Where("assignment_id IN ?", assignmentIDs).Delete(&Submission{}).Error; err != nil {
			rollbackWithError("Failed to delete submissions")
			return
		}

		if err := tx.Where("id IN ?", assignmentIDs).Delete(&Assignment{}).Error; err != nil {
			rollbackWithError("Failed to delete assignments")
			return
		}
	}

	if err := tx.Where("course_id = ?", courseID).Delete(&CourseMaterial{}).Error; err != nil {
		rollbackWithError("Failed to delete course materials")
		return
	}

	if err := tx.Where("course_id = ?", courseID).Delete(&Enrollment{}).Error; err != nil {
		rollbackWithError("Failed to delete enrollments")
		return
	}

	if err := tx.Where("id = ?", courseID).Delete(&Course{}).Error; err != nil {
		rollbackWithError("Failed to delete course")
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to finalize course deletion",
		})
		return
	}

	// Best-effort file cleanup after successful DB transaction.
	for _, path := range filePaths {
		_ = os.Remove(path)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Course deleted successfully",
	})
}

func generateCourseCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var code strings.Builder
	code.Grow(length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}

		code.WriteByte(charset[n.Int64()])
	}

	return code.String(), nil
}

func GenerateCourseCode() string {
	code, err := generateCourseCode(6)
	if err != nil {
		// Fallback in rare randomness failures.
		return "ABC123"
	}
	return code
}

func isDuplicateCourseCodeError(err error) bool {
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "duplicate key value") ||
		strings.Contains(errText, "unique constraint")
}
