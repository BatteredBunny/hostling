-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_file_views" table
CREATE TABLE `new_file_views` (
  `id` integer NULL PRIMARY KEY AUTOINCREMENT,
  `created_at` datetime NULL,
  `updated_at` datetime NULL,
  `ip_hash` text NULL,
  `files_id` integer NULL,
  CONSTRAINT `fk_files_views` FOREIGN KEY (`files_id`) REFERENCES `files` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "file_views" to new temporary table "new_file_views"
INSERT INTO `new_file_views` (`id`, `created_at`, `updated_at`, `ip_hash`, `files_id`) SELECT `id`, `created_at`, `updated_at`, `ip_hash`, `files_id` FROM `file_views`;
-- Drop "file_views" table after copying rows
DROP TABLE `file_views`;
-- Rename temporary table "new_file_views" to "file_views"
ALTER TABLE `new_file_views` RENAME TO `file_views`;
-- Create index "idx_file_views_hash_collision" to table: "file_views"
CREATE UNIQUE INDEX `idx_file_views_hash_collision` ON `file_views` (`ip_hash`, `files_id`);
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
