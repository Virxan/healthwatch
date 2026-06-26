{pkgs}:
pkgs.mkShell {
  packages = with pkgs; [
    # Go backend
    go
    gotools
    golangci-lint
    gofumpt

    # Frontend
    nodejs_22

    # Local Postgres client (the server itself runs via docker-compose -
    # see `task db` / docker-compose.yml)
    postgresql

    # Kubernetes / GitOps
    k3d
    kubectl
    kubernetes-helm
    argocd

    # Security / supply chain
    gitleaks
    syft
    grype
    dive

    # Task runner & git hooks
    go-task
    lefthook

    # Testing
    k6
    hurl

    # Misc linters
    markdownlint-cli
    shellcheck
  ];

  shellHook = ''
    echo "healthwatch dev shell — go $(go version | cut -d' ' -f3), node $(node --version)"
    echo "run 'task' to list available tasks"
  '';
}
