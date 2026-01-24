-- Create "accounts" table
CREATE TABLE `accounts` (
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
-- Create "files" table
CREATE TABLE `files` (
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
  CONSTRAINT `fk_files_uploader` FOREIGN KEY (`uploader_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "tags" table
CREATE TABLE `tags` (
  `name` text NULL,
  PRIMARY KEY (`name`)
);
-- Create "file_tags" table
CREATE TABLE `file_tags` (
  `files_id` integer NULL,
  `tag_name` text NULL,
  PRIMARY KEY (`files_id`, `tag_name`),
  CONSTRAINT `fk_file_tags_tag` FOREIGN KEY (`tag_name`) REFERENCES `tags` (`name`) ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT `fk_file_tags_files` FOREIGN KEY (`files_id`) REFERENCES `files` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "file_views" table
CREATE TABLE `file_views` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `ip_hash` text NULL,
  `files_id` integer NULL,
  CONSTRAINT `fk_files_views` FOREIGN KEY (`files_id`) REFERENCES `files` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_file_views_hash_collision" to table: "file_views"
CREATE UNIQUE INDEX `idx_file_views_hash_collision` ON `file_views` (`ip_hash`, `files_id`);
-- Create "invite_codes" table
CREATE TABLE `invite_codes` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `code` text NULL,
  `uses` integer NULL,
  `expiry_date` datetime NULL,
  `account_type` text NULL,
  `invite_creator_id` integer NULL DEFAULT (null),
  CONSTRAINT `fk_invite_codes_invite_creator` FOREIGN KEY (`invite_creator_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "session_tokens" table
CREATE TABLE `session_tokens` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `last_used` datetime NULL,
  `expiry_date` datetime NULL,
  `token` text NULL,
  `account_id` integer NULL,
  CONSTRAINT `fk_session_tokens_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_session_tokens_token" to table: "session_tokens"
CREATE UNIQUE INDEX `idx_session_tokens_token` ON `session_tokens` (`token`);
-- Create "upload_tokens" table
CREATE TABLE `upload_tokens` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `last_used` datetime NULL,
  `nickname` text NULL,
  `token` text NULL,
  `account_id` integer NULL,
  CONSTRAINT `fk_upload_tokens_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_upload_tokens_token" to table: "upload_tokens"
CREATE UNIQUE INDEX `idx_upload_tokens_token` ON `upload_tokens` (`token`);
