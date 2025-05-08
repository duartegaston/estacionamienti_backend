package api

import (
	"encoding/json"
	"estacionamienti/internal/service"
	"net/http"
)

type AdminAuthHandler struct {
	service service.AdminAuthService
}

func NewAdminAuthHandler(svc service.AdminAuthService) *AdminAuthHandler {
	return &AdminAuthHandler{service: svc}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func (h *AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	resp := LoginResponse{Token: token}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *AdminAuthHandler) CreateUserAdmin(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Decodificamos el cuerpo de la solicitud
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Llamamos al servicio para registrar el admin
	err = h.service.CreateAdmin(request.Email, request.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Admin registered successfully"))
}
