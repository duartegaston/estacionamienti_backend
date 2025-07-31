package service

import (
	"errors"
	"estacionamienti/internal/repository"
	"log"
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
		log.Printf("Error from GetByEmail: %v", err)
		return "", err
	}
	if admin == nil {
		log.Printf("User %s not found", user)
		return "", errors.New("invalid credentials")
	}

	// Comparamos el password hasheado
	err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password))
	if err != nil {
		log.Printf("Error from CompareHashAndPassword: %v", err)
		return "", errors.New("invalid credentials")
	}

	// Creamos un JWT
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Println("JWT_SECRET not set")
		return "", errors.New("JWT_SECRET not set")
	}

	claims := jwt.MapClaims{
		"admin_id": admin.ID,
		"user":     admin.User,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Token expira en 1 hora
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *adminAuthService) CreateAdmin(user, password string) error {
	if user == "" || password == "" {
		log.Println("user and password cannot be empty")
		return errors.New("user and password cannot be empty")
	}

	err := s.repo.CreateNewUser(user, password)
	if err != nil {
		log.Printf("Error from CreateNewUser: %v", err)
		return err
	}

	return nil
}
