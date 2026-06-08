CREATE TABLE users (
    id    INT8 PRIMARY KEY,
    name  STRING NOT NULL,
    email STRING,
    INDEX users_name_idx (name)
);
