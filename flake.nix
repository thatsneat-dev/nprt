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
          packages.default = pkgs.buildGoModule {
            pname = "nprt";
            inherit version;
            src = ./.;

            vendorHash = null;

            ldflags = [
              "-X main.version=${version}"
            ];

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
