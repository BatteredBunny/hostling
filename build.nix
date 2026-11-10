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
      hash = "sha256-XD0jXyf/RHex8RhQxIaV6UwI06Y8N9FWUZ16c9VcJtk=";
    };

    postBuild = ''
      pnpm run build
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

  passthru = {
    inherit frontend;
    frontendDemo = frontend.overrideAttrs {
      pname = "hostling-frontend-demo";
      postBuild = "pnpm run build:demo";
      installPhase = "cp -r dist-demo $out";
    };
  };

  excludedPackages = [
    "frontend"
    "cmd/loader"
    "cmd/openapi"
  ];

  checkFlags = [ "-trimpath" ];

  vendorHash = "sha256-L1E4vUMytZTzQ0Xz9U0Gzb9J6w2m1dEMVvHQARxPdqY=";

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
