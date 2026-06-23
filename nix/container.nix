{ pkgs, backendPackage }:

pkgs.dockerTools.buildLayeredImage {
  name = "healthwatch-backend";
  tag = "latest";

  # Uncompressed: syft/grype/dive all read "docker-archive:" as a plain
  # tar, and k3d/docker load both handle uncompressed tars fine too.
  # Gzip (the default) breaks the first three.
  compressor = "none";

  contents = [
    backendPackage
    pkgs.cacert # needed for TLS if DATABASE_URL ever points at a TLS-terminated Postgres
  ];

  config = {
    Entrypoint = [ "/bin/backend" ];
    # Required by the CDC: numeric UID 1000, not the distroless
    # convention's 65532 - no /etc/passwd entry needed either way since
    # this is a static Go binary that never calls getpwuid.
    User = "1000:1000";
    ExposedPorts = {
      "8080/tcp" = { };
    };
  };
}
