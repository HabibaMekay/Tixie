package api

import (
    "encoding/json"
    "net/http"
	"strconv"
    "github.com/gorilla/mux"
    "github.com/lib/pq"
    "user-service/internal/db/models"
    "user-service/internal/db/repos"
    "golang.org/x/crypto/bcrypt"
)

func hashPassword(password string) (string, error) {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(hashedPassword), nil
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
    var user models.User
    err := json.NewDecoder(r.Body).Decode(&user)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Hash the password before saving the user
    hashedPassword, err := hashPassword(user.Password)
    if err != nil {
        http.Error(w, "Failed to hash password", http.StatusInternalServerError)
        return
    }
    user.Password = hashedPassword

    err = repos.CreateUser(user)
    if err != nil {
        if pgErr, ok := err.(*pq.Error); ok {
            if pgErr.Code == "23505" { // unique violation
                http.Error(w, "Username or Email already exists", http.StatusConflict)
                return
            }
        }

        http.Error(w, "Failed to create user", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
}



func GetUsers(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")

    users, err := repos.GetAllUsers()
    if err != nil {
        http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(users)
}

func GetUserByID(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    id, err := strconv.Atoi(params["id"])
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    user, err := repos.GetUserByID(id)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    id, err := strconv.Atoi(params["id"])
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    var updatedUser models.User
    if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Hash 
    if updatedUser.Password != "" {
        hashedPassword, err := hashPassword(updatedUser.Password)
        if err != nil {
            http.Error(w, "Failed to hash password", http.StatusInternalServerError)
            return
        }
        updatedUser.Password = hashedPassword
    }

    err = repos.UpdateUser(id, updatedUser)
    if err != nil {
        if pgErr, ok := err.(*pq.Error); ok {
            if pgErr.Code == "23505" { // unique violation
                http.Error(w, "Username or Email already exists", http.StatusConflict)
                return
            }
        }
    
        http.Error(w, "Failed to update user", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}


func DeleteUser(w http.ResponseWriter, r *http.Request) {
    params := mux.Vars(r)
    id, err := strconv.Atoi(params["id"])
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    err = repos.DeleteUser(id)
    if err != nil {
        http.Error(w, "Failed to delete user", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}


func AuthenticateUser(w http.ResponseWriter, r *http.Request) {
    var creds models.Credentials
    if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Check if the username and password match
    valid, err := repos.CheckCredentials(creds.Username, creds.Password)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    if !valid {
        http.Error(w, "Invalid username or password", http.StatusUnauthorized)
        return
    }

    w.WriteHeader(http.StatusOK)
}



