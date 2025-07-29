-- Migration: Create whatsmeow tables
-- This creates the tables that the whatsmeow sqlstore package expects

-- Devices table - stores device registration info
CREATE TABLE IF NOT EXISTS whatsmeow_device (
    jid TEXT PRIMARY KEY,
    registration_id INTEGER NOT NULL,
    noise_key BYTEA NOT NULL,
    identity_key BYTEA NOT NULL,
    signed_pre_key BYTEA NOT NULL,
    signed_pre_key_id INTEGER NOT NULL,
    signed_pre_key_sig BYTEA NOT NULL,
    adv_key BYTEA,
    adv_details BYTEA,
    adv_account_sig BYTEA,
    adv_account_sig_key BYTEA,
    adv_device_sig BYTEA,
    platform TEXT NOT NULL DEFAULT '',
    business_name TEXT NOT NULL DEFAULT '',
    push_name TEXT NOT NULL DEFAULT ''
);

-- Sessions table - stores session keys for contacts
CREATE TABLE IF NOT EXISTS whatsmeow_identity_keys (
    our_jid TEXT,
    their_id TEXT,
    identity BYTEA NOT NULL,
    PRIMARY KEY (our_jid, their_id)
);

-- Pre-keys table
CREATE TABLE IF NOT EXISTS whatsmeow_pre_keys (
    jid TEXT,
    key_id INTEGER,
    key_pair BYTEA NOT NULL,
    uploaded BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (jid, key_id)
);

-- Sender keys table
CREATE TABLE IF NOT EXISTS whatsmeow_sender_keys (
    our_jid TEXT,
    chat_jid TEXT,
    sender_jid TEXT,
    sender_key BYTEA NOT NULL,
    PRIMARY KEY (our_jid, chat_jid, sender_jid)
);

-- App state sync keys
CREATE TABLE IF NOT EXISTS whatsmeow_app_state_sync_keys (
    jid TEXT,
    key_id BYTEA,
    key_data BYTEA NOT NULL,
    timestamp BIGINT NOT NULL,
    fingerprint BYTEA NOT NULL,
    PRIMARY KEY (jid, key_id)
);

-- App state version table
CREATE TABLE IF NOT EXISTS whatsmeow_app_state_version (
    jid TEXT,
    name TEXT,
    version BIGINT NOT NULL,
    hash BYTEA NOT NULL,
    PRIMARY KEY (jid, name)
);

-- App state mutations table
CREATE TABLE IF NOT EXISTS whatsmeow_app_state_mutation_macs (
    jid TEXT,
    name TEXT,
    version BIGINT,
    index_mac BYTEA,
    value_mac BYTEA,
    PRIMARY KEY (jid, name, version, index_mac)
);

-- Contacts table
CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
    our_jid TEXT,
    their_jid TEXT,
    first_name TEXT,
    full_name TEXT,
    push_name TEXT,
    business_name TEXT,
    PRIMARY KEY (our_jid, their_jid)
);

-- Chat settings table
CREATE TABLE IF NOT EXISTS whatsmeow_chat_settings (
    our_jid TEXT,
    chat_jid TEXT,
    muted_until BIGINT NOT NULL DEFAULT 0,
    pinned BOOLEAN NOT NULL DEFAULT FALSE,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (our_jid, chat_jid)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS whatsmeow_identity_keys_our_jid ON whatsmeow_identity_keys (our_jid);
CREATE INDEX IF NOT EXISTS whatsmeow_pre_keys_jid ON whatsmeow_pre_keys (jid);
CREATE INDEX IF NOT EXISTS whatsmeow_sender_keys_our_jid ON whatsmeow_sender_keys (our_jid);
CREATE INDEX IF NOT EXISTS whatsmeow_app_state_sync_keys_jid ON whatsmeow_app_state_sync_keys (jid);
CREATE INDEX IF NOT EXISTS whatsmeow_contacts_our_jid ON whatsmeow_contacts (our_jid);