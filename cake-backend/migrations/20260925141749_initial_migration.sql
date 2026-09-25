-- +goose Up
CREATE TABLE users (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash BLOB NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,

    CONSTRAINT PK_users PRIMARY KEY (id)
);

CREATE TABLE conversations (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    owner_id INT NOT NULL,
    title VARCHAR(255),
    created_at DATETIME NOT NULL,

    CONSTRAINT PK_conversations PRIMARY KEY (id),
    CONSTRAINT FK_users_owner FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE roles (
    id INT AUTO_INCREMENT,
    name VARCHAR(32) NOT NULL UNIQUE,

    CONSTRAINT PK_roles PRIMARY KEY (id)
);

CREATE TABLE messages (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    conversation_id INT NOT NULL,
    role_id INT NOT NULL,

    CONSTRAINT PK_messages PRIMARY KEY (id),
    CONSTRAINT FK_conversations_conversation_id FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    CONSTRAINT FK_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(id)
);

INSERT INTO roles (id, name) VALUES (1, 'user');
INSERT INTO roles (id, name) VALUES (2, 'system');
INSERT INTO roles (id, name) VALUES (3, 'assistant');

-- +goose Down
DROP TABLE roles;
DROP TABLE messages;
DROP TABLE conversations;
DROP TABLE users;