
-- name: GetConnection :one
Select * from connections where uid = $1;