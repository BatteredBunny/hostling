{
  self,
  pkgs,
  testers,
}:
let
  port = 8080;
  token = "test-token";
  bucket = "hostling";
  accessKey = "GK0123456789abcdef01234567";
  secretKey = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210";
  s3Port = 3900;
  adminPort = 3903;
  rpcPort = 3901;
  rpcSecret = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef";

  dexPort = 5556;
  dexClientID = "hostling";
  dexClientSecret = "hostling-secret";
  dexUserEmail = "alice@example.com";
  dexUserPassword = "testpass";
  dexUserHash = "$2b$12$CHdVQpLOYn3blAhuVrSVHOus2ARHU0Tny8cvMYjuSmtSmQXcbnl5K"; # hash for the password above
in
testers.nixosTest {
  name = "hostling-advanced";

  interactive.nodes.machine = {
    services.hostling.openFirewall = true;

    virtualisation.forwardPorts = [
      {
        from = "host";
        host.port = 8839;
        guest.port = port;
      }
    ];
  };

  nodes.machine =
    { ... }:
    {
      imports = [
        self.nixosModules.default
      ];

      services.garage = {
        enable = true;
        package = pkgs.garage;
        settings = {
          replication_factor = 1;
          rpc_bind_addr = "127.0.0.1:${toString rpcPort}";
          rpc_public_addr = "127.0.0.1:${toString rpcPort}";
          rpc_secret = rpcSecret;
          s3_api = {
            api_bind_addr = "127.0.0.1:${toString s3Port}";
            s3_region = "garage";
            root_domain = ".s3.garage";
          };
          admin = {
            api_bind_addr = "127.0.0.1:${toString adminPort}";
            admin_token = "admintoken";
          };
        };
      };

      services.hostling = {
        enable = true;
        settings = {
          database_type = "sqlite";
          database_connection_url = "hostling.db";
          inherit port;
          rate_limit = 1000.0;
          max_upload_size = 8192;

          s3 = {
            endpoint = "localhost:${toString s3Port}";
            access_key_id = accessKey;
            secret_access_key = secretKey;
            inherit bucket;
            region = "garage";
            insecure = true;
            proxyfiles = true;
          };
        };
      };

      systemd.services.garage-init = {
        description = "Initialize Garage cluster layout, bucket and access key";
        after = [ "garage.service" ];
        requires = [ "garage.service" ];
        wantedBy = [ "multi-user.target" ];
        path = [ pkgs.garage ];
        serviceConfig = {
          Type = "oneshot";
          RemainAfterExit = true;
        };
        script = ''
          set -eu

          # Wait until garage is active
          until garage status >/dev/null 2>&1; do sleep 1; done

          node=$(garage node id -q | cut -d@ -f1)
          garage layout assign -z dc1 -c 1G "$node"
          garage layout apply --version 1

          garage key import --yes ${accessKey} ${secretKey}
          garage bucket create ${bucket}
          garage bucket allow --read --write --owner ${bucket} --key ${accessKey}
        '';
      };

      services.dex = {
        enable = true;
        settings = {
          issuer = "http://localhost:${toString dexPort}/dex";
          storage.type = "memory";
          web.http = "127.0.0.1:${toString dexPort}";
          oauth2.skipApprovalScreen = true;
          enablePasswordDB = true;
          staticPasswords = [
            {
              email = dexUserEmail;
              hash = dexUserHash;
              username = "alice";
              userID = "alice-user-id";
            }
          ];
          staticClients = [
            {
              id = dexClientID;
              secret = dexClientSecret;
              redirectURIs = [
                "http://localhost:${toString port}/api/auth/login/openid-connect/callback"
              ];
              name = "Hostling";
            }
          ];
        };
      };

      systemd.services.hostling = {
        after = [
          "garage-init.service"
          "dex.service"
        ];
        requires = [
          "garage-init.service"
          "dex.service"
        ];
        environment = {
          # in real world scenariou this would be in environmentFile
          INITIAL_REGISTER_TOKEN = token;
          OPENID_CONNECT_CLIENT_ID = dexClientID;
          OPENID_CONNECT_CLIENT_SECRET = dexClientSecret;
          OPENID_CONNECT_DISCOVERY_URL = "http://localhost:${toString dexPort}/dex/.well-known/openid-configuration";
        };
      };
    };

  testScript =
    { nodes, ... }:
    ''
      PORT = "${toString nodes.machine.services.hostling.settings.port}"
      TOKEN = "${token}"
      DEX_PORT = "${toString dexPort}"
      DEX_EMAIL = "${dexUserEmail}"
      DEX_PASSWORD = "${dexUserPassword}"
      exec(open("${./test_script_advanced.py}").read())
    '';
}
