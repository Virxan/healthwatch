{pkgs}: let
  # Built in isolation from the backend - vite.config.js's outDir
  # ("../backend/web/dist") is a local-dev convenience and doesn't exist
  # inside this sandboxed derivation, so --outDir overrides it back to a
  # plain local dist/ here. The backend derivation below copies that
  # output into backend/web/dist itself before compiling, so go:embed
  # picks up the real build instead of the placeholder index.html.
  frontend = pkgs.buildNpmPackage {
    pname = "healthwatch-frontend";
    version = "0.1.0";
    src = ../frontend;

    # First build fails with the expected hash printed in the error -
    # paste it in here once package-lock.json is stable.
    npmDepsHash = "sha256-mS4KPD9I/YotGQW6D+B5NDl7ZqRVhHpbBPCN8r7N/u8=";

    buildPhase = ''
      runHook preBuild
      npx vite build --outDir dist
      runHook postBuild
    '';

    installPhase = ''
      runHook preInstall
      mkdir -p "$out"
      cp -r dist/. "$out/"
      runHook postInstall
    '';
  };

  # The backend's own source tree, but with web/dist replaced by the
  # real frontend build instead of the committed placeholder.
  backendSrc = pkgs.runCommand "healthwatch-backend-src" {} ''
    cp -r ${../backend} "$out"
    chmod -R u+w "$out"
    rm -rf "$out/web/dist"
    mkdir -p "$out/web/dist"
    cp -r ${frontend}/. "$out/web/dist/"
  '';
in
  pkgs.buildGoModule {
    pname = "healthwatch-backend";
    version = "0.1.0";
    src = backendSrc;

    # Same deal as npmDepsHash above: first build prints the real hash.
    vendorHash = "sha256-+twj0uHxCAOagiUifXQCFEyCGXmpGp6hwS8vFT2onA4=";

    env.CGO_ENABLED = "0";
    ldflags = ["-s" "-w"];

    # tests/integration needs Docker (Testcontainers), which isn't
    # available inside a sandboxed Nix build - it's not part of `nix
    # build`/`nix flake check`, run it explicitly via `task
    # test-integration` instead.
    doCheck = false;

    meta.mainProgram = "backend";
  }
