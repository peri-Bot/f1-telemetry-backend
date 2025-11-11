{
  description = "Nix flake for the F1 Telemetry Go Backend Service (using native Nix functions)";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      forAllSystems = function: nixpkgs.lib.genAttrs supportedSystems (system:
        let
          pkgs = import nixpkgs { inherit system; };
          pythonVersion = pkgs.python311;

          # --- DEFINE IT ONCE HERE ---
          # This pythonSidecarEnv will be available to both devShells and packages.
          pythonSidecarEnv = pythonVersion.withPackages (ps: [
            (pkgs.callPackage ./sidecar/requirements.nix { })
          ]);

        in
        function { inherit pkgs pythonSidecarEnv; } # Pass them into the function
      );

    in
    {
      devShells = forAllSystems ({ pkgs, pythonSidecarEnv }: {
        # <-- Receive it here
        default = pkgs.mkShell {
          packages = [
            pkgs.go_1_24
            pkgs.gopls
            pkgs.air
            pythonSidecarEnv # <-- Reuse it here
          ];
          shellHook = ''
            echo "Entered F1 Telemetry Backend dev environment."
          '';
        };
      });

      packages = forAllSystems ({ pkgs, pythonSidecarEnv }: # <-- And also receive it here
        let
          backend = pkgs.buildGoModule {
            pname = "f1-telemetry-service";
            version = "0.1.0";
            src = ./.;
            vendorHash = pkgs.lib.fakeSha256;
          };

          entrypoint = pkgs.writeShellScriptBin "entrypoint" ''
            #!${pkgs.stdenv.shell}
            echo "Starting Python sidecar in the background..."
            ${pythonSidecarEnv}/bin/python ./sidecar/data_forwarder.py &
            echo "Starting Go backend service..."
            exec ${backend}/bin/f1-telemetry-service
          '';

        in
        {
          container = pkgs.dockerTools.buildImage {
            name = "f1-telemetry-service";
            tag = "latest";
            copyToRoot = pkgs.buildEnv {
              name = "image-root";
              paths = [ backend entrypoint pythonSidecarEnv ./sidecar ];
            };
            config = {
              Cmd = [ "${entrypoint}/bin/entrypoint" ];
              ExposedPorts = { "8080/tcp" = { }; };
              Env = [ "PORT=8080" "SIDECAR_API_URL=http://localhost:5000/data" ];
            };
          };

          default = self.packages.x86_64-linux.container;
        });
    };
}
