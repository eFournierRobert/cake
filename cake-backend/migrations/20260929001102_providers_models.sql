-- +goose Up
CREATE TABLE providers (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    base_url VARCHAR(512) NOT NULL,
    api_key BLOB,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,

    CONSTRAINT PK_providers PRIMARY KEY (id)
);

CREATE TABLE models (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    context_length INT UNSIGNED NOT NULL,
    provider_model_id VARCHAR(255) NOT NULL,
    provider_id INT NOT NULL,
    created_at DATETIME NOT NULL,
    activated boolean NOT NULL DEFAULT FALSE,

    CONSTRAINT PK_models PRIMARY KEY (id),
    CONSTRAINT FK_providers_provider_id FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    CONSTRAINT UNIQUE_models_providers_model_id UNIQUE (provider_id, provider_model_id)
);

-- +goose Down
DROP TABLE models;
DROP TABLE providers;
