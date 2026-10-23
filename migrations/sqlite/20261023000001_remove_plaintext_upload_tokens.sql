-- Apply through Hostling so the Go backfill runs first. The NOT NULL constraint
-- prevents removing the plaintext column if any token has not been migrated.
CREATE TABLE `new_upload_tokens` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `last_used` datetime NULL,
  `nickname` text NULL,
  `token_hash` text NOT NULL,
  `account_id` integer NULL,
  CONSTRAINT `fk_upload_tokens_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
INSERT INTO `new_upload_tokens` (`id`, `created_at`, `updated_at`, `last_used`, `nickname`, `token_hash`, `account_id`)
SELECT `id`, `created_at`, `updated_at`, `last_used`, `nickname`, `token_hash`, `account_id` FROM `upload_tokens`;
DROP TABLE `upload_tokens`;
ALTER TABLE `new_upload_tokens` RENAME TO `upload_tokens`;
CREATE INDEX `idx_upload_tokens_account_id` ON `upload_tokens` (`account_id`);
CREATE UNIQUE INDEX `idx_upload_tokens_token_hash` ON `upload_tokens` (`token_hash`);
