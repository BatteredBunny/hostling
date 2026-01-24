-- Create "accounts" table
CREATE TABLE "accounts" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "github_id" bigint NULL,
  "github_username" text NULL,
  "oidc_id" text NULL,
  "oidc_username" text NULL,
  "invited_by" bigint NULL,
  "account_type" text NULL,
  PRIMARY KEY ("id")
);
-- Create "files" table
CREATE TABLE "files" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "file_name" text NULL,
  "original_file_name" text NULL,
  "file_size" bigint NULL,
  "mime_type" text NULL,
  "public" boolean NULL,
  "expiry_date" timestamptz NULL,
  "uploader_id" bigint NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_files_uploader" FOREIGN KEY ("uploader_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "tags" table
CREATE TABLE "tags" (
  "name" text NOT NULL,
  PRIMARY KEY ("name")
);
-- Create "file_tags" table
CREATE TABLE "file_tags" (
  "files_id" bigint NOT NULL,
  "tag_name" text NOT NULL,
  PRIMARY KEY ("files_id", "tag_name"),
  CONSTRAINT "fk_file_tags_files" FOREIGN KEY ("files_id") REFERENCES "files" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_file_tags_tag" FOREIGN KEY ("tag_name") REFERENCES "tags" ("name") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "file_views" table
CREATE TABLE "file_views" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "ip_hash" text NULL,
  "files_id" bigint NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_files_views" FOREIGN KEY ("files_id") REFERENCES "files" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_file_views_hash_collision" to table: "file_views"
CREATE UNIQUE INDEX "idx_file_views_hash_collision" ON "file_views" ("ip_hash", "files_id");
-- Create "invite_codes" table
CREATE TABLE "invite_codes" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "code" text NULL,
  "uses" bigint NULL,
  "expiry_date" timestamptz NULL,
  "account_type" text NULL,
  "invite_creator_id" bigint NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_invite_codes_invite_creator" FOREIGN KEY ("invite_creator_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create "session_tokens" table
CREATE TABLE "session_tokens" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "last_used" timestamptz NULL,
  "expiry_date" timestamptz NULL,
  "token" text NULL,
  "account_id" bigint NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_session_tokens_account" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_session_tokens_token" to table: "session_tokens"
CREATE UNIQUE INDEX "idx_session_tokens_token" ON "session_tokens" ("token");
-- Create "upload_tokens" table
CREATE TABLE "upload_tokens" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "last_used" timestamptz NULL,
  "nickname" text NULL,
  "token" text NULL,
  "account_id" bigint NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_upload_tokens_account" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_upload_tokens_token" to table: "upload_tokens"
CREATE UNIQUE INDEX "idx_upload_tokens_token" ON "upload_tokens" ("token");
