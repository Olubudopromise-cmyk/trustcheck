package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type verifyRequest struct {
	Input string `json:"input"`
}

type verifyResponse struct {
	Input      string `json:"input"`
	Type       string `json:"type"`
	Status     string `json:"status"`
	TrustScore int    `json:"trustScore"`
	Summary    string `json:"summary"`
}

// corsMiddleware enables cross-origin requests from the allowed frontend
// origin only. It answers CORS preflight (OPTIONS) requests and injects the
// Access-Control-Allow-Origin header on actual responses. No third-party
// packages are required.
func corsMiddleware() gin.HandlerFunc {
	allowedOrigins := map[string]bool{
		os.Getenv("ALLOWED_ORIGIN"): true,
	}
	if allowedOrigins[""] {
		allowedOrigins["http://localhost:3000"] = true
	}
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func main() {
	r := gin.Default()
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "trustcheck-api",
		})
	})

	r.POST("/verify", func(c *gin.Context) {
		var req verifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: input is required"})
			return
		}
		if req.Input == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "input is required"})
			return
		}

		resp := verifyResponse{
			Input:      req.Input,
			Type:       "website",
			Status:     "verified",
			TrustScore: 100,
			Summary:    "Verification engine is not connected yet.",
		}
		c.JSON(http.StatusOK, resp)
	})

	r.Run(":8080")
}
