package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UpdateEvaluationRequest struct {
	Marks    float64 `json:"marks"`
	Feedback string  `json:"feedback"`
}

func UpdateEvaluationHandler(c *gin.Context) {
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
			"error": "Only professors can update evaluations",
		})
		return
	}

	evaluationIDParam := c.Param("id")
	parsedEvaluationID, err := strconv.ParseUint(evaluationIDParam, 10, 64)
	if err != nil || parsedEvaluationID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid evaluation id",
		})
		return
	}
	evaluationID := uint(parsedEvaluationID)

	var req UpdateEvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.Feedback = strings.TrimSpace(req.Feedback)
	if req.Feedback == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "feedback is required",
		})
		return
	}

	var evaluation Evaluation
	if err := DB.Where("id = ?", evaluationID).First(&evaluation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Evaluation not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch evaluation",
		})
		return
	}

	evaluation.Marks = req.Marks
	evaluation.Feedback = req.Feedback
	evaluation.IsFinal = true

	if err := DB.Save(&evaluation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update evaluation",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Evaluation updated",
	})
}
