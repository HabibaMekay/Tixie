package handlers

import (
	"auth-service/config"
	"auth-service/models"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// makeTestLoginHandler returns a custom /login handler using injected dependencies.
func makeTestLoginHandler(
	mockAuth func(models.Credentials) (bool, error),
	mockJWT func(string) (string, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds models.Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		valid, err := mockAuth(creds)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if !valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := mockJWT(creds.Username)
		if err != nil {
			http.Error(w, "Could not sign token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	}
}

func Test_SuccessfulLogin(t *testing.T) {
	handler := makeTestLoginHandler(
		func(creds models.Credentials) (bool, error) {
			if creds.Username == "validuser" && creds.Password == "pass" {
				return true, nil
			}
			return false, nil
		},
		func(username string) (string, error) {
			return "mocked.jwt.token", nil
		},
	)

	creds := models.Credentials{Username: "validuser", Password: "pass"}
	body, _ := json.Marshal(creds)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var response map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&response)
	assert.Equal(t, "mocked.jwt.token", response["token"])
}

func Test_InvalidJSON(t *testing.T) {
	handler := makeTestLoginHandler(nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer([]byte("{invalid")))
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test_InvalidCredentials(t *testing.T) {
	handler := makeTestLoginHandler(
		func(creds models.Credentials) (bool, error) {
			return false, nil
		},
		func(username string) (string, error) {
			return "", nil
		},
	)

	creds := models.Credentials{Username: "invaliduser", Password: "wrongpass"}
	body, _ := json.Marshal(creds)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func Test_InternalAuthError(t *testing.T) {
	handler := makeTestLoginHandler(
		func(creds models.Credentials) (bool, error) {
			return false, errors.New("auth service down")
		},
		func(username string) (string, error) {
			return "", nil
		},
	)

	creds := models.Credentials{Username: "error", Password: "pass"}
	body, _ := json.Marshal(creds)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func Test_JWTGenerationError(t *testing.T) {
	handler := makeTestLoginHandler(
		func(creds models.Credentials) (bool, error) {
			return true, nil
		},
		func(username string) (string, error) {
			return "", errors.New("jwt signing error")
		},
	)

	creds := models.Credentials{Username: "validuser", Password: "pass"}
	body, _ := json.Marshal(creds)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func Test_OAuth2Login(t *testing.T) {
	config.OAuth2Config = &oauth2.Config{
		ClientID:     "mock-client-id",
		ClientSecret: "mock-client-secret",
		RedirectURL:  "http://localhost:8080/oauth2/callback",
		Scopes:       []string{"profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://provider.com/o/oauth2/auth",
			TokenURL: "https://provider.com/o/oauth2/token",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/oauth2/login", nil)
	rec := httptest.NewRecorder()

	OAuth2Login(rec, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	assert.True(t, strings.HasPrefix(rec.Header().Get("Location"), config.OAuth2Config.Endpoint.AuthURL))
}

func Test_OAuth2Callback_MissingCode(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/oauth2/callback", nil)
	rec := httptest.NewRecorder()

	OAuth2Callback(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test_OAuth2Callback_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://provider.com/o/oauth2/token",
		httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
			"access_token": "fake-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}),
	)

	// httpmock.RegisterResponder("GET", "https://www.googleapis.com/oauth2/v2/userinfo",
	// 	httpmock.NewJsonResponderOrPanic(200, map[string]interface{}{
	// 		"email": "test@example.com",
	// 		"name":  "Test User",
	// 	}),
	// )

	httpmock.RegisterResponder("POST", config.UserServiceURL+"/v1/users",
		httpmock.NewStringResponder(201, ""),
	)

	req := httptest.NewRequest(http.MethodGet, "/oauth2/callback?code=fake-code", nil)
	rec := httptest.NewRecorder()

	OAuth2Callback(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func Test_Protected_MissingToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	Protected(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func Test_Protected_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()

	Protected(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
