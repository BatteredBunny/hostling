{
  buildGoLatestModule,
  fetchPnpmDeps,
  pnpmConfigHook,
  pnpm_12,
  nodejs,
  stdenv,
  lib,
}:
let
  pnpm = pnpm_12;
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
      fetcherVersion = 5;
      hash = "sha256-JajsH9oh9Tmf461px2W7YIokVCDwXnmIPpHW+cxcvpY=";
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

  vendorHash = "sha256-o4Ho+w2RJ0ytlPixmwVYkLILqsY7XHzC1K2I0odWthQ=";

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
