package repos

import (
	"database/sql"
	"errors"
	"user-service/internal/db/models"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) CreateUser(user models.User) error {
	query := `INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`
	_, err := r.DB.Exec(query, user.Username, user.Email, user.Password)
	return err

}

func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	query := `SELECT id, username, email, password FROM users`
	rows, err := r.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Password)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *UserRepository) GetUserByID(id int) (models.User, error) {
	query := `SELECT id, username, email, password FROM users WHERE id = $1`
	var user models.User
	err := r.DB.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, nil
		}
		return user, err
	}
	return user, nil
}

func (r *UserRepository) UpdateUser(id int, updatedUser models.User) error {
	query := `UPDATE users SET username = $1, email = $2, password = $3 WHERE id = $4`
	_, err := r.DB.Exec(query, updatedUser.Username, updatedUser.Email, updatedUser.Password, id)
	return err
}

func (r *UserRepository) DeleteUser(id int) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.DB.Exec(query, id)
	return err
}

func (r *UserRepository) CheckCredentials(username, password string) (bool, error) {
	var storedPassword string

	query := `SELECT password FROM users WHERE username = $1`
	err := r.DB.QueryRow(query, username).Scan(&storedPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // No user found
		}
		return false, err // Database error
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		return false, nil //  don't match
	}

	return true, nil //  match
}
