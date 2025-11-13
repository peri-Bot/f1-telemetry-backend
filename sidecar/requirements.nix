# This function now expects the Python package set AND the Nix standard library.
{ pythonPackages, lib }:

let
  # Use the more robust lib.splitString instead of builtins.split
  requirements = lib.splitString "\n" (builtins.readFile ./requirements.txt);

  # Use lib.filter and lib.hasPrefix, which are the correct functions.
  filteredRequirements = lib.filter (p: p != "" && !lib.hasPrefix "#" p) requirements;

  packages = map (p: pythonPackages.${builtins.toLower p}) filteredRequirements;
in
packages
