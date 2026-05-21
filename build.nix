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
      fetcherVersion = 3;
      hash = "sha256-/RiPI2FNS4XC0GaMOgl+bzaeYsaWRp7jLuObAiSgEtE=";
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

  vendorHash = "sha256-ZLozZWXhbqHMGslNFgA0YYGy0YvxD124A4ONSmbU49c=";

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
