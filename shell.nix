let
  nixpkgs = fetchTarball "https://github.com/NixOS/nixpkgs/tarball/nixos-26.05";
  pkgs = import nixpkgs {
    config = {};
    overlays = [];
  };
in
  pkgs.mkShell {
    shellHook = ''
      export PATH="$PWD/frontend/node_modules/.bin:$PATH"
    '';
    buildInputs = with pkgs; [
      go
      golangci-lint
      goreleaser
      nilaway
      just
      just-lsp
      nil
      nixd
      govulncheck
      air
      delve
      markdownlint-cli2
      sql-formatter
      oapi-codegen
      sqlc
      air
      gow
      zellij
    ];
  }
