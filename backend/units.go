package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CreateCourseUnitRequest struct {
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	UnitOrder   int     `json:"unit_order"`
	Description string  `json:"description"`
}

type CourseUnitResponse struct {
	ID          uint    `json:"id"`
	CourseID    uint    `json:"course_id"`
	Name        string  `json:"name"`
	Weight      float64 `json:"weight"`
	UnitOrder   int     `json:"unit_order"`
	Description string  `json:"description"`
}

func CreateCourseUnitHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseID, _, ok := requireProfessorOwnsCourse(c)
	if !ok {
		return
	}

	var req CreateCourseUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	if req.Weight <= 0 {
		req.Weight = 1
	}
	if req.UnitOrder < 0 {
		req.UnitOrder = 0
	}

	unit := CourseUnit{
		CourseID:    courseID,
		Name:        req.Name,
		Weight:      req.Weight,
		UnitOrder:   req.UnitOrder,
		Description: req.Description,
	}
	if err := DB.Create(&unit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create unit"})
		return
	}

	c.JSON(http.StatusCreated, CourseUnitResponse{
		ID:          unit.ID,
		CourseID:    unit.CourseID,
		Name:        unit.Name,
		Weight:      unit.Weight,
		UnitOrder:   unit.UnitOrder,
		Description: unit.Description,
	})
}

func ListCourseUnitsHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database is not initialized"})
		return
	}

	courseIDParam := strings.TrimSpace(c.Param("id"))
	parsedCourseID, err := strconv.ParseUint(courseIDParam, 10, 64)
	if err != nil || parsedCourseID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid course id"})
		return
	}
	courseID := uint(parsedCourseID)

	if _, _, _, ok := requireCourseAccessByID(c, courseID); !ok {
		return
	}

	var units []CourseUnit
	if err := DB.Where("course_id = ?", courseID).Order("unit_order ASC, id ASC").Find(&units).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch units"})
		return
	}

	response := make([]CourseUnitResponse, 0, len(units))
	for _, unit := range units {
		response = append(response, CourseUnitResponse{
			ID:          unit.ID,
			CourseID:    unit.CourseID,
			Name:        unit.Name,
			Weight:      unit.Weight,
			UnitOrder:   unit.UnitOrder,
			Description: unit.Description,
		})
	}

	c.JSON(http.StatusOK, response)
}

func ensureAssignmentUnitBelongsToCourse(tx *gorm.DB, courseID uint, unitID *uint) error {
	if unitID == nil || *unitID == 0 {
		return nil
	}
	var unit CourseUnit
	if err := tx.Where("id = ? AND course_id = ?", *unitID, courseID).First(&unit).Error; err != nil {
		return err
	}
	return nil
}

