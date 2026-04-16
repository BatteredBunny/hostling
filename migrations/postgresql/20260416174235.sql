-- Create index "idx_files_uploader_id" to table: "files"
CREATE INDEX "idx_files_uploader_id" ON "files" ("uploader_id");
-- Create index "idx_invite_codes_invite_creator_id" to table: "invite_codes"
CREATE INDEX "idx_invite_codes_invite_creator_id" ON "invite_codes" ("invite_creator_id");
-- Create index "idx_session_tokens_account_id" to table: "session_tokens"
CREATE INDEX "idx_session_tokens_account_id" ON "session_tokens" ("account_id");
-- Create index "idx_upload_tokens_account_id" to table: "upload_tokens"
CREATE INDEX "idx_upload_tokens_account_id" ON "upload_tokens" ("account_id");
