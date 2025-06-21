package repository

import (
	"database/sql"
)

type StripeRepository struct {
	DB *sql.DB
}

func NewStripeRepository(db *sql.DB) *StripeRepository {
	return &StripeRepository{DB: db}
}
