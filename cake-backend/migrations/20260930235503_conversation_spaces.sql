-- +goose Up
CREATE TABLE spaces (
    id INT AUTO_INCREMENT,
    uuid CHAR(36) NOT NULL,
    name VARCHAR(255) NOT NULL,
    system_prompt TEXT,
    created_at DATETIME NOT NULL,
    owner_id INT NOT NULL,

    CONSTRAINT PK_spaces PRIMARY KEY (id),
    CONSTRAINT FK_users_owner_id FOREIGN KEY (owner_id) REFERENCES users(id)
);

ALTER TABLE conversations ADD space_id INT NOT NULL;
ALTER TABLE conversations ADD CONSTRAINT FK_spaces_space_id FOREIGN KEY (space_id) REFERENCES spaces(id);

-- +goose Down
ALTER TABLE conversations DROP space_id;
DROP TABLE spaces;
