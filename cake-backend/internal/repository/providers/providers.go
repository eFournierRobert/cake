// Package providers provides a repository for the providers table.
package providers

import (
	"database/sql"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

// Repository implements CRUD operations on the providers table.
type Repository struct {
	conn *sqlx.DB
}

// New creates a new Provider repository with the given database connection.
func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

// GetAllProviders returns all providers, newest first.
func (r *Repository) GetAllProviders() ([]Provider, error) {
	var providers []Provider
	if err := r.conn.Select(&providers,
		`SELECT
    			id,
    			uuid,
    			name,
    			base_url,
    			api_key,
    			created_at, 
    			updated_at
			FROM providers
			ORDER BY created_at DESC`); err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return providers, nil
}

// GetProvider returns the provider with the given uuid, or *repo_errors.ProviderNotFound.
func (r *Repository) GetProvider(uuid uuid.UUID) (Provider, error) {
	var provider Provider
	err := r.conn.Get(&provider,
		`SELECT * FROM providers WHERE uuid = ?`,
		uuid.String())
	if errors.Is(err, sql.ErrNoRows) {
		return provider, &repo_errors.ProviderNotFound{}
	}
	if err != nil {
		return provider, &repo_errors.InternalDbError{Err: err}
	}

	return provider, nil
}

// CreateProvider inserts a provider, filling in the timestamps, and
// returns the created provider as stored in the database, including
// its auto-generated id.
func (r *Repository) CreateProvider(provider Provider) (Provider, error) {
	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec(
		`INSERT INTO providers (
                       uuid, 
                       name, 
                       base_url, 
                       api_key, 
                       created_at, 
                       updated_at) 
				VALUES (
				        :uuid, 
				        :name,
				        :base_url,
				        :api_key,
				        :created_at,
				        :updated_at
				)`, provider)
	if err != nil {
		return Provider{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetProvider(uuid.MustParse(provider.Uuid))
}

// UpdateProvider updates a provider by uuid and returns the updated
// provider as stored in the database, or *repo_errors.ProviderNotFound
// if no provider matches. Pass the full row (e.g. from GetProvider) so
// fields you do not touch are written back unchanged.
func (r *Repository) UpdateProvider(provider Provider) (Provider, error) {
	provider.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec(
		`UPDATE providers SET
                     name = :name,
                     base_url = :base_url,
                     api_key = :api_key,
                     updated_at = :updated_at
                     WHERE uuid = :uuid`,
		provider)
	if err != nil {
		return Provider{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetProvider(uuid.MustParse(provider.Uuid))
}

// DeleteProvider removes the provider with the given uuid and, by
// cascading, the model rows that reference it. It returns
// *repo_errors.ProviderNotFound if no provider matches.
func (r *Repository) DeleteProvider(uuid uuid.UUID) error {
	results, err := r.conn.Exec(
		`DELETE FROM providers WHERE uuid = ?`,
		uuid.String())
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	if rowsAffected == 0 {
		return &repo_errors.ProviderNotFound{}
	}

	return nil
}
