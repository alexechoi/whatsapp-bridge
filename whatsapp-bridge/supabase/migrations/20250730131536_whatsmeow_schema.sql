-- Complete WhatsApp Bridge Schema Migration
-- This creates the exact schema that whatsmeow expects based on actual SQLite databases
-- Starting from scratch - assumes no pre-existing migrations

-- Create whatsmeow version table
CREATE TABLE whatsmeow_version (
    version INTEGER, 
    compat INTEGER
);

-- Create device table with all required columns
CREATE TABLE whatsmeow_device (
    jid TEXT PRIMARY KEY,
    lid TEXT,
    facebook_uuid UUID,
    registration_id BIGINT NOT NULL CHECK (registration_id >= 0 AND registration_id < 4294967296),
    noise_key BYTEA NOT NULL CHECK (length(noise_key) = 32),
    identity_key BYTEA NOT NULL CHECK (length(identity_key) = 32),
    signed_pre_key BYTEA NOT NULL CHECK (length(signed_pre_key) = 32),
    signed_pre_key_id INTEGER NOT NULL CHECK (signed_pre_key_id >= 0 AND signed_pre_key_id < 16777216),
    signed_pre_key_sig BYTEA NOT NULL CHECK (length(signed_pre_key_sig) = 64),
    adv_key BYTEA NOT NULL,
    adv_details BYTEA NOT NULL,
    adv_account_sig BYTEA NOT NULL CHECK (length(adv_account_sig) = 64),
    adv_account_sig_key BYTEA CHECK (length(adv_account_sig_key) = 32),
    adv_device_sig BYTEA NOT NULL CHECK (length(adv_device_sig) = 64),
    platform TEXT NOT NULL DEFAULT '',
    business_name TEXT NOT NULL DEFAULT '',
    push_name TEXT NOT NULL DEFAULT '',
    lid_migration_ts BIGINT NOT NULL DEFAULT 0
);

