-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_accounts" table
CREATE TABLE `new_accounts` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `github_id` integer NULL,
  `github_username` text NULL,
  `oidc_id` text NULL,
  `oidc_username` text NULL,
  `invited_by` integer NULL,
  `account_type` text NULL
);
-- Copy rows from old table "accounts" to new temporary table "new_accounts"
INSERT INTO `new_accounts` (`id`, `created_at`, `updated_at`, `github_id`, `github_username`, `oidc_id`, `oidc_username`, `invited_by`, `account_type`) SELECT `id`, `created_at`, `updated_at`, `github_id`, `github_username`, `oidc_id`, `oidc_username`, `invited_by`, `account_type` FROM `accounts`;
-- Drop "accounts" table after copying rows
DROP TABLE `accounts`;
-- Rename temporary table "new_accounts" to "accounts"
ALTER TABLE `new_accounts` RENAME TO `accounts`;
-- Create index "idx_accounts_oidc_id" to table: "accounts"
CREATE UNIQUE INDEX `idx_accounts_oidc_id` ON `accounts` (`oidc_id`) WHERE oidc_id <> '';
-- Create index "idx_accounts_github_id" to table: "accounts"
CREATE UNIQUE INDEX `idx_accounts_github_id` ON `accounts` (`github_id`) WHERE github_id <> 0;
-- Create index "idx_files_file_name" to table: "files"
CREATE UNIQUE INDEX `idx_files_file_name` ON `files` (`file_name`);
-- Create index "idx_invite_codes_code" to table: "invite_codes"
CREATE UNIQUE INDEX `idx_invite_codes_code` ON `invite_codes` (`code`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
