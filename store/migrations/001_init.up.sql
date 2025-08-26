CREATE TABLE IF NOT EXISTS player (
    steam_id BIGINT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    kills_against INTEGER NOT NULL DEFAULT 0,
    killed_by INTEGER NOT NULL DEFAULT 0,
    created_on INTEGER NOT NULL,
    updated_on INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS match (
    match_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    hostname TEXT NOT NULL,
    address TEXT NOT NULL,
    duration INTEGER NOT NULL DEFAULT 0,
    created_on INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS match_player (
    match_id INTEGER NOT NULL,
    steam_id BIGINT NOT NULL,
    score INTEGER NOT NULL DEFAULT 0,
    deaths INTEGER NOT NULL DEFAULT 0,
    ping INTEGER NOT NULL DEFAULT 0,
    connected INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY(steam_id) REFERENCES player(steam_id),
    FOREIGN KEY(match_id) REFERENCES match(match_id)
);

CREATE TABLE IF NOT EXISTS notes (
    steam_id BIGINT NOT NULL PRIMARY KEY,
    note TEXT NOT NULL,
    updated_on INTEGER NOT NULL,
    FOREIGN KEY(steam_id) REFERENCES player(steam_id)
);

CREATE TABLE IF NOT EXISTS chat_history (
    chat_id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    match_id INTEGER NOT NULL,
    steam_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    message TEXT NOT NULL,
    team_only INTEGER NOT NULL,
    created_on INTEGER NOT NULL,
    FOREIGN KEY(steam_id) REFERENCES player(steam_id),
    FOREIGN KEY(match_id) REFERENCES match(match_id)
);

CREATE TABLE IF NOT EXISTS marks (
    steam_id BIGINT NOT NULL,
    tags TEXT NOT NULL,
    note TEXT NOT NULL,
    created_on INTEGER NOT NULL,
    updated_on INTEGER NOT NULL,
    FOREIGN KEY(steam_id) REFERENCES player(steam_id)
);
