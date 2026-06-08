CREATE TABLE authors (
    id    INT8 PRIMARY KEY,
    name  STRING NOT NULL,
    bio   STRING
);

CREATE TABLE books (
    id         INT8 PRIMARY KEY,
    author_id  INT8 NOT NULL,
    title      STRING NOT NULL,
    published  TIMESTAMPTZ
);
