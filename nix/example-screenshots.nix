{
  self,
  testers,
  pkgs,
}:
let
  port = 8080;
  token = "screenshot-token";
  mascotFile = ../public/assets/mascot.png;
  pythonEnv = pkgs.python3.withPackages (ps: [
    ps.selenium
  ]);
in
testers.nixosTest {
  name = "hostling-screenshots";

  nodes.machine =
    { ... }:
    {
      imports = [ self.nixosModules.default ];

      services.hostling = {
        enable = true;
        settings = {
          database_type = "sqlite";
          database_connection_url = "hostling.db";
          port = port;
        };
      };

      systemd.services.hostling.environment.INITIAL_REGISTER_TOKEN = token;

      environment.systemPackages = with pkgs; [
        curl
        chromium
        chromedriver
        pythonEnv
      ];
    };

  testScript =
    { ... }:
    ''
      import os

      start_all()
      machine.wait_for_unit("hostling.service")
      machine.wait_for_open_port(${toString port})

      machine.succeed(
          "PORT='${toString port}' "
          "TOKEN='${token}' "
          "MASCOT='${mascotFile}' "
          "${pythonEnv}/bin/python3 ${./hostling-capture.py}"
      )

      os.makedirs("screenshots", exist_ok=True)
      machine.copy_from_vm("/tmp/upload.png", "screenshots")
      machine.copy_from_vm("/tmp/gallery.png", "screenshots")
      machine.copy_from_vm("/tmp/modal.png", "screenshots")
      machine.copy_from_vm("/tmp/admin.png", "screenshots")
    '';
}
