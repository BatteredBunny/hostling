-- Modify "files" table
ALTER TABLE "files" DROP CONSTRAINT "fk_files_uploader", ADD CONSTRAINT "fk_files_uploader" FOREIGN KEY ("uploader_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "invite_codes" table
ALTER TABLE "invite_codes" DROP CONSTRAINT "fk_invite_codes_invite_creator", ADD CONSTRAINT "fk_invite_codes_invite_creator" FOREIGN KEY ("invite_creator_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "session_tokens" table
ALTER TABLE "session_tokens" DROP CONSTRAINT "fk_session_tokens_account", ADD CONSTRAINT "fk_session_tokens_account" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
-- Modify "upload_tokens" table
ALTER TABLE "upload_tokens" DROP CONSTRAINT "fk_upload_tokens_account", ADD CONSTRAINT "fk_upload_tokens_account" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE CASCADE;
