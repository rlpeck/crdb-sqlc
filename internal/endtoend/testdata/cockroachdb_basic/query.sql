-- name: GetAuthor :one
SELECT id, name, bio FROM authors WHERE id = $1;

-- name: ListAuthors :many
SELECT * FROM authors;

-- name: CreateAuthor :one
INSERT INTO authors (name, bio) VALUES (sqlc.arg(name), sqlc.arg(bio)) RETURNING id, name, bio;

-- name: DeleteAuthor :exec
DELETE FROM authors WHERE id = $1;

-- name: BooksByAuthor :many
SELECT b.id, b.title, a.name
FROM books b
JOIN authors a ON a.id = b.author_id
WHERE a.id = sqlc.arg(author_id)
LIMIT $1;

-- name: CountBooks :one
SELECT count(*) FROM books WHERE author_id = $1;

-- name: GetAuthorByName :one
SELECT id, name, bio FROM authors WHERE name = @name;
