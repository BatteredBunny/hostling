{
  buildGoLatestModule,
  fetchPnpmDeps,
  pnpmConfigHook,
  pnpm_11,
  nodejs,
  stdenv,
  lib,
}:
let
  pnpm = pnpm_11;
  version = "0.6.0";

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
      hash = "sha256-7SU/76ZAnhULH33miS9LafGcKnQa1PSCOg/d37ArQWA=";
    };

    postBuild = ''
      pnpm run build
    '';

    doCheck = true;
    checkPhase = ''
      runHook preCheck
      pnpm run lint
      runHook postCheck
    '';

    installPhase = ''
      cp -r ../public/dist $out
    '';
  };
in
buildGoLatestModule {
  pname = "hostling";
  inherit version;

  src = ./.;

  passthru = { inherit frontend; };

  excludedPackages = [ "frontend" ];

  vendorHash = "sha256-zHdH/l/CtstwvgCvwDuJClANyUtzZ55YiNcOx+sGUEc=";

  prePatch = ''
    mkdir -p ./public/dist
    cp -r ${frontend}/. ./public/dist/
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
