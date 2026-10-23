-- Apply through Hostling so the Go backfill runs first.
ALTER TABLE "upload_tokens" ALTER COLUMN "token_hash" SET NOT NULL;
ALTER TABLE "upload_tokens" DROP COLUMN "token";
