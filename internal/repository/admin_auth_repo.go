package repository

import (
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type Admin struct {
	ID           int
	User         string
	PasswordHash string
}

type AdminAuthRepository interface {
	GetByEmail(user string) (*Admin, error)
	CreateNewUser(user, password string) error
}

type adminAuthRepository struct {
	db *sql.DB
}

func NewAdminAuthRepository(db *sql.DB) AdminAuthRepository {
	return &adminAuthRepository{db: db}
}

func (r *adminAuthRepository) GetByEmail(email string) (*Admin, error) {
	var admin Admin
	err := r.db.QueryRow("SELECT user_name, password_hash FROM admins WHERE user_name = $1", email).
		Scan(&admin.User, &admin.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

func (r *adminAuthRepository) CreateNewUser(user, password string) error {
	// Hashear la contraseña usando bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := "INSERT INTO admins (user_name, password_hash) VALUES ($1, $2)"
	_, err = r.db.Exec(query, user, hashedPassword)
	if err != nil {
		return err
	}

	return nil
}
