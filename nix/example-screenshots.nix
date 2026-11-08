# Captures the demo into $out/screenshots. With screenshotBaseline set to an
# earlier capture, compares against it instead and writes $out/changes.
{
  self,
  pkgs,
  lib,
  screenshotBaseline ? null,
}:
let
  hostling = self.packages.${pkgs.stdenv.hostPlatform.system}.hostling;
in
pkgs.stdenv.mkDerivation {
  name = if screenshotBaseline == null then "hostling-screenshots" else "hostling-screenshot-changes";
  src = ../frontend;

  nativeBuildInputs = with pkgs; [
    nodejs
    pnpm_11
    pnpmConfigHook
    chromium
  ];

  pnpmDeps = hostling.frontend.pnpmDeps;

  env = {
    FONTCONFIG_FILE = pkgs.makeFontsConf {
      fontDirectories = [ pkgs.dejavu_fonts ];
    };
    PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD = "1";
    PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH = lib.getExe pkgs.chromium;
    XDG_CONFIG_HOME = "/tmp/hostling-screenshots/config";
    XDG_CACHE_HOME = "/tmp/hostling-screenshots/cache";
  }
  // lib.optionalAttrs (screenshotBaseline != null) {
    SCREENSHOT_BASELINE = "${screenshotBaseline}/screenshots";
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

    mkdir -p "$out"
    cp -r screenshot-report "$out/report"
    ${
      if screenshotBaseline == null then
        ''cp -r screenshots "$out/screenshots"''
      else
        ''cp -r screenshot-changes "$out/changes"''
    }

    runHook postInstall
  '';
}
