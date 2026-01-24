-- Modify "file_views" table
ALTER TABLE "file_views" DROP CONSTRAINT "fk_files_views", ADD CONSTRAINT "fk_files_views" FOREIGN KEY ("files_id") REFERENCES "files" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
