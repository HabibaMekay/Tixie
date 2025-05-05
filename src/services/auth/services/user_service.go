package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"auth-service/config"
	"auth-service/models"
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
		return fmt.Errorf("user-service responded with status: %v", resp.Status)
	}
	return nil
}

func AuthenticateUser(creds models.Credentials) (bool, error) {
	data, err := json.Marshal(creds)
	if err != nil {
		return false, err
	}

	resp, err := http.Post(fmt.Sprintf("%s/authenticate", config.UserServiceURL), "application/json", bytes.NewBuffer(data))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}
