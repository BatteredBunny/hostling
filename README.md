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

TODO

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