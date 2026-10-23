-- Apply through Hostling so the Go backfill runs first.
ALTER TABLE "session_tokens" ALTER COLUMN "token_hash" SET NOT NULL;
ALTER TABLE "session_tokens" DROP COLUMN "token";
