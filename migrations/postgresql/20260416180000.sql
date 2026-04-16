-- Create partial unique index "idx_accounts_github_id" to table: "accounts"
CREATE UNIQUE INDEX "idx_accounts_github_id" ON "accounts" ("github_id") WHERE "github_id" <> 0;
-- Create partial unique index "idx_accounts_oidc_id" to table: "accounts"
CREATE UNIQUE INDEX "idx_accounts_oidc_id" ON "accounts" ("oidc_id") WHERE "oidc_id" <> '';
