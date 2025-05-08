package repository

import (
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type Admin struct {
	ID           int
	Email        string
	PasswordHash string
}

type AdminAuthRepository interface {
	GetByEmail(email string) (*Admin, error)
	CreateNewUser(email, password string) error
}

type adminAuthRepository struct {
	db *sql.DB
}

func NewAdminAuthRepository(db *sql.DB) AdminAuthRepository {
	return &adminAuthRepository{db: db}
}

func (r *adminAuthRepository) GetByEmail(email string) (*Admin, error) {
	var admin Admin
	err := r.db.QueryRow("SELECT email, password_hash FROM admins WHERE email = $1", email).
		Scan(&admin.Email, &admin.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

func (r *adminAuthRepository) CreateNewUser(email, password string) error {
	// Hashear la contraseña usando bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// Insertar el admin con el password hasheado
	query := "INSERT INTO admins (email, password_hash) VALUES ($1, $2)"
	_, err = r.db.Exec(query, email, hashedPassword)
	if err != nil {
		return err
	}

	return nil
}
