package v1

import (
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
)

// GithubOAuth handles GitHub OAuth login
func GithubOAuth(c *gin.Context) {
	response.Success(c, gin.H{"message": "GitHub OAuth login"})
}

// GithubOAuthCallback handles GitHub OAuth callback
func GithubOAuthCallback(c *gin.Context) {
	response.Success(c, gin.H{"message": "GitHub OAuth callback"})
}
