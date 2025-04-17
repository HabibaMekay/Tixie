package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type UserDTO struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func createUserInUserService(user UserDTO) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	resp, err := http.Post("http://user-service:8081/users", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("user-service responded with status: %v", resp.Status)
	}
	return nil
}

func authenticateUser(creds Credentials) (bool, error) {
	data, err := json.Marshal(creds)
	if err != nil {
		return false, err
	}

	resp, err := http.Post("http://user-service:8081/authenticate", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

var jwtKey []byte
var oauth2Config *oauth2.Config // Declare it globally

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func init() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	// assigning secret key to jwtKey
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		fmt.Println("JWT_SECRET is not set in the environment")
		os.Exit(1)
	}
	jwtKey = []byte(jwtSecret)

	// Initialize the OAuth2 config AFTER loading env variables
	oauth2Config = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}

	fmt.Println("Loaded client ID:", oauth2Config.ClientID)
}

func login(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Call user-service to validate credentials
	authValid, err := authenticateUser(creds)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if !authValid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Create JWT token
	expiration := time.Now().Add(1 * time.Hour)
	claims := &Claims{
		Username: creds.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Could not sign token", http.StatusInternalServerError)
		return
	}

	// Return the token as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": signed,
	})
}

func oauth2Login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("==> /oauth2-login handler called")
	url := oauth2Config.AuthCodeURL("", oauth2.AccessTypeOffline)
	fmt.Println("OAuth2 URL:", url)
	http.Redirect(w, r, url, http.StatusFound)
}

func oauth2Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	token, err := oauth2Config.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange code for token", http.StatusInternalServerError)
		return
	}

	client := oauth2Config.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "Failed to decode user info", http.StatusInternalServerError)
		return
	}

	email := fmt.Sprintf("%v", userInfo["email"])
	name := fmt.Sprintf("%v", userInfo["name"])

	userDTO := UserDTO{
		Username: name,
		Email:    email,
		Password: "oauth2_default", // This should probably be handled differently, maybe leave blank if no password.
	}
	err = createUserInUserService(userDTO)
	if err != nil {
		http.Error(w, "Failed to create user in user-service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

func protected(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	tokenStr := authHeader[len("Bearer "):]
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(w, "Hello, %s! You accessed a protected route.\n", claims.Username)
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/login", login).Methods("POST")
	r.HandleFunc("/oauth2-login", oauth2Login).Methods("GET")
	r.HandleFunc("/callback", oauth2Callback).Methods("GET")
	r.HandleFunc("/protected", protected).Methods("GET")

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}
