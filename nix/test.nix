{
  self,
  testers,
  lib,
  dbType ? "postgresql",
}:
let
  port = 8080;
  token = "test-token";
in
testers.nixosTest {
  name = "hostling-${dbType}";

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

      services.hostling = {
        enable = true;
        createDbLocally = dbType == "postgresql";
        settings = {
          database_type = dbType;
          database_connection_url = lib.mkIf (dbType == "sqlite") "hostling.db";
          inherit port;
          rate_limit = 1000.0; # To make sure test can run as fast as possible. Might better to add a compile flag to just run off rate limiting, not sure.
          max_upload_size = 8192;
        };
      };

      services.postgresql.enable = dbType == "postgresql";

      # Usually you should only include secrets in environmentFile, but for testing its fine
      systemd.services.hostling.environment.INITIAL_REGISTER_TOKEN = token;
    };

  testScript =
    { nodes, ... }:
    ''
      PORT = "${toString nodes.machine.services.hostling.settings.port}"
      TOKEN = "${token}"
      DB_TYPE = "${dbType}"
      exec(open("${./test_script.py}").read())
    '';
}
