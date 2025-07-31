CREATE TABLE IF NOT EXISTS notes (
    steam_id BIGINT NOT NULL PRIMARY KEY,
    note TEXT NOT NULL,
    updated_on INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS chat_history (
    chat_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    steam_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    message TEXT NOT NULL,
    team_only INTEGER NOT NULL,
    created_on INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS marks (
    mark_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    steam_id BIGINT NOT NULL,
    tag TEXT NOT NULL,
    note TEXT NOT NULL,
    created_on INTEGER NOT NULL,
    updated_on INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS marks_uidx ON marks (steam_id, tag);