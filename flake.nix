{
  description = "F1 Telemetry Services (Nix-based Artifact Builder)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };

        # --- 1. Python Sidecar Definition ---

        livef1 = pkgs.python311Packages.buildPythonPackage rec {
          pname = "livef1"; # <-- CRITICAL: Match PyPI capitalization exactly
          version = "1.0.953";
          src = pkgs.fetchPypi {
            inherit pname version;
            # Put a fake hash first to get the real one from the error message
            sha256 = "sha256-3Me5SadodXEEI7QNy7QWhSplkG/OL2hyvaZ5N0Uz6sw=";
          };

          # Necessary because this package likely doesn't have a pyproject.toml
          # compatible with Nix's strict defaults.
          pyproject = true;
          build-system = [
            pkgs.python311Packages.setuptools
            pkgs.python311Packages.wheel
          ];

          postPatch = ''
            touch requirements.txt
          '';

          # We also likely need to manually add the dependencies since we just 
          # bypassed reading them from the file. 'requests' is a safe bet for 
          # API libraries, but we can add more if the app crashes at runtime.
          propagatedBuildInputs = [
            pkgs.python311Packages.requests
            pkgs.python311Packages.python-dateutil # <-- The fix for your error
            pkgs.python311Packages.jellyfish
            pkgs.python311Packages.numpy
            pkgs.python311Packages.pandas
            pkgs.python311Packages.setuptools
            pkgs.python311Packages.ujson
            pkgs.python311Packages.websockets
            pkgs.python311Packages.scipy
            pkgs.python311Packages.beautifulsoup4
          ];

          # Skip tests to avoid import errors during build
          doCheck = false;
        };

        pythonEnv = pkgs.python311.withPackages (ps: [
          ps.flask
          livef1
        ]);

        # Bundle the Sidecar into a "Runtime" directory
        sidecarRuntime = pkgs.symlinkJoin {
          name = "sidecar-runtime";
          paths = [ pythonEnv ./sidecar ];
          postBuild = ''
            mkdir -p $out/bin
            # Create a startup script that knows exactly where Python is
            cat > $out/bin/entrypoint <<EOF
            #!${pkgs.stdenv.shell}
            exec $out/bin/python $out/data_forwarder.py
            EOF
            chmod +x $out/bin/entrypoint
          '';
        };

        # --- 2. Go Backend Definition ---

        goBuilder = (pkgs.buildGoModule {
          pname = "f1-telemetry-service";
          version = "0.1.0";
          src = ./.;
          # Replace with your real Go vendor hash
          vendorHash = "sha256-0Qxw+MUYVgzgWB8vi3HBYtVXSq/btfh4ZfV/m1chNrA=";
        }).overrideAttrs (old: {
          preBuild = '' export CGO_ENABLED=0 '';
        });

        # Bundle the Backend into a "Runtime" directory
        backendRuntime = pkgs.symlinkJoin {
          name = "backend-runtime";
          paths = [ goBuilder ];
          postBuild = ''
            # We don't strictly need a script for Go, but we'll ensure 
            # the binary is in a standard location.
            mkdir -p $out/bin
          '';
        };

      in
      {
        packages = {
          backend = backendRuntime;
          sidecar = sidecarRuntime;
          # Default to backend if unspecified
          default = backendRuntime;
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go_1_24 pkgs.gopls pkgs.air pythonEnv ];
        };
      }
    );
}

