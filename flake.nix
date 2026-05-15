{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    {
      self,
      nixpkgs,
      ...
    }:
    let
      inherit (nixpkgs) lib;

      systems = lib.systems.flakeExposed;

      forAllSystems = lib.genAttrs systems;

      nixpkgsFor = forAllSystems (
        system:
        import nixpkgs {
          inherit system;
        }
      );
    in
    {
      overlays.default = final: prev: {
        hostling = final.callPackage ./build.nix { };
      };

      nixosModules.default = import ./module.nix;

      checks = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          service-postgresql = pkgs.callPackage ./nix/test.nix {
            inherit self;
            dbType = "postgresql";
          };
          service-sqlite = pkgs.callPackage ./nix/test.nix {
            inherit self;
            dbType = "sqlite";
          };
          service-advanced = pkgs.callPackage ./nix/test-advanced.nix {
            inherit self;
          };
        }
      );

      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
          overlay = lib.makeScope pkgs.newScope (final: self.overlays.default final pkgs);
        in
        {
          inherit (overlay) hostling;
          default = overlay.hostling;

          # nix run .#test-service-postgresql.driverInteractive
          test-service-postgresql = pkgs.callPackage ./nix/test.nix {
            inherit self;
            dbType = "postgresql";
          };
          # nix run .#test-service-sqlite.driverInteractive
          test-service-sqlite = pkgs.callPackage ./nix/test.nix {
            inherit self;
            dbType = "sqlite";
          };
          # nix run .#test-service-advanced.driverInteractive
          test-service-advanced = pkgs.callPackage ./nix/test-advanced.nix {
            inherit self;
          };
          example-screenshots = pkgs.callPackage ./nix/example-screenshots.nix { inherit self; };
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};

          # TODO: upstream?
          atlas-unfree =
            let
              version = "v1.2.0";
            in
            pkgs.stdenvNoCC.mkDerivation {
              pname = "atlas";
              inherit version;
              src = pkgs.fetchurl {
                url =
                  if pkgs.stdenv.isLinux && pkgs.stdenv.isx86_64 then
                    "https://release.ariga.io/atlas/atlas-linux-amd64-${version}"
                  else
                    throw "Unsupported system for Atlas: ${system}";
                hash = "sha256-wSuInjNJ8OVhCuwy/jJ+WmkRoORydUoMOBwwx8BjDog=";
              };
              dontUnpack = true;
              installPhase = ''
                mkdir -p $out/bin
                install -m755 $src $out/bin/atlas
              '';
            };
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              cloudflared # cloudflared tunnel --url localhost:8081
              go
              wire
              atlas-unfree # Unfree version of atlas
              sqlite
              pnpm_10
              nodejs
              just

              # hot reloading during development
              # air -- -c examples/example_local_sqlite.toml
              air
              golangci-lint
              golines
            ];
          };
        }
      );
    };
}
