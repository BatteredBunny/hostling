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
          service = pkgs.callPackage ./test.nix {
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

          # nix run .#test-service.driverInteractive
          test-service = pkgs.callPackage ./test.nix {
            inherit self;
          };
        }
      );

      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};

          # TODO: make it non hacky
          atlas-unfree = pkgs.stdenvNoCC.mkDerivation {
            pname = "atlas";
            version = "latest";
            src = pkgs.fetchurl {
              url =
                if pkgs.stdenv.isLinux && pkgs.stdenv.isx86_64 then
                  "https://release.ariga.io/atlas/atlas-linux-amd64-latest"
                else if pkgs.stdenv.isLinux && pkgs.stdenv.isAarch64 then
                  "https://release.ariga.io/atlas/atlas-linux-arm64-latest"
                else if pkgs.stdenv.isDarwin && pkgs.stdenv.isx86_64 then
                  "https://release.ariga.io/atlas/atlas-darwin-amd64-latest"
                else if pkgs.stdenv.isDarwin && pkgs.stdenv.isAarch64 then
                  "https://release.ariga.io/atlas/atlas-darwin-arm64-latest"
                else
                  throw "Unsupported system for Atlas: ${system}";
              hash = "sha256-0Er3V5Fiud87KUPin22xb34iJmVgnQv1hJ9NQXXvrY8=";
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

              # hot reloading during development
              # air -- -c examples/example_local_sqlite.toml
              air
            ];
          };
        }
      );
    };
}
