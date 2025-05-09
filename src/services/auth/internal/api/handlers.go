package api

import (
	"auth-service/config"
	"auth-service/internal/db/models"
	"auth-service/internal/db/repos"
	"auth-service/internal/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"log"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

var logger *log.Logger

func init() {
	if _, err := os.Stat("logs"); os.IsNotExist(err) {
		os.MkdirAll("logs", os.ModePerm)
	}
	logFile, err := os.OpenFile("logs/service.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	logger = log.New(logFile, "AUTHORIZATION: ", log.LstdFlags|log.Lshortfile)
}

func Login(c *gin.Context) {
	var creds models.Credentials
	if err := c.BindJSON(&creds); err != nil {
		logger.Println("Invalid login request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	valid, err := repos.AuthenticateUser(creds)
	if err != nil {
		logger.Println("Internal error happened during login request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		return
	}
	if !valid {
		logger.Println("Unauthorized login access request")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	token, err := utils.GenerateJWT(creds.Username)
	if err != nil {
		logger.Println("An error in token surfaced")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token error"})
		return
	}
	logger.Println("Login success, a token is being returned")
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func OAuth2Login(c *gin.Context) {
	logger.Println("Google login started")
	url := config.OAuth2Config.AuthCodeURL("", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func OAuth2Callback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		logger.Println("Oauth successful login but no redirection because no fronend lol")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing code"})
		return
	}
	token, err := config.OAuth2Config.Exchange(c.Request.Context(), code)
	if err != nil {
		logger.Println("token exchange from google failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
		return
	}
	client := config.OAuth2Config.Client(c.Request.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		logger.Println("Oauth login failed to get user info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}
	defer resp.Body.Close()
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		logger.Println("Oauth got user info but an error occured after")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Decode error"})
		return
	}
	email := fmt.Sprintf("%v", userInfo["email"])
	name := fmt.Sprintf("%v", userInfo["name"])
	user := models.UserDTO{
		Username: name,
		Email:    email,
		Password: "oauth2_default",
	}
	if err := repos.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	logger.Println("Oauth2 login successful!")
	c.JSON(http.StatusOK, userInfo)
}

func Protected(c *gin.Context) {
	tokenStr := c.GetHeader("Authorization")
	if tokenStr == "" || !strings.HasPrefix(tokenStr, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	claims, err := utils.ParseJWT(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Hello, %s!", claims.Username)})
}
