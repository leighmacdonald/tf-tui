-- name: GetNotes :many
SELECT * FROM notes
WHERE steam_id IN (sqlc.slice(steam_ids));

-- name: InsertNote :exec
INSERT INTO notes (steam_id, note, updated_on) VALUES (?, ?, ?);

-- name: UpdateNote :exec
UPDATE notes SET note = ?,  updated_on = ? WHERE steam_id = ?;
