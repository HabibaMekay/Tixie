package repos

import (
    "database/sql"
    "user-service/internal/db"
    "user-service/internal/db/models"
)

func CreateUser(user models.User) error {
    query := `INSERT INTO users (username, email, password) VALUES ($1, $2, $3)`
    _, err := db.DB.Exec(query, user.Username, user.Email, user.Password)
    return err

}

func GetAllUsers() ([]models.User, error) {
    query := `SELECT id, username, email, password FROM users`
    rows, err := db.DB.Query(query)
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

func GetUserByID(id int) (models.User, error) {
    query := `SELECT id, username, email, password FROM users WHERE id = $1`
    var user models.User
    err := db.DB.QueryRow(query, id).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
    if err != nil {
        if err == sql.ErrNoRows {
            return user, nil 
        }
        return user, err
    }
    return user, nil
}

func UpdateUser(id int, updatedUser models.User) error {
    query := `UPDATE users SET username = $1, email = $2, password = $3 WHERE id = $4`
    _, err := db.DB.Exec(query, updatedUser.Username, updatedUser.Email, updatedUser.Password, id)
    return err
}

func DeleteUser(id int) error {
    query := `DELETE FROM users WHERE id = $1`
    _, err := db.DB.Exec(query, id)
    return err
}



