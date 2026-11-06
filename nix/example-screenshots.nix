{
  self,
  pkgs,
  lib,
}:
let
  hostling = self.packages.${pkgs.stdenv.hostPlatform.system}.hostling;
in
pkgs.stdenv.mkDerivation {
  name = "hostling-screenshots";
  src = ../frontend;

  nativeBuildInputs = with pkgs; [
    nodejs
    pnpm_11
    pnpmConfigHook
    chromium
  ];

  pnpmDeps = hostling.frontend.pnpmDeps;

  env = {
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1";
    PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = lib.getExe pkgs.chromium;
    XDG_CONFIG_HOME = "/tmp/hostling-screenshots/config";
    XDG_CACHE_HOME = "/tmp/hostling-screenshots/cache";
  };

  buildPhase = ''
    runHook preBuild

    mkdir -p "$XDG_CONFIG_HOME" "$XDG_CACHE_HOME"
    cp -r ${hostling.frontendDemo} dist-demo
    chmod -R u+w dist-demo
    pnpm exec playwright test

    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall

    mkdir -p "$out/screenshots"
    cp screenshots/*.png "$out/screenshots/"

    runHook postInstall
  '';
}
