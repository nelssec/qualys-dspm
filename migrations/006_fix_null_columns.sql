-- Migration: Fix NULL columns that map to non-nullable Go strings

-- Fix cloud_accounts NULL columns
UPDATE cloud_accounts SET status_message = '' WHERE status_message IS NULL;
UPDATE cloud_accounts SET last_scan_status = '' WHERE last_scan_status IS NULL;
UPDATE cloud_accounts SET display_name = '' WHERE display_name IS NULL;

ALTER TABLE cloud_accounts
    ALTER COLUMN status_message SET DEFAULT '',
    ALTER COLUMN last_scan_status SET DEFAULT '',
    ALTER COLUMN display_name SET DEFAULT '';

-- Fix data_assets NULL columns
UPDATE data_assets SET region = '' WHERE region IS NULL;
UPDATE data_assets SET encryption_status = 'NONE' WHERE encryption_status IS NULL;
UPDATE data_assets SET encryption_key_arn = '' WHERE encryption_key_arn IS NULL;
UPDATE data_assets SET sensitivity_level = 'UNKNOWN' WHERE sensitivity_level IS NULL;
UPDATE data_assets SET scan_status = '' WHERE scan_status IS NULL;
UPDATE data_assets SET scan_error = '' WHERE scan_error IS NULL;

ALTER TABLE data_assets
    ALTER COLUMN region SET DEFAULT '',
    ALTER COLUMN encryption_status SET DEFAULT 'NONE',
    ALTER COLUMN encryption_key_arn SET DEFAULT '',
    ALTER COLUMN sensitivity_level SET DEFAULT 'UNKNOWN',
    ALTER COLUMN scan_status SET DEFAULT '',
    ALTER COLUMN scan_error SET DEFAULT '';

-- Fix access_edges NULL columns
UPDATE access_edges SET source_name = '' WHERE source_name IS NULL;
UPDATE access_edges SET target_arn = '' WHERE target_arn IS NULL;
UPDATE access_edges SET permission_level = 'READ' WHERE permission_level IS NULL;
UPDATE access_edges SET grant_type = '' WHERE grant_type IS NULL;

ALTER TABLE access_edges
    ALTER COLUMN source_name SET DEFAULT '',
    ALTER COLUMN target_arn SET DEFAULT '',
    ALTER COLUMN grant_type SET DEFAULT '';

-- Fix access_policies NULL columns
UPDATE access_policies SET policy_name = '' WHERE policy_name IS NULL;
UPDATE access_policies SET policy_type = '' WHERE policy_type IS NULL;
UPDATE access_policies SET policy_version = '' WHERE policy_version IS NULL;
UPDATE access_policies SET analysis_notes = '' WHERE analysis_notes IS NULL;

ALTER TABLE access_policies
    ALTER COLUMN policy_name SET DEFAULT '',
    ALTER COLUMN policy_type SET DEFAULT '',
    ALTER COLUMN policy_version SET DEFAULT '',
    ALTER COLUMN analysis_notes SET DEFAULT '';

-- Fix findings NULL columns
UPDATE findings SET description = '' WHERE description IS NULL;
UPDATE findings SET remediation = '' WHERE remediation IS NULL;
UPDATE findings SET status_reason = '' WHERE status_reason IS NULL;
UPDATE findings SET assigned_to = '' WHERE assigned_to IS NULL;

ALTER TABLE findings
    ALTER COLUMN description SET DEFAULT '',
    ALTER COLUMN remediation SET DEFAULT '',
    ALTER COLUMN status_reason SET DEFAULT '',
    ALTER COLUMN assigned_to SET DEFAULT '';

-- Fix scan_jobs NULL columns
UPDATE scan_jobs SET triggered_by = '' WHERE triggered_by IS NULL;
UPDATE scan_jobs SET worker_id = '' WHERE worker_id IS NULL;

ALTER TABLE scan_jobs
    ALTER COLUMN triggered_by SET DEFAULT '',
    ALTER COLUMN worker_id SET DEFAULT '';
