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
          service = pkgs.callPackage ./test.nix { };
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
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [
              cloudflared # cloudflared tunnel --url localhost:8081
              go
              wire
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
