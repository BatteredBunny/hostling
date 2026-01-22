<h1 align="center">Hostling</h1>

Simple file hosting service

Main page             | Mobile view             | File modal
:-------------------------:|:-------------------------:|:-------------------------:
<img width="1416" height="1338" alt="image" src="https://github.com/user-attachments/assets/27f1cefd-87c3-413e-9b58-79ff2cf69ceb" />  |  <img width="489" height="861" alt="image" src="https://github.com/user-attachments/assets/fd12e620-3741-454b-b12b-7f88d50decdc" />  |  <img width="1408" height="1006" alt="image" src="https://github.com/user-attachments/assets/1b51f0dd-b245-4c0c-8ce5-8e6e13b54132" />

# Features
- Easy social login via github
- Account invite codes for enrolling new users
- Image automatic deletion, tagging, filtering, sorting
- Seperate upload tokens for automation setups (e.g scripts)
- Store data locally or on a S3/B2 bucket
- Sqlite and postgresql support
- View tracking

# Usage

Deploy the service with either the nixos module or docker-compose then configure the service.

Have a look at the example configs in ``examples/``

# Config reference

Configuration is done via a TOML file (default: `config.toml`). Use the `-c` flag to specify a different location.

## Setting up social login

These environment variables are required for using Github for social login:

* `GITHUB_CLIENT_ID`: GitHub OAuth application ID
* `GITHUB_SECRET`: GitHub OAuth application secret

## Config options

* `data_folder`: Directory for local file storage. Only used when S3 is not configured.
* `max_upload_size`: Maximum file upload size in bytes. Defaults to 100MB.
* `database_type`: Database type: `"sqlite"` or `"postgresql"` |
* `database_connection_url`: Database connection string. For SQLite: filename (e.g., `"hostling.db"`). For PostgreSQL: connection string (e.g., `"host=localhost port=5432 user=postgres * sslmode=disable"`) |
* `port`: Port to run the HTTP server on (e.g., `"8080"`) |
* `behind_reverse_proxy`: Set to `true` if running behind a reverse proxy (nginx, Caddy, etc.) |
* `trusted_proxy`: Trusted proxy IP address. Used for rate limiting and IP detection. Required when hosting it from behind a reverse proxy.
* `public_url`: Public URL of the service. Required for GitHub OAuth callbacks. Include protocol and domain (e.g., `"https://files.example.com"`) |
* `branding`: Custom branding text displayed in the interface. Maximum 20 characters. Defaults to `"Hostling"`
* `tagline`: Tagline for meta description and index page. Maximum 100 characters. Defaults to `"Simple file hosting service"`

## Bucket storage setup

The below options will go in the `[s3]` section

* `access_key_id`: S3/B2 access key ID (can also be set via `S3_ACCESS_KEY_ID` environment variable)
* `secret_access_key`: S3/B2 secret access key (can also be set via `S3_SECRET_ACCESS_KEY` environment variable)
* `bucket`: S3/B2 bucket name (NOT the bucket ID)
* `region`: S3/B2 region (e.g., `"us-east-1"`)
* `endpoint`: S3/B2 endpoint URL (e.g., `"https://s3.us-west-002.backblazeb2.com"`)

# Setup
## Setup with nixos module

```nix
inputs = {
    hostling.url = "github:BatteredBunny/hostling";
};
```

```nix
imports = [ inputs.hostling.nixosModules.default ];

services = {
    hostling = {
        enable = true;
        createDbLocally = true;
        openFirewall = false;
        settings.database_type = "postgresql";
    };

    postgresql.enable = true;
};
```

## Setup with docker

Have a look at docker-compose.yml

# Development

## Dev setup with nix

```
nix run .#test-service.driverInteractive
# Then visit http://localhost:8080
```