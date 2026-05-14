-- Create index "idx_file_views_files_id" to table: "file_views"
CREATE INDEX `idx_file_views_files_id` ON `file_views` (`files_id`);
-- Create index "idx_files_expiry_date" to table: "files"
CREATE INDEX `idx_files_expiry_date` ON `files` (`expiry_date`);
-- Create index "idx_file_tags_tag_name" to table: "file_tags"
CREATE INDEX `idx_file_tags_tag_name` ON `file_tags` (`tag_name`);
