package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type EvaluateRequest struct {
	Question        string `json:"question"`
	StudentAnswer   string `json:"student_answer"`
	ReferenceAnswer string `json:"reference_answer"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found, using existing environment variables")
	}

	DB = InitDB()

	// Create a Gin router with default middleware (logger and recovery).
	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	router.POST("/signup", SignupHandler)
	router.POST("/login", LoginHandler)
	router.POST("/courses", AuthMiddleware(), CreateCourseHandler)
	router.DELETE("/courses/:id", AuthMiddleware(), DeleteCourseHandler)
	router.POST("/courses/join", AuthMiddleware(), JoinCourseHandler)
	router.GET("/courses", AuthMiddleware(), ListCoursesHandler)
	router.POST("/courses/:id/materials", AuthMiddleware(), CreateCourseMaterialHandler)
	router.GET("/courses/:id/materials", AuthMiddleware(), ListCourseMaterialsHandler)
	router.POST("/courses/:id/doubt", AuthMiddleware(), AskCourseDoubtHandler)
	router.GET("/materials/:id/file", AuthMiddleware(), GetCourseMaterialFileHandler)
	router.GET("/courses/:id/assignments", AuthMiddleware(), ListCourseAssignmentsHandler)
	router.GET("/assignments/:id/submissions", AuthMiddleware(), GetSubmissionsHandler)
	router.GET("/assignments/:id/question-file", AuthMiddleware(), GetAssignmentQuestionPDFHandler)
	router.GET("/assignments/:id/answer-key-file", AuthMiddleware(), GetAssignmentAnswerKeyPDFHandler)
	router.POST("/assignments", AuthMiddleware(), CreateAssignmentHandler)
	router.POST("/submit", AuthMiddleware(), SubmitHandler)
	router.POST("/submit-file", AuthMiddleware(), SubmitFileHandler)
	router.GET("/my-submissions", AuthMiddleware(), GetMySubmissionsHandler)
	router.GET("/submissions/:id", AuthMiddleware(), GetSubmissionHandler)
	router.GET("/submissions/:id/file", AuthMiddleware(), GetSubmissionFileHandler)
	router.PUT("/evaluations/:id", AuthMiddleware(), UpdateEvaluationHandler)

	router.GET("/profile", AuthMiddleware(), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"role":    role,
		})
	})

	protected := router.Group("/protected")
	protected.Use(AuthMiddleware())
	protected.GET("/me", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		role, _ := c.Get("role")

		c.JSON(http.StatusOK, gin.H{
			"user_id": userID,
			"role":    role,
		})
	})

	// GET / -> returns a simple health message.
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Backend is running",
		})
	})

	// POST /evaluate -> forwards request to FastAPI and returns its response.
	router.POST("/evaluate", func(c *gin.Context) {
		var reqBody EvaluateRequest
		if err := c.ShouldBindJSON(&reqBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request body",
			})
			return
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to prepare request for AI service",
			})
			return
		}

		fastAPIResponse, err := http.Post(
			fmt.Sprintf("%s/evaluate", aiServiceBaseURL()),
			"application/json",
			bytes.NewBuffer(jsonBody),
		)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error": "Could not connect to AI service",
			})
			return
		}
		defer fastAPIResponse.Body.Close()

		responseBytes, err := io.ReadAll(fastAPIResponse.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to read AI service response",
			})
			return
		}

		// Return FastAPI response as-is so status and error details are preserved.
		c.Data(fastAPIResponse.StatusCode, "application/json", responseBytes)
	})

	// Start server on port 5000.
	router.Run(":5000")
}
