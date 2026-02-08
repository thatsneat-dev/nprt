{
  description = "nprt - NixPkgs PR Tracker";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem = { config, self', pkgs, system, ... }:
        let
          version = builtins.readFile ./VERSION;
        in
        {
          packages.nprt-man = pkgs.stdenvNoCC.mkDerivation {
            pname = "nprt-man";
            inherit version;
            src = ./docs;

            nativeBuildInputs = [ pkgs.pandoc ];

            buildPhase = ''
              pandoc USAGE.md -s -t man -o nprt.1
            '';

            installPhase = ''
              install -Dm644 nprt.1 $out/share/man/man1/nprt.1
            '';
          };

          packages.default = pkgs.buildGoModule {
            pname = "nprt";
            inherit version;
            src = ./.;

            vendorHash = null;

            ldflags = [
              "-X main.version=${version}"
            ];

            postInstall = ''
              install -Dm644 ${self'.packages.nprt-man}/share/man/man1/nprt.1.gz $out/share/man/man1/nprt.1.gz
            '';

            meta = with pkgs.lib; {
              description = "CLI tool to track which nixpkgs channels contain a given pull request";
              homepage = "https://github.com/taylrfnt/nixpkgs-pr-tracker";
              license = licenses.mit;
              maintainers = [ ];
            };
          };

          devShells.default = pkgs.mkShell {
            buildInputs = with pkgs; [
              go
              gofumpt
              pkg-config
            ];
          };

          apps.default = {
            type = "app";
            program = "${self'.packages.default}/bin/nprt";
          };
        };
    };
}
