{
  description = "Agentic Issue Searcher Environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          devShells = {
            default = self.devShells.${system}.go;

            go = pkgs.mkShell {
              buildInputs = [
                pkgs.go
                pkgs.gopls
              ];

              shellHook = ''
                echo " go: $(go version)"
              '';
            };
          };
        }
    );
}
