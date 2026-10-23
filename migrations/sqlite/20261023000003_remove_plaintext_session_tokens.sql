-- Apply through Hostling so the Go backfill runs first, in the same transaction
-- as schema finalization and Atlas revision tracking.
CREATE TABLE `new_session_tokens` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `last_used` datetime NULL,
  `expiry_date` datetime NULL,
  `token_hash` text NOT NULL,
  `account_id` integer NULL,
  CONSTRAINT `fk_session_tokens_account` FOREIGN KEY (`account_id`) REFERENCES `accounts` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
INSERT INTO `new_session_tokens` (`id`, `created_at`, `updated_at`, `last_used`, `expiry_date`, `token_hash`, `account_id`)
SELECT `id`, `created_at`, `updated_at`, `last_used`, `expiry_date`, `token_hash`, `account_id` FROM `session_tokens`;
DROP TABLE `session_tokens`;
ALTER TABLE `new_session_tokens` RENAME TO `session_tokens`;
CREATE INDEX `idx_session_tokens_account_id` ON `session_tokens` (`account_id`);
CREATE UNIQUE INDEX `idx_session_tokens_token_hash` ON `session_tokens` (`token_hash`);
