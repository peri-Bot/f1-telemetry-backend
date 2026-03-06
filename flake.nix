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
          ps.grpcio
          ps.grpcio-tools
          ps.grpcio-health-checking
          livef1
        ]);

        # Bundle the Sidecar into a "Runtime" directory
        sidecarRuntime = pkgs.symlinkJoin {
          name = "sidecar-runtime";
          paths = [ pythonEnv ./sidecar ];
          nativeBuildInputs = [ pkgs.protobuf ];
          postBuild = ''
            mkdir -p $out/bin

            # Remove the symlinked proto dir (prevents read-only errors if it exists locally)
            rm -rf $out/proto
            
            # Generate Python proto stubs
            mkdir -p $out/proto
            ${pythonEnv}/bin/python -m grpc_tools.protoc \
              -I ${./proto} \
              --python_out=$out/proto \
              --grpc_python_out=$out/proto \
              ${./proto}/telemetry.proto

            # Fix absolute → relative import
            sed -i 's/^import telemetry_pb2 as/from . import telemetry_pb2 as/' $out/proto/telemetry_pb2_grpc.py
            touch $out/proto/__init__.py

            # Create a startup script that knows exactly where Python is
            cat > $out/bin/sidecar-runtime <<EOF
            #!${pkgs.stdenv.shell}
            exec $out/bin/python $out/data_forwarder.py
            EOF
            chmod +x $out/bin/sidecar-runtime
          '';
        };

        # --- 2. Go Backend Definition ---

        goBuilder = pkgs.buildGoModule {
          pname = "f1-telemetry-service";
          version = "0.1.0";
          src = ./.;

          nativeBuildInputs = [
            pkgs.protobuf
            pkgs.protoc-gen-go
            pkgs.protoc-gen-go-grpc
          ];

          # Generate stubs right before `go build` runs
          preBuild = ''
            echo "Generating Go protobuf stubs..."
            mkdir -p proto/gen/telemetrypb
            protoc -I proto \
              --go_out=proto/gen/telemetrypb --go_opt=paths=source_relative \
              --go-grpc_out=proto/gen/telemetrypb --go-grpc_opt=paths=source_relative \
              proto/telemetry.proto
              
            export CGO_ENABLED=0
          '';

          vendorHash = "sha256-EvC5AtO90A3HI3oPsoKmsspwsnytS00GPJ8LDxXz3h0=";
        };

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
          packages = [
            pkgs.go_1_24
            pkgs.gopls
            pkgs.air
            pkgs.golangci-lint
            pkgs.protobuf
            pkgs.protoc-gen-go
            pkgs.protoc-gen-go-grpc
            pythonEnv
          ];
        };
      }
    );
}


