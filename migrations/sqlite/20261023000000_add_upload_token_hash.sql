-- The application backfills this column before applying the next migration.
ALTER TABLE `upload_tokens` ADD COLUMN `token_hash` text NULL;
CREATE UNIQUE INDEX `idx_upload_tokens_token_hash` ON `upload_tokens` (`token_hash`);
