-- PostgreSQL schema for the integrations server
CREATE TYPE integration_status AS ENUM ('pending', 'active', 'needs_reauth', 'disabled');

CREATE TABLE
    connections (
        uid text,
        connection_id text, -- The Nango connection ID
        status integration_status,
        error text,
        updated_at timestamp
        with
            time zone,
            created_at timestamp
        with
            time zone,
            metadata jsonb,
            PRIMARY KEY (uid, connection_id)
    );

CREATE TABLE
    profile (
        profile_id text PRIMARY KEY,
        profile_name text,
        profile_description text,
        user_config jsonb,
        created_at timestamp
        with
            time zone,
            updated_at timestamp
        with
            time zone,
            metadata jsonb,
            connection_id text REFERENCES connections (connection_id)
    );