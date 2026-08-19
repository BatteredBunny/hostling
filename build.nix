{
  buildGoModule,
  fetchPnpmDeps,
  pnpmConfigHook,
  pnpm_11,
  nodejs,
  stdenv,
  lib,
}:
let
  pnpm = pnpm_11;
  version = "0.4.0";

  frontend = stdenv.mkDerivation {
    pname = "hostling-frontend";
    inherit version;

    src = ./frontend;

    nativeBuildInputs = [
      nodejs
      pnpmConfigHook
      pnpm
    ];

    pnpmDeps = fetchPnpmDeps {
      pname = "hostling-frontend";
      inherit version pnpm;
      src = ./frontend;
      fetcherVersion = 4;
      hash = "sha256-C2VFOfoX2O44BX3otOvKHnQ0luoDbIbd84lKsN7Z2WI=";
    };

    postBuild = ''
      pnpm run build
    '';

    installPhase = ''
      cp -r ../public/dist $out
    '';
  };
in
buildGoModule {
  pname = "hostling";
  inherit version;

  src = ./.;

  vendorHash = "sha256-wlexKHC2/GXPsnjQmtbY8SE5qHDDu4dgOd8TFGs+X6M=";

  prePatch = ''
    cp -r ${frontend} ./public/dist
  '';

  ldflags = [
    "-s"
    "-w"
  ];

  meta = {
    description = "Simple file hosting service";
    homepage = "https://github.com/BatteredBunny/hostling";
    license = lib.licenses.mit;
    mainProgram = "hostling";
  };
}
