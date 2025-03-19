CREATE TABLE
    IF NOT EXISTS `tunnels` (
        id TEXT PRIMARY KEY,
        hashedToken TEXT,
        expiresAt TEXT,
        createdAt TEXT,
        serverHost TEXT,
        metadata TEXT DEFAULT '{}'
    ) STRICT;

PRAGMA journal_mode = WAL;

PRAGMA foreign_keys = ON;

PRAGMA synchronous = NORMAL;