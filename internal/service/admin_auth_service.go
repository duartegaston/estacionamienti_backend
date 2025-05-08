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
	Login(email, password string) (string, error)
	CreateAdmin(email, password string) error
}

type adminAuthService struct {
	repo repository.AdminAuthRepository
}

func NewAdminAuthService(repo repository.AdminAuthRepository) AdminAuthService {
	return &adminAuthService{repo: repo}
}

func (s *adminAuthService) Login(email, password string) (string, error) {
	admin, err := s.repo.GetByEmail(email)
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
		"email":    admin.Email,
		"exp":      time.Now().Add(time.Hour * 1).Unix(), // Token expira en 1 hora
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (s *adminAuthService) CreateAdmin(email, password string) error {
	// Validación simple del email y la contraseña
	if email == "" || password == "" {
		return errors.New("email and password cannot be empty")
	}

	// Intentamos insertar el admin en la base de datos
	err := s.repo.CreateNewUser(email, password)
	if err != nil {
		return err
	}

	return nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
