{
  pkgs ? import <nixpkgs> { },
  compile ? false,
  ...
}:
let
  inherit (pkgs) lib mkShell buildGoModule;
  inherit (lib) fetchFromGitHub cleanSource;

  src = cleanSource ./.;

  package = buildGoModule {
    name = "kittenMQ-consumer";
    inherit src;
    # proxyVendor = true;
    vendorHash = "sha256-XE5npxjcTRDmINM2IFS4C9NWfsAYiGs+h4sDIZX8AhU=";

    postInstall = ''
      mv $out/bin/rh-api $out/bin/kittenMQ-consumer
    '';

  };
in
mkShell { packages = (with pkgs; [ go ]) ++ lib.optional (compile) package; }
