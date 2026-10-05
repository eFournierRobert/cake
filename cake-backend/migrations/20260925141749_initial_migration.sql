-- +goose Up
CREATE TABLE user_roles (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    name VARCHAR(32) NOT NULL UNIQUE,

    CONSTRAINT PK_user_roles PRIMARY KEY (id)
);

CREATE TABLE users (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash BLOB NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    created_at DATETIME(6) NOT NULL,
    updated_at DATETIME(6) NOT NULL,
    role_id INT NOT NULL,

    CONSTRAINT PK_users PRIMARY KEY (id),
    CONSTRAINT FK_user_role_role_id FOREIGN KEY (role_id) REFERENCES user_roles(id)
);

CREATE TABLE conversations (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    owner_id INT NOT NULL,
    title VARCHAR(255),
    created_at DATETIME(6) NOT NULL,

    CONSTRAINT PK_conversations PRIMARY KEY (id),
    CONSTRAINT FK_users_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE message_roles (
    id INT AUTO_INCREMENT,
    name VARCHAR(32) NOT NULL UNIQUE,

    CONSTRAINT PK_message_roles PRIMARY KEY (id)
);

CREATE TABLE messages (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL UNIQUE,
    content TEXT NOT NULL,
    created_at DATETIME(6) NOT NULL,
    conversation_id INT NOT NULL,
    role_id INT NOT NULL,

    CONSTRAINT PK_messages PRIMARY KEY (id),
    CONSTRAINT FK_conversations_conversation_id FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    CONSTRAINT FK_message_roles_role_id FOREIGN KEY (role_id) REFERENCES message_roles(id)
);

INSERT INTO message_roles (name) VALUES
                                     ('user'),
                                     ('system'),
                                     ('assistant');

INSERT INTO user_roles (uuid, name) VALUES
                                        ('5a18559d-9d20-4251-8a72-b36efbaff514', 'admin'),
                                        ('15afe83a-fd92-4d66-8f22-5a3bedbb53e1', 'user');

-- +goose Down
DROP TABLE messages;
DROP TABLE message_roles;
DROP TABLE conversations;
DROP TABLE users;
DROP TABLE user_roles;