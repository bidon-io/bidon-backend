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
            pkgs.git-spice
            pkgs.gh
          ];

          # git-spice's package only ships the `gs` binary; this silences its "use git-spice" warning.
          GIT_SPICE_NO_GS_WARNING = "1";
        };

      }
    );
}
