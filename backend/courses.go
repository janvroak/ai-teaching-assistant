package main

import (
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
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
