package config

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var JWTKey []byte
var OAuth2Config *oauth2.Config
var UserServiceURL string

func LoadEnv() {
	jwtSecret := os.Getenv("JWT_SECRET")
	UserServiceURL = os.Getenv("USER_SERVICE_URL")

	if jwtSecret == "" {
		fmt.Println("JWT_SECRET is not set")
		os.Exit(1)
	}
	JWTKey = []byte(jwtSecret)

	OAuth2Config = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
		Endpoint:     google.Endpoint,
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	}
}
