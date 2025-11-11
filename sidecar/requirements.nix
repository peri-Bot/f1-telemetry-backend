{ pkgs ? import <nixpkgs> {} }:
let
  requirements = pkgs.lib.splitString "\n" (builtins.readFile ./requirements.txt);
  filteredRequirements = pkgs.lib.filter (p: p != "" && !pkgs.lib.hasPrefix "#" p) requirements;
  packages = map (p: pkgs.python3Packages.${builtins.toLower p}) filteredRequirements;
in
packages
