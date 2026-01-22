{
  pkgs,
  config ? pkgs.config,
  lib ? pkgs.lib,
  ...
}:
let
  cfg = config.services.hostling;
  toml = pkgs.formats.toml { };
  tomlSetting = toml.generate "config.toml" cfg.settings;
in
{
  options.services.hostling = {
    enable = lib.mkEnableOption "hostling";

    package = lib.mkOption {
      description = "package to use";
      default = pkgs.callPackage ./build.nix { };
    };

    openFirewall = lib.mkEnableOption "" // {
      description = "Open service port in firewall.";
      default = false;
    };

    createDbLocally = lib.mkEnableOption "creation of database on the instance";

    environmentFile = lib.mkOption {
      type = lib.types.nullOr lib.types.path;
      default = null;
      description = "Used for specifying GITHUB_CLIENT_ID and GITHUB_SECRET";
    };

    settings = {
      port = lib.mkOption {
        type = lib.types.int;
        apply = toString;
        description = "port to run service on";
        default = 8872;
      };

      branding = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Branding name for the instance";
      };

      tagline = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Tagline for the instance, used for meta description and homepage";
      };

      database_type = lib.mkOption {
        type = lib.types.enum [
          "postgresql"
          "sqlite"
        ];
        default = "postgresql";
        example = "sqlite";
        description = "Database type";
      };

      database_connection_url = lib.mkOption {
        type = lib.types.str;
        default = "";
        description = "Database connection string";
      };

      max_upload_size = lib.mkOption {
        type = lib.types.int;
        default = 104857600;
        description = "Max upload size in bytes";
      };

      data_folder = lib.mkOption {
        type = lib.types.path;
        default = "/var/lib/hostling/data";
        description = "Folder to store local image data in";
      };

      behind_reverse_proxy = lib.mkOption {
        type = lib.types.bool;
        default = false;
        example = true;
        description = "Allows using trusted proxy settings";
      };

      trusted_proxy = lib.mkOption {
        type = lib.types.str;
        default = "";
        example = "127.0.0.1";
        description = "Which proxy to trust for IP information";
      };

      public_url = lib.mkOption {
        type = lib.types.str;
        default = "";
        example = "https://cdn.example.com";
        description = "Public url that its hosted on";
      };

      # s3 = {
      #   access_key_id
      #   secret_access_key
      #   bucket
      #   region
      #   endpoint
      # };
    };
  };

  config = lib.mkIf cfg.enable {
    systemd.services.hostling = {
      enable = true;
      serviceConfig = {
        User = "hostling";
        Group = "hostling";
        ProtectSystem = "full";
        ProtectHome = "yes";
        DeviceAllow = [ "" ];
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        PrivateDevices = true;
        ProtectClock = true;
        ProtectControlGroups = true;
        ProtectHostname = true;
        ProtectKernelLogs = true;
        ProtectKernelModules = true;
        ProtectKernelTunables = true;
        ProtectProc = "invisible";
        RestrictNamespaces = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        SystemCallArchitectures = "native";
        PrivateUsers = true;
        StateDirectory = "hostling";
        EnvironmentFile = cfg.environmentFile;
        ExecStart = "${lib.getExe cfg.package} -c=${tomlSetting}";
        Restart = "always";
      };

      environment.GIN_MODE = "release";
      wantedBy = [ "default.target" ];

      after = lib.mkIf (cfg.settings.database_type == "postgresql") [ "postgresql.service" ];
      requires = lib.mkIf (cfg.settings.database_type == "postgresql") [ "postgresql.service" ];
    };

    services.hostling = lib.mkIf (cfg.createDbLocally && cfg.settings.database_type == "postgresql") {
      settings.database_connection_url = "postgresql:///hostling?host=/run/postgresql&user=hostling";
    };

    services.postgresql = lib.mkIf (cfg.createDbLocally && cfg.settings.database_type == "postgresql") {
      enable = true;
      ensureDatabases = [ "hostling" ];
      ensureUsers = [
        {
          name = "hostling";
          ensureDBOwnership = true;
        }
      ];
    };

    users.users.hostling = {
      isSystemUser = true;
      group = "hostling";
    };

    users.groups.hostling = { };

    networking.firewall = lib.mkIf cfg.openFirewall {
      allowedTCPPorts = [ (builtins.fromJSON cfg.settings.port) ];
    };
  };
}
