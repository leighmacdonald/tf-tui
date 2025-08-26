-- name: InsertPlayer :exec
INSERT INTO player (steam_id, name, created_on, updated_on)
VALUES (sqlc.arg(steam_id), sqlc.arg(name), sqlc.arg(created_on), sqlc.arg(updated_on))
ON CONFLICT (steam_id) DO UPDATE
    SET name       = sqlc.arg(name),
        updated_on = sqlc.arg(updated_on);

-- name: GetNotes :many
SELECT *
FROM notes
WHERE steam_id IN (sqlc.slice(steam_ids));

-- name: InsertNote :exec
INSERT INTO notes (steam_id, note, updated_on)
VALUES (?, ?, ?);

-- name: UpdateNote :exec
UPDATE notes
SET note       = ?,
    updated_on = ?
WHERE steam_id = ?;

-- name: InsertChat :exec
INSERT INTO chat_history (chat_id, steam_id, name, message, team_only, created_on)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetChatHistory :many
SELECT *
FROM chat_history
WHERE steam_id = ?;

-- name: InsertMark :exec
INSERT INTO marks (steam_id, tags, note, created_on, updated_on)
VALUES (?, ?, ?, ?, ?);
