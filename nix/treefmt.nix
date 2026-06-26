{pkgs, ...}: {
  projectRootFile = "flake.nix";

  programs.alejandra.enable = true; # formats *.nix files
}
