-- Create index "idx_files_file_name" to table: "files"
CREATE UNIQUE INDEX "idx_files_file_name" ON "files" ("file_name");
-- Create index "idx_invite_codes_code" to table: "invite_codes"
CREATE UNIQUE INDEX "idx_invite_codes_code" ON "invite_codes" ("code");
