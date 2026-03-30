package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func SignupHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	req.Role = strings.TrimSpace(strings.ToLower(req.Role))

	if req.Name == "" || req.Email == "" || req.Password == "" || req.Role == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "name, email, password, and role are required",
		})
		return
	}

	if req.Role != "student" && req.Role != "professor" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "role must be either 'student' or 'professor'",
		})
		return
	}

	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process password",
		})
		return
	}

	user := User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
		Role:     req.Role,
	}

	if err := DB.Create(&user).Error; err != nil {
		if isDuplicateEmailError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create user",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Signup successful",
	})
}

func LoginHandler(c *gin.Context) {
	if DB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Database is not initialized",
		})
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email and password are required",
		})
		return
	}

	var user User
	err := DB.Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid credentials",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch user",
		})
		return
	}

	if !CheckPasswordHash(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	token, err := GenerateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func isDuplicateEmailError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "duplicate key value") ||
		strings.Contains(errText, "unique constraint")
}
