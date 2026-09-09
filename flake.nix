{
  description = "samber-linter: static analyzer that finds health-washing in samber/do v2 containers";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self
    , nixpkgs
    , flake-utils
    ,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
        goVersion = pkgs.go_1_26 or pkgs.go;

        samber-linter = pkgs.buildGoModule {
          pname = "samber-linter";
          version = "0.1.0";
          src = ./.;
          vendorHash = null; # deps are vendored
          doCheck = true;
        };
      in
      {
        packages.default = samber-linter;
        packages.samber-linter = samber-linter;

        apps.default = flake-utils.lib.mkApp { drv = samber-linter; };

        devShells.default = pkgs.mkShell {
          buildInputs = [
            goVersion
            pkgs.golangci-lint
          ];
        };

        checks.build = samber-linter;
        checks.test =
          pkgs.runCommand "samber-linter-test"
            {
              nativeBuildInputs = [ goVersion ];
              src = ./.;
            }
            ''
              export GOCACHE="$TMPDIR/go-cache"
              export GOPATH="$TMPDIR/gopath"
              cp -r "$src" work
              chmod -R u+w work
              cd work
              go test ./...
              touch "$out"
            '';
        checks.lint =
          pkgs.runCommand "samber-linter-lint"
            {
              nativeBuildInputs = [ goVersion pkgs.golangci-lint ];
              src = ./.;
            }
            ''
              export GOCACHE="$TMPDIR/go-cache"
              export GOPATH="$TMPDIR/gopath"
              cp -r "$src" work
              chmod -R u+w work
              cd work
              golangci-lint run
              touch "$out"
            '';
        checks.flake = self.packages.${system}.default;
      }
    );
}
