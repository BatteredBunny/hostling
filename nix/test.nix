{
  self,
  testers,
  lib,
  dbType ? "postgresql",
}:
let
  port = 8080;
  token = "test-token";
  dummyFile = ./dummy.txt;
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
        };
      };

      services.postgresql.enable = dbType == "postgresql";

      # Usually you should only include secrets in environmentFile, but for testing its fine
      systemd.services.hostling.environment.INITIAL_REGISTER_TOKEN = token;
    };

  testScript =
    { nodes, ... }:
    let
      port = toString nodes.machine.services.hostling.settings.port;
    in
    ''
      import json

      start_all()
      ${lib.optionalString (dbType == "postgresql") ''machine.wait_for_unit("postgresql.service")''}
      machine.wait_for_unit("hostling.service")
      machine.wait_for_open_port(${port})
      machine.succeed("curl -f http://localhost:${port}/")

      # These are supposed to be called by a browser usually, thats why the usage of cookies
      machine.succeed("curl -f -c /tmp/cookies.txt -L -X POST --data-urlencode 'code=${token}' 'http://localhost:${port}/api/auth/register'")

      # Make sure upload works
      uploaded_path = machine.succeed("curl -f -b /tmp/cookies.txt -F 'file=@${dummyFile}' -F 'plain=true' 'http://localhost:${port}/api/file/upload'").strip()

      # Verify the uploaded file is accessible and has correct content
      downloaded = machine.succeed(f"curl -f 'http://localhost:${port}{uploaded_path}'")
      expected = machine.succeed("cat ${dummyFile}")
      assert downloaded == expected, f"File content mismatch: got {repr(downloaded)}, expected {repr(expected)}"

      # Upload a second larger file for sorting tests
      machine.succeed("head -c 1024 /dev/urandom > /tmp/larger_dummy.bin")
      uploaded_path2 = machine.succeed("curl -f -b /tmp/cookies.txt -F 'file=@/tmp/larger_dummy.bin' -F 'plain=true' 'http://localhost:${port}/api/file/upload'").strip()

      small_file = uploaded_path.lstrip("/")
      large_file = uploaded_path2.lstrip("/")

      def api_get(path):
          return json.loads(machine.succeed(f"curl -f -b /tmp/cookies.txt 'http://localhost:${port}{path}'"))

      # File stats
      stats = api_get("/api/account/files/stats")
      assert stats["count"] == 2, f"Expected 2 files, got {stats['count']}"
      assert stats["size_total"] > 0, "Expected non-zero total size"

      # File listing (default: created_at desc)
      listing = api_get("/api/account/files")
      assert listing["count"] == 2, f"Expected count=2, got {listing['count']}"
      assert len(listing["files"]) == 2, f"Expected 2 files, got {len(listing['files'])}"

      # Sorting by file_size
      listing_desc = api_get("/api/account/files?sort=file_size&desc=true")
      assert listing_desc["files"][0]["FileName"] == large_file, \
          f"Expected large file first when sorting desc, got {listing_desc['files'][0]['FileName']}"
      assert listing_desc["files"][1]["FileName"] == small_file, \
          f"Expected small file second when sorting desc, got {listing_desc['files'][1]['FileName']}"

      listing_asc = api_get("/api/account/files?sort=file_size&desc=false")
      assert listing_asc["files"][0]["FileName"] == small_file, \
          f"Expected small file first when sorting asc, got {listing_asc['files'][0]['FileName']}"
      assert listing_asc["files"][1]["FileName"] == large_file, \
          f"Expected large file second when sorting asc, got {listing_asc['files'][1]['FileName']}"

      # Sorting by created_at
      listing_newest = api_get("/api/account/files?sort=created_at&desc=true")
      assert listing_newest["files"][0]["FileName"] == large_file, \
          "Expected newest file (large) first when sorting created_at desc"

      listing_oldest = api_get("/api/account/files?sort=created_at&desc=false")
      assert listing_oldest["files"][0]["FileName"] == small_file, \
          "Expected oldest file (small) first when sorting created_at asc"

      # Tagging
      machine.succeed(f"curl -f -b /tmp/cookies.txt -F 'file_name={small_file}' -F 'tag=alpha' 'http://localhost:${port}/api/account/file/tag'")
      machine.succeed(f"curl -f -b /tmp/cookies.txt -F 'file_name={small_file}' -F 'tag=beta' 'http://localhost:${port}/api/account/file/tag'")
      machine.succeed(f"curl -f -b /tmp/cookies.txt -F 'file_name={large_file}' -F 'tag=alpha' 'http://localhost:${port}/api/account/file/tag'")

      stats = api_get("/api/account/files/stats")
      assert sorted(stats["tags"]) == ["alpha", "beta"], f"Expected tags [alpha, beta], got {stats['tags']}"

      # Filter by tag
      alpha_files = api_get("/api/account/files?tag=alpha")
      assert alpha_files["count"] == 2, f"Expected 2 files with tag alpha, got {alpha_files['count']}"

      beta_files = api_get("/api/account/files?tag=beta")
      assert beta_files["count"] == 1, f"Expected 1 file with tag beta, got {beta_files['count']}"
      assert beta_files["files"][0]["FileName"] == small_file

      # Filter untagged (none should be untagged now since both have alpha)
      # Actually large_file has alpha, small_file has alpha+beta, so none are untagged
      untagged = api_get("/api/account/files?filter=untagged")
      assert untagged["count"] == 0, f"Expected 0 untagged files, got {untagged['count']}"

      # Filter by public/private
      public_files = api_get("/api/account/files?filter=public")
      assert public_files["count"] == 2, f"Expected 2 public files, got {public_files['count']}"

      # Toggle small_file to private
      machine.succeed(f"curl -f -b /tmp/cookies.txt -F 'file_name={small_file}' 'http://localhost:${port}/api/account/file/public'")

      private_files = api_get("/api/account/files?filter=private")
      assert private_files["count"] == 1, f"Expected 1 private file, got {private_files['count']}"
      assert private_files["files"][0]["FileName"] == small_file

      public_files = api_get("/api/account/files?filter=public")
      assert public_files["count"] == 1, f"Expected 1 public file after toggle, got {public_files['count']}"
      assert public_files["files"][0]["FileName"] == large_file

      # Delete a tag
      machine.succeed(f"curl -f -b /tmp/cookies.txt -X DELETE -F 'file_name={small_file}' -F 'tag=beta' 'http://localhost:${port}/api/account/file/tag'")

      stats = api_get("/api/account/files/stats")
      assert stats["tags"] == ["alpha"], f"Expected only [alpha] after deleting beta, got {stats['tags']}"

      # small_file still has alpha, so untagged should still be 0
      untagged = api_get("/api/account/files?filter=untagged")
      assert untagged["count"] == 0, f"Expected 0 untagged files, got {untagged['count']}"

      # Remove alpha from large_file, now it should be untagged
      machine.succeed(f"curl -f -b /tmp/cookies.txt -X DELETE -F 'file_name={large_file}' -F 'tag=alpha' 'http://localhost:${port}/api/account/file/tag'")

      untagged = api_get("/api/account/files?filter=untagged")
      assert untagged["count"] == 1, f"Expected 1 untagged file, got {untagged['count']}"
      assert untagged["files"][0]["FileName"] == large_file

      # Delete a file
      machine.succeed(f"curl -f -b /tmp/cookies.txt -X DELETE -F 'file_name={large_file}' 'http://localhost:${port}/api/account/file'")

      # Verify it no longer appears in the listing
      listing = api_get("/api/account/files")
      assert listing["count"] == 1, f"Expected 1 file after deletion, got {listing['count']}"
      assert listing["files"][0]["FileName"] == small_file

      # Verify the deleted file
      status = machine.succeed(f"curl -s -o /dev/null -w '%{{http_code}}' 'http://localhost:${port}/{large_file}'").strip()
      assert status == "307", f"Expected 307 redirect for deleted file, got {status}"

      stats = api_get("/api/account/files/stats")
      assert stats["count"] == 1, f"Expected 1 file in stats after deletion, got {stats['count']}"

      # Delete a file that has tags
      machine.succeed(f"curl -f -b /tmp/cookies.txt -X DELETE -F 'file_name={small_file}' 'http://localhost:${port}/api/account/file'")

      # Verify its deleted
      listing = api_get("/api/account/files")
      assert listing["count"] == 0, f"Expected 0 files after deleting tagged file, got {listing['count']}"
    '';
}
