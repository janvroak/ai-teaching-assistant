package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateAssignmentRequest struct {
	CourseID  uint   `json:"course_id"`
	Title     string `json:"title"`
	Question  string `json:"question"`
	AnswerKey string `json:"answer_key"`
}

type AssignmentListItem struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	Question string `json:"question"`
}

func CreateAssignmentHandler(c *gin.Context) {
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
			"error": "Only professors can create assignments",
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

	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Question = strings.TrimSpace(req.Question)
	req.AnswerKey = strings.TrimSpace(req.AnswerKey)
	if req.CourseID == 0 || req.Title == "" || req.Question == "" || req.AnswerKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "course_id, title, question, and answer_key are required",
		})
		return
	}

	var course Course
	err := DB.Where("id = ? AND professor_id = ?", req.CourseID, professorID).First(&course).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "You can only create assignments for your own courses",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to verify course ownership",
		})
		return
	}

	assignment := Assignment{
		CourseID:  req.CourseID,
		Title:     req.Title,
		Question:  req.Question,
		AnswerKey: req.AnswerKey,
	}

	if err := DB.Create(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create assignment",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":        assignment.ID,
		"title":     assignment.Title,
		"question":  assignment.Question,
		"course_id": assignment.CourseID,
	})
}

func ListCourseAssignmentsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
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

	switch role {
	case "professor":
		var course Course
		err := DB.Where("id = ? AND professor_id = ?", courseID, userID).First(&course).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "You can only view assignments for your own courses",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify course ownership",
			})
			return
		}
	case "student":
		var enrollment Enrollment
		err := DB.Where("student_id = ? AND course_id = ?", userID, courseID).First(&enrollment).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "You are not enrolled in this course",
				})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to verify enrollment",
			})
			return
		}
	default:
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid role",
		})
		return
	}

	var assignments []AssignmentListItem
	if err := DB.Model(&Assignment{}).
		Where("course_id = ?", courseID).
		Select("id", "title", "question").
		Order("id ASC").
		Find(&assignments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch assignments",
		})
		return
	}

	c.JSON(http.StatusOK, assignments)
}
