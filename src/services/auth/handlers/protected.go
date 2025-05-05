package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"auth-service/utils"
)

func Protected(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := utils.ParseJWT(tokenStr)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	fmt.Fprintf(w, "Hello, %s! You accessed a protected route.\n", claims.Username)
}
