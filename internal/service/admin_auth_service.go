package service

import (
	"errors"
	"estacionamienti/internal/repository"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AdminAuthService interface {
	Login(user, password string) (string, error)
	CreateAdmin(user, password string) error
}

type adminAuthService struct {
	repo repository.AdminAuthRepository
}

func NewAdminAuthService(repo repository.AdminAuthRepository) AdminAuthService {
	return &adminAuthService{repo: repo}
}

func (s *adminAuthService) Login(user, password string) (string, error) {
	admin, err := s.repo.GetByEmail(user)
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", errors.New("invalid credentials")
	}

	// Comparamos el password hasheado
	err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	// Creamos un JWT
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", errors.New("JWT_SECRET not set")
	}

	claims := jwt.MapClaims{
		"admin_id": admin.ID,
		"user":     admin.User,
		"exp":      time.Now().Add(time.Hour * 1).Unix(), // Token expira en 1 hora
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *adminAuthService) CreateAdmin(user, password string) error {
	if user == "" || password == "" {
		return errors.New("user and password cannot be empty")
	}

	err := s.repo.CreateNewUser(user, password)
	if err != nil {
		return err
	}

	return nil
}
