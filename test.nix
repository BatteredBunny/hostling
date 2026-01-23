{ self, testers }:
let
  port = 8080;
in
testers.nixosTest {
  name = "hostling";

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
        createDbLocally = true;
        settings = {
          database_type = "postgresql";
          inherit port;
        };
      };

      services.postgresql.enable = true;
    };

  testScript =
    { nodes, ... }:
    let
      port = toString nodes.machine.services.hostling.settings.port;
    in
    ''
      start_all()
      machine.wait_for_unit("postgresql.service")
      machine.wait_for_unit("hostling.service")
      machine.wait_for_open_port(${port})
      machine.succeed("curl -f http://localhost:${port}/")
    '';
}