-- Create identity keys table
CREATE TABLE whatsmeow_identity_keys (
    our_jid TEXT,
    their_id TEXT,
    identity BYTEA NOT NULL CHECK (length(identity) = 32),
    PRIMARY KEY (our_jid, their_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create pre-keys table (note: column is 'key' not 'key_pair')
CREATE TABLE whatsmeow_pre_keys (
    jid TEXT,
    key_id INTEGER CHECK (key_id >= 0 AND key_id < 16777216),
    key BYTEA NOT NULL CHECK (length(key) = 32),
    uploaded BOOLEAN NOT NULL,
    PRIMARY KEY (jid, key_id),
    FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create sessions table
CREATE TABLE whatsmeow_sessions (
    our_jid TEXT,
    their_id TEXT,
    session BYTEA,
    PRIMARY KEY (our_jid, their_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create sender keys table
CREATE TABLE whatsmeow_sender_keys (
    our_jid TEXT,
    chat_id TEXT,
    sender_id TEXT,
    sender_key BYTEA NOT NULL,
    PRIMARY KEY (our_jid, chat_id, sender_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create app state sync keys table
CREATE TABLE whatsmeow_app_state_sync_keys (
    jid TEXT,
    key_id BYTEA,
    key_data BYTEA NOT NULL,
    timestamp BIGINT NOT NULL,
    fingerprint BYTEA NOT NULL,
    PRIMARY KEY (jid, key_id),
    FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create app state version table
CREATE TABLE whatsmeow_app_state_version (
    jid TEXT,
    name TEXT,
    version BIGINT NOT NULL,
    hash BYTEA NOT NULL CHECK (length(hash) = 128),
    PRIMARY KEY (jid, name),
    FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create app state mutation macs table
CREATE TABLE whatsmeow_app_state_mutation_macs (
    jid TEXT,
    name TEXT,
    version BIGINT,
    index_mac BYTEA CHECK (length(index_mac) = 32),
    value_mac BYTEA NOT NULL CHECK (length(value_mac) = 32),
    PRIMARY KEY (jid, name, version, index_mac),
    FOREIGN KEY (jid, name) REFERENCES whatsmeow_app_state_version(jid, name) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create contacts table
CREATE TABLE whatsmeow_contacts (
    our_jid TEXT,
    their_jid TEXT,
    first_name TEXT,
    full_name TEXT,
    push_name TEXT,
    business_name TEXT,
    PRIMARY KEY (our_jid, their_jid),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create chat settings table
CREATE TABLE whatsmeow_chat_settings (
    our_jid TEXT,
    chat_jid TEXT,
    muted_until BIGINT NOT NULL DEFAULT 0,
    pinned BOOLEAN NOT NULL DEFAULT false,
    archived BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (our_jid, chat_jid),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create message secrets table
CREATE TABLE whatsmeow_message_secrets (
    our_jid TEXT,
    chat_jid TEXT,
    sender_jid TEXT,
    message_id TEXT,
    key BYTEA NOT NULL,
    PRIMARY KEY (our_jid, chat_jid, sender_jid, message_id),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create privacy tokens table
CREATE TABLE whatsmeow_privacy_tokens (
    our_jid TEXT,
    their_jid TEXT,
    token BYTEA NOT NULL,
    timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, their_jid)
);

-- Create LID map table (correct structure: lid -> pn mapping)
CREATE TABLE whatsmeow_lid_map (
    lid TEXT PRIMARY KEY,
    pn TEXT UNIQUE NOT NULL
);

-- Create event buffer table
CREATE TABLE whatsmeow_event_buffer (
    our_jid TEXT NOT NULL,
    ciphertext_hash BYTEA NOT NULL CHECK (length(ciphertext_hash) = 32),
    plaintext BYTEA,
    server_timestamp BIGINT NOT NULL,
    insert_timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, ciphertext_hash),
    FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create your custom application tables (from messages.db)
CREATE TABLE chats (
    jid TEXT PRIMARY KEY,
    name TEXT,
    last_message_time TIMESTAMP
);

CREATE TABLE messages (
    id TEXT,
    chat_jid TEXT,
    sender TEXT,
    content TEXT,
    timestamp TIMESTAMP,
    is_from_me BOOLEAN,
    media_type TEXT,
    filename TEXT,
    url TEXT,
    media_key BYTEA,
    file_sha256 BYTEA,
    file_enc_sha256 BYTEA,
    file_length INTEGER,
    PRIMARY KEY (id, chat_jid),
    FOREIGN KEY (chat_jid) REFERENCES chats(jid)
);

-- Create indexes for performance
CREATE INDEX whatsmeow_identity_keys_our_jid ON whatsmeow_identity_keys (our_jid);
CREATE INDEX whatsmeow_pre_keys_jid ON whatsmeow_pre_keys (jid);
CREATE INDEX whatsmeow_sessions_our_jid ON whatsmeow_sessions (our_jid);
CREATE INDEX whatsmeow_sender_keys_our_jid ON whatsmeow_sender_keys (our_jid);
CREATE INDEX whatsmeow_app_state_sync_keys_jid ON whatsmeow_app_state_sync_keys (jid);
CREATE INDEX whatsmeow_contacts_our_jid ON whatsmeow_contacts (our_jid);
CREATE INDEX whatsmeow_chat_settings_our_jid ON whatsmeow_chat_settings (our_jid);
CREATE INDEX whatsmeow_message_secrets_our_jid ON whatsmeow_message_secrets (our_jid);
CREATE INDEX whatsmeow_privacy_tokens_our_jid ON whatsmeow_privacy_tokens (our_jid);
CREATE INDEX whatsmeow_event_buffer_our_jid ON whatsmeow_event_buffer (our_jid);

-- Insert version info
INSERT INTO whatsmeow_version (version, compat) VALUES (1, 1);