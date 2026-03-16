{
  buildGoModule,
  fetchPnpmDeps,
  pnpmConfigHook,
  pnpm_10,
  nodejs,
  stdenv,
  lib,
}:
let
  pnpm = pnpm_10;
  version = "0.3.1";

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
      fetcherVersion = 3;
      hash = "sha256-qZiK8RQVlblR6bM1W4rIn8J6s3rMKShQpPhomGWFrsY=";
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
  name = "hostling";
  inherit version;

  src = ./.;

  vendorHash = "sha256-AEqtKBnmwnbbGcYPvlv/5noQgJaZJad8rR3gWXFFEQY=";

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
