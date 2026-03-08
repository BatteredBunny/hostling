-- Modify "file_tags" table
ALTER TABLE "file_tags" DROP CONSTRAINT "fk_file_tags_files", DROP CONSTRAINT "fk_file_tags_tag", ADD CONSTRAINT "fk_file_tags_files" FOREIGN KEY ("files_id") REFERENCES "files" ("id") ON UPDATE NO ACTION ON DELETE CASCADE, ADD CONSTRAINT "fk_file_tags_tag" FOREIGN KEY ("tag_name") REFERENCES "tags" ("name") ON UPDATE NO ACTION ON DELETE CASCADE;
