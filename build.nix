{
  buildGoModule,
  fetchPnpmDeps,
  pnpmConfigHook,
  pnpm_10,
  nodejs,
  stdenv,
}:
let
  pnpm = pnpm_10;
  version = "0.2.1";

  frontend = stdenv.mkDerivation {
    pname = "hostling-frontend";
    inherit version;

    src = ./cmd/frontend;

    nativeBuildInputs = [
      nodejs
      pnpmConfigHook
      pnpm
    ];

    pnpmDeps = fetchPnpmDeps {
      pname = "hostling-frontend";
      inherit version pnpm;
      src = ./cmd/frontend;
      fetcherVersion = 3;
      hash = "sha256-eKImv6YgN8Ebl0ZWLTqbTpcYrwGvLPSNDapsJscPlr0=";
    };

    postBuild = ''
      pnpm run build
    '';

    installPhase = ''
      cp -r ../public/dist/* $out
    '';
  };
in
buildGoModule {
  name = "hostling";
  inherit version;

  src = ./.;

  vendorHash = "sha256-sNQH/mKPEmeU1OVT7SadIyDGB9p57GpefYVMN804U8Y=";

  prePatch = ''
    cp -r ${frontend} ./cmd/public/dist
  '';

  ldflags = [
    "-s"
    "-w"
  ];

  meta = {
    description = "Simple file hosting service";
    homepage = "https://github.com/BatteredBunny/hostling";
    mainProgram = "hostling";
  };
}
