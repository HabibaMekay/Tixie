package repos

import (
	"auth-service/config"
	"auth-service/internal/db/models"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func CreateUser(user models.UserDTO) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	resp, err := http.Post(config.UserServiceURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("user service returned: %s", resp.Status)
	}
	return nil
}

func AuthenticateUser(creds models.Credentials) (bool, error) {
	data, err := json.Marshal(creds)
	if err != nil {
		return false, err
	}

	// Determine which endpoint to call based on role
	role := "user" // Default role

	if creds.Role != "" {
		role = creds.Role
	}

	var endpoint string
	if role == "vendor" {
		endpoint = fmt.Sprintf("%s/vendor/v1/authenticate", config.VendorServiceURL)
	} else {
		endpoint = fmt.Sprintf("%s/user/v1/authenticate", config.UserServiceURL)
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}
