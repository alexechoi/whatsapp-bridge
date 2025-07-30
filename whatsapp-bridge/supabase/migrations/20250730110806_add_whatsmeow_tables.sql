-- Migration: Add missing whatsmeow tables
-- This adds the missing tables that whatsmeow expects but weren't in the original migration

-- LID-PN mapping table - stores LID to phone number mappings
CREATE TABLE IF NOT EXISTS whatsmeow_lid_map (
    our_jid TEXT,
    lid_jid TEXT,
    phone_jid TEXT,
    PRIMARY KEY (our_jid, lid_jid)
);

-- Privacy tokens table - stores privacy tokens from contacts
CREATE TABLE IF NOT EXISTS whatsmeow_privacy_tokens (
    our_jid TEXT,
    their_jid TEXT,
    token BYTEA NOT NULL,
    timestamp BIGINT NOT NULL,
    PRIMARY KEY (our_jid, their_jid)
);

-- Add missing 'key' column to pre_keys table if it doesn't exist
-- This is a safe operation that won't fail if the column already exists
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'whatsmeow_pre_keys' AND column_name = 'key'
    ) THEN
        ALTER TABLE whatsmeow_pre_keys ADD COLUMN key BYTEA;
    END IF;
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS whatsmeow_lid_map_our_jid ON whatsmeow_lid_map (our_jid);
CREATE INDEX IF NOT EXISTS whatsmeow_privacy_tokens_our_jid ON whatsmeow_privacy_tokens (our_jid);
