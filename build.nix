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
      fetcherVersion = 3;
      hash = "sha256-+2YedGs7Oo/VYT8ibdnNAXJg/XHwR76c2C+3YlS04hA=";
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

  vendorHash = "sha256-0HEfw89bYgrY+s/WLlTPmo/4pF6VFW3k+krK9ZhO4r8=";

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
