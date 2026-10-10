package models

import (
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		conn: db,
	}
}

func (r *Repository) GetAllModels() ([]Model, error) {
	var models []Model
	if err := r.conn.Select(&models,
		`SELECT *
				FROM models
				ORDER BY created_at DESC`); err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return models, nil
}

func (r *Repository) GetAllActivatedModels() ([]Model, error) {
	var models []Model
	if err := r.conn.Select(&models,
		`SELECT *
				FROM models
				WHERE activated = TRUE
				ORDER BY created_at DESC`); err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return models, nil
}

func (r *Repository) CreateModel(model Model) error {
	model.CreatedAt = time.Now()

	_, err := r.conn.NamedExec(
		`INSERT INTO models (
                    uuid, 
                    name, 
                    description, 
                    context_length, 
                    provider_model_id, 
                    provider_id, 
                    created_at) 
                    VALUES (
                            :uuid,
                            :name,
                            :description,
                            :context_length,
                            :provider_model_id,
                            :provider_id,
                            :created_at
                    )`, model)
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}

func (r *Repository) UpdateModel(model Model) error {
	_, err := r.conn.NamedExec(
		`UPDATE models SET
                  name = :name,
                  description = :description,
                  provider_model_id = :provider_model_id,
                  activated = :activated
                  WHERE uuid = :uuid`, model)
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}

func (r *Repository) DeleteModel(uuid uuid.UUID) error {
	_, err := r.conn.Exec(
		`DELETE FROM models WHERE uuid = ?`,
		uuid.String())
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}
