{
  description = "Healthwatch - a Go + PostgreSQL CRUD API with a Vue 3 frontend";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    treefmt-nix.url = "github:numtide/treefmt-nix";
  };

  outputs =
    { self, nixpkgs, treefmt-nix }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          default = import ./nix/dev-shell.nix { inherit pkgs; };
        }
      );

      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          backend = import ./nix/package.nix { inherit pkgs; };
        in
        {
          backend = backend;
          container = import ./nix/container.nix {
            inherit pkgs;
            backendPackage = backend;
          };
          default = backend;
        }
      );

      formatter = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          treefmtEval = treefmt-nix.lib.evalModule pkgs ./nix/treefmt.nix;
        in
        treefmtEval.config.build.wrapper
      );

      checks = forAllSystems (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          treefmtEval = treefmt-nix.lib.evalModule pkgs ./nix/treefmt.nix;
        in
        {
          formatting = treefmtEval.config.build.check self;
        }
      );
    };
}
