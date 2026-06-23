{
  description = "Healthwatch - a lightweight website health-check aggregator";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
  };

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs systems;
    in
    {
      devShells = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in {
          default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gotools
              golangci-lint
              gofumpt
              yamllint
              markdownlint-cli

              k3d
              kubectl
              kubernetes-helm
              argocd

              gitleaks
              syft
              grype
              dive

              just
              lefthook
              k6
            ];

            shellHook = ''
              echo "healthwatch dev shell — $(go version)"
              echo "run 'just' to list available tasks"
            '';
          };
        });

      packages = forAllSystems (system:
        let pkgs = nixpkgs.legacyPackages.${system};
        in rec {
          # The Go binary itself: CGO disabled, stripped, fully static.
          healthwatch = pkgs.buildGoModule {
            pname = "healthwatch";
            version = "0.1.0";
            src = ./.;

            # First build will fail with the expected hash printed in the
            # error message - paste it in here once go.sum is stable.
            vendorHash = "sha256-fGvqTqHF+WpspJ9Z+Vzle9ifjLdgAFjKHoLlVJsx+rg=";

            env.CGO_ENABLED = "0";
            ldflags = [ "-s" "-w" ];

            # The godog/api features tests need network or a running
            # instance and don't belong in a hermetic Nix build.
            doCheck = false;
          };

          # Distroless-equivalent image: scratch base, the static binary,
          # CA certificates (needed for the TLS checks against real
          # targets), and a non-root numeric user - no shell, no package
          # manager, nothing else.
          container = pkgs.dockerTools.buildLayeredImage {
            name = "healthwatch";
            tag = "latest";
            # Uncompressed: syft/grype/dive all read "docker-archive:" as a
            # plain tar, and k3d/docker load both handle uncompressed tars
            # fine too. Gzip (the default) breaks the first three.
            compressor = "none";
            contents = [
              healthwatch
              pkgs.cacert
            ];
            config = {
              Entrypoint = [ "/bin/healthwatch" ];
              Cmd = [ "-config" "/etc/healthwatch/targets.yaml" "-addr" ":8080" ];
              User = "65532:65532";
              ExposedPorts = { "8080/tcp" = { }; };
            };
          };

          default = healthwatch;
        });
    };
}
