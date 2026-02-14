{
  description = "nprt - Nixpkgs PR Tracker";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs =
    inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];

      perSystem =
        {
          self',
          pkgs,
          ...
        }:
        let
          version = pkgs.lib.fileContents ./VERSION;
        in
        {
          packages.nprt-man = pkgs.stdenvNoCC.mkDerivation {
            pname = "nprt-man";
            inherit version;
            src = ./docs;

            nativeBuildInputs = with pkgs; [
              pandoc
              installShellFiles
            ];

            buildPhase = ''
              runHook preBuild
              pandoc USAGE.md -s -t man -o nprt.1
              runHook postBuild
            '';

            installPhase = ''
              runHook preInstall
              installManPage nprt.1
              runHook postInstall
            '';
          };

          packages.nprt = pkgs.buildGoModule {
            pname = "nprt";
            inherit version;
            src = ./.;

            vendorHash = null;

            subPackages = [ "cmd/nprt" ];

            ldflags = [
              "-X main.version=${version}"
            ];

            postInstall = ''
              mkdir -p $out/share/man/man1
              ln -sf ${self'.packages.nprt-man}/share/man/man1/nprt.1* $out/share/man/man1/
            '';

            meta = {
              description = "CLI tool to track which nixpkgs channels contain a given pull request";
              homepage = "https://github.com/thatsneat-dev/nprt";
              license = pkgs.lib.licenses.mit;
              mainProgram = "nprt";
              platforms = pkgs.lib.platforms.unix;
            };
          };

          devShells.default = pkgs.mkShellNoCC {
            packages = with pkgs; [
              go
              gofumpt
              alejandra
              statix
              deadnix
            ];
          };

          apps.nprt = {
            type = "app";
            program = "${self'.packages.nprt}/bin/nprt";
            meta.description = "CLI tool to track which nixpkgs channels contain a given pull request";
          };

          checks = {
            formatting = pkgs.runCommand "check-formatting" {
              nativeBuildInputs = with pkgs; [ alejandra ];
              src = ./.;
            } ''
              cd $src
              alejandra -c . 2>&1
              touch $out
            '';

            statix = pkgs.runCommand "check-statix" {
              nativeBuildInputs = with pkgs; [ statix ];
              src = ./.;
            } ''
              cd $src
              statix check .
              touch $out
            '';

            deadnix = pkgs.runCommand "check-deadnix" {
              nativeBuildInputs = with pkgs; [ deadnix ];
              src = ./.;
            } ''
              cd $src
              deadnix -f .
              touch $out
            '';
          };
        };
    };
}
