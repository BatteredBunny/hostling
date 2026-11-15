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

  nodes.machine = {
    imports = [ self.nixosModules.default ];

    services.hostling = {
      enable = true;
      createDbLocally = dbType == "postgresql";
      settings = {
        database_type = dbType;
        database_connection_url = lib.mkIf (dbType == "sqlite") "hostling.db";
        inherit port;
      };
    };

    # Usually you should only include secrets in environmentFile, but for testing its fine
    systemd.services.hostling.environment.INITIAL_REGISTER_TOKEN = token;
  };

  testScript = ''
    url = "http://localhost:${toString port}"
    content = "hostling-service-smoke-test"

    start_all()
    machine.wait_for_unit("hostling.service")
    machine.wait_for_open_port(${toString port})
    machine.succeed(f"curl -fsS {url}/ > /dev/null")

    machine.succeed(f"curl -fsS -c /tmp/cookies -L --data-urlencode code=${token} {url}/api/auth/register > /dev/null")
    machine.succeed(f"printf %s {content} > /tmp/dummy.txt")
    uploaded = machine.succeed(
        f"curl -fsS -b /tmp/cookies -F file=@/tmp/dummy.txt -F plain=true {url}/api/file/upload"
    ).strip()
    assert uploaded, "expected upload to return a file name"

    downloaded = machine.succeed(f"curl -fsS {url}{uploaded}")
    assert downloaded == content, f"file content mismatch: got {downloaded!r}"
  '';
}
