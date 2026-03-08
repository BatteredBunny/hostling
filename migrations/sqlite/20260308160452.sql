-- Disable the enforcement of foreign-keys constraints
PRAGMA foreign_keys = off;
-- Create "new_file_tags" table
CREATE TABLE `new_file_tags` (
  `files_id` integer NULL,
  `tag_name` text NULL,
  PRIMARY KEY (`files_id`, `tag_name`),
  CONSTRAINT `fk_file_tags_tag` FOREIGN KEY (`tag_name`) REFERENCES `tags` (`name`) ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT `fk_file_tags_files` FOREIGN KEY (`files_id`) REFERENCES `files` (`id`) ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Copy rows from old table "file_tags" to new temporary table "new_file_tags"
INSERT INTO `new_file_tags` (`files_id`, `tag_name`) SELECT `files_id`, `tag_name` FROM `file_tags`;
-- Drop "file_tags" table after copying rows
DROP TABLE `file_tags`;
-- Rename temporary table "new_file_tags" to "file_tags"
ALTER TABLE `new_file_tags` RENAME TO `file_tags`;
-- Enable back the enforcement of foreign-keys constraints
PRAGMA foreign_keys = on;
