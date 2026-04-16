-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_files" table
CREATE TABLE `new_files` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `file_name` text NULL,
  `original_file_name` text NULL,
  `file_size` integer NULL,
  `mime_type` text NULL,
  `public` numeric NULL,
  `expiry_date` datetime NULL DEFAULT (null),
  `uploader_id` integer NULL,
  CONSTRAINT `fk_files_uploader` FOREIGN KEY (`uploader_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "files" to new temporary table "new_files"
INSERT INTO `new_files` (`id`, `created_at`, `updated_at`, `file_name`, `original_file_name`, `file_size`, `mime_type`, `public`, `expiry_date`, `uploader_id`) SELECT `id`, `created_at`, `updated_at`, `file_name`, `original_file_name`, `file_size`, `mime_type`, `public`, `expiry_date`, `uploader_id` FROM `files`;
-- Drop "files" table after copying rows
DROP TABLE `files`;
-- Rename temporary table "new_files" to "files"
ALTER TABLE `new_files` RENAME TO `files`;
-- Create index "idx_files_uploader_id" to table: "files"
CREATE INDEX `idx_files_uploader_id` ON `files` (`uploader_id`);
-- Create index "idx_files_file_name" to table: "files"
CREATE UNIQUE INDEX `idx_files_file_name` ON `files` (`file_name`);
-- Create "new_invite_codes" table
CREATE TABLE `new_invite_codes` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `code` text NULL,
  `uses` integer NULL,
  `expiry_date` datetime NULL,
  `account_type` text NULL,
  `invite_creator_id` integer NULL DEFAULT (null),
  CONSTRAINT `fk_invite_codes_invite_creator` FOREIGN KEY (`invite_creator_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "invite_codes" to new temporary table "new_invite_codes"
INSERT INTO `new_invite_codes` (`id`, `created_at`, `updated_at`, `code`, `uses`, `expiry_date`, `account_type`, `invite_creator_id`) SELECT `id`, `created_at`, `updated_at`, `code`, `uses`, `expiry_date`, `account_type`, `invite_creator_id` FROM `invite_codes`;
-- Drop "invite_codes" table after copying rows
DROP TABLE `invite_codes`;
-- Rename temporary table "new_invite_codes" to "invite_codes"
ALTER TABLE `new_invite_codes` RENAME TO `invite_codes`;
-- Create index "idx_invite_codes_invite_creator_id" to table: "invite_codes"
CREATE INDEX `idx_invite_codes_invite_creator_id` ON `invite_codes` (`invite_creator_id`);
-- Create index "idx_invite_codes_code" to table: "invite_codes"
CREATE UNIQUE INDEX `idx_invite_codes_code` ON `invite_codes` (`code`);
-- Create "new_session_tokens" table
CREATE TABLE `new_session_tokens` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `last_used` datetime NULL,
  `expiry_date` datetime NULL,
  `token` text NULL,
  `account_id` integer NULL,
  CONSTRAINT `fk_session_tokens_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "session_tokens" to new temporary table "new_session_tokens"
INSERT INTO `new_session_tokens` (`id`, `created_at`, `updated_at`, `last_used`, `expiry_date`, `token`, `account_id`) SELECT `id`, `created_at`, `updated_at`, `last_used`, `expiry_date`, `token`, `account_id` FROM `session_tokens`;
-- Drop "session_tokens" table after copying rows
DROP TABLE `session_tokens`;
-- Rename temporary table "new_session_tokens" to "session_tokens"
ALTER TABLE `new_session_tokens` RENAME TO `session_tokens`;
-- Create index "idx_session_tokens_account_id" to table: "session_tokens"
CREATE INDEX `idx_session_tokens_account_id` ON `session_tokens` (`account_id`);
-- Create index "idx_session_tokens_token" to table: "session_tokens"
CREATE UNIQUE INDEX `idx_session_tokens_token` ON `session_tokens` (`token`);
-- Create "new_upload_tokens" table
CREATE TABLE `new_upload_tokens` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `last_used` datetime NULL,
  `nickname` text NULL,
  `token` text NULL,
  `account_id` integer NULL,
  CONSTRAINT `fk_upload_tokens_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "upload_tokens" to new temporary table "new_upload_tokens"
INSERT INTO `new_upload_tokens` (`id`, `created_at`, `updated_at`, `last_used`, `nickname`, `token`, `account_id`) SELECT `id`, `created_at`, `updated_at`, `last_used`, `nickname`, `token`, `account_id` FROM `upload_tokens`;
-- Drop "upload_tokens" table after copying rows
DROP TABLE `upload_tokens`;
-- Rename temporary table "new_upload_tokens" to "upload_tokens"
ALTER TABLE `new_upload_tokens` RENAME TO `upload_tokens`;
-- Create index "idx_upload_tokens_account_id" to table: "upload_tokens"
CREATE INDEX `idx_upload_tokens_account_id` ON `upload_tokens` (`account_id`);
-- Create index "idx_upload_tokens_token" to table: "upload_tokens"
CREATE UNIQUE INDEX `idx_upload_tokens_token` ON `upload_tokens` (`token`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
