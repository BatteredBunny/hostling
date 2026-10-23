-- The application backfills this column before applying the next migration.
ALTER TABLE "session_tokens" ADD COLUMN "token_hash" text NULL;
CREATE UNIQUE INDEX "idx_session_tokens_token_hash" ON "session_tokens" ("token_hash");
