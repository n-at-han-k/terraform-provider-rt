{
  # The shell the generator and the provider share: openapi-generator writes
  # the Go, Go builds it, tofu runs it.
  description = "Request Tracker Terraform provider, generated from the REST2 OpenAPI description";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, utils }:
    (utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};

        # The one hook a template cannot reach: which operations are one
        # resource. javac against the CLI's own jar and an SPI entry -- no
        # Maven, no checkout of the generator.
        rt-codegen = pkgs.stdenv.mkDerivation {
          name = "rt-terraform-codegen";
          src = ./generators/rt;

          nativeBuildInputs = [ pkgs.jdk ];

          buildPhase = ''
            mkdir -p classes
            javac -nowarn -proc:none \
              -cp ${pkgs.openapi-generator-cli}/share/java/openapi-generator-cli.jar \
              -d classes $(find src -name '*.java')
            cp -r resources/. classes/
            jar cf rt-codegen.jar -C classes .
          '';

          installPhase = ''
            install -Dm644 rt-codegen.jar $out/share/java/rt-codegen.jar
          '';
        };

        # The packaged CLI runs `java -jar`, which ignores -cp; a generator on
        # the classpath needs the main class named.
        openapi-generator-rt = pkgs.writeShellApplication {
          name = "openapi-generator-rt";
          runtimeInputs = [ pkgs.jre ];
          text = ''
            exec java -cp ${rt-codegen}/share/java/rt-codegen.jar:${pkgs.openapi-generator-cli}/share/java/openapi-generator-cli.jar \
              org.openapitools.codegen.OpenAPIGenerator "$@"
          '';
        };

      in
      {
        packages = { inherit rt-codegen openapi-generator-rt; };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            gopls

            # The patched generator (`-g rt-terraform`), which groups the
            # document's operations into resources.
            openapi-generator-rt

            # And upstream's, unpatched, for looking at what stock
            # `-g terraform-provider` does with the same document.
            openapi-generator-cli

            # terraform itself is BUSL and unfree; tofu runs the same provider.
            opentofu
          ];

          # A Terraform provider is pure Go, and cgo only costs a C compiler.
          shellHook = ''
            export CGO_ENABLED=0
          '';
        };
      }));
}
