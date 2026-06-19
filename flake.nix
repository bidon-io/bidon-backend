{
  description = "BidOn development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {

        devShells.default = pkgs.mkShellNoCC {
          packages = [
            pkgs.go
            pkgs.golangci-lint
            pkgs.just
            pkgs.buf
            pkgs.pre-commit
          ];

          shellHook = '' [ ! -f .env ] && cp .env.sample .env.local && ln -s .env.local .env '';
        };

      }
    );
}
