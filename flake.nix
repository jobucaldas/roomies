{
  description = "Roomies release baseline shells and checks";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";
  inputs.rust-overlay = {
    url = "github:oxalica/rust-overlay";
    inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, rust-overlay }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs {
        inherit system;
        overlays = [ (import rust-overlay) ];
        config = {
          allowUnfree = true;
          android_sdk.accept_license = true;
        };
      };
      rustToolchain = pkgs.rust-bin.stable."1.89.0".default.override {
        targets = [ "aarch64-linux-android" ];
      };
      cliRustPlatform = pkgs.makeRustPlatform {
        cargo = rustToolchain;
        rustc = rustToolchain;
      };
      wasmBindgenCli = pkgs.callPackage ./nix/wasm-bindgen-cli-0.2.127.nix {
        rustPlatform = cliRustPlatform;
      };
      dioxusCli = pkgs.callPackage ./nix/dioxus-cli.nix {
        rustPlatform = cliRustPlatform;
        wasm-bindgen-cli = wasmBindgenCli;
      };
      commonPackages = with pkgs; [
        rustToolchain
        curl
        git
        jq
      ];
      webPackages = with pkgs; [
        dioxusCli
        llvmPackages.lld
        go
        nodejs
        pkg-config
        postgresql
        sqlite
      ];
      desktopPackages = with pkgs; [
        atk
        cairo
        gdk-pixbuf
        glib
        gtk3
        libxkbcommon
        openssl
        pango
        webkitgtk_4_1
        xorg.libX11
        xorg.libXcursor
        xorg.libXi
        xorg.libXrandr
      ];
      androidSdk = pkgs.androidenv.composeAndroidPackages {
        buildToolsVersions = [ "34.0.0" ];
        includeNDK = true;
        includeEmulator = false;
        includeSystemImages = false;
        ndkVersion = "27.0.12077973";
        platformVersions = [ "34" ];
      };
      containerPackages = with pkgs; [
        buildah
        kustomize
        podman
        podman-compose
        kubectl
        skopeo
      ];
      ocrPackages = with pkgs; [
        imagemagick
        poppler_utils
        tesseract
      ];
      exportPackages = with pkgs; [
        ghostscript
        imagemagick
        pandoc
        qpdf
      ];
      e2ePackages = with pkgs; [
        chromium
        nodejs
      ];
      mkShell = packages: shellHook: pkgs.mkShell {
        inherit packages shellHook;
      };
    in
    {
      formatter.${system} = pkgs.nixpkgs-fmt;

      checks.${system} = {
        kustomize-production = pkgs.runCommand "roomies-kustomize-production" {
          nativeBuildInputs = [ pkgs.kustomize ];
        } ''
          cp -R ${./deploy/kustomize} ./kustomize-tree
          kustomize build ./kustomize-tree/overlays/production > $out
        '';

        readme-baseline = pkgs.runCommand "roomies-readme-baseline" {
          nativeBuildInputs = [ pkgs.gnugrep ];
        } ''
          grep -q "Supported platforms" ${./README.md}
          touch $out
        '';
      };

      devShells.${system} = {
        default = mkShell (commonPackages ++ webPackages) ''
          export ROOMIES_API_URL=/api
          export ROOMIES_PUBLIC_BASE_URL=http://localhost
          echo "Roomies web shell: cargo test -p roomies-app --no-default-features --features web"
        '';

        web = mkShell (commonPackages ++ webPackages) ''
          export ROOMIES_API_URL=/api
          export ROOMIES_PUBLIC_BASE_URL=http://localhost
        '';

        desktop = mkShell (commonPackages ++ webPackages ++ desktopPackages) ''
          export ROOMIES_API_URL=http://localhost:8080/api
          echo "Roomies desktop shell: cargo check -p roomies-app --no-default-features --features desktop"
        '';

        android = mkShell (commonPackages ++ webPackages ++ [ androidSdk.androidsdk pkgs.jdk21 ]) ''
          export ANDROID_HOME=${androidSdk.androidsdk}/libexec/android-sdk
          export ANDROID_SDK_ROOT=${androidSdk.androidsdk}/libexec/android-sdk
          export NDK_HOME=${androidSdk.androidsdk}/libexec/android-sdk/ndk/27.0.12077973
          export JAVA_HOME=${pkgs.jdk21}
          echo "Roomies Android shell: dx build --release --platform android --target aarch64-linux-android"
        '';

        e2e = mkShell (commonPackages ++ e2ePackages) ''
          export CHROMIUM_PATH=${pkgs.chromium}/bin/chromium
          export PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
          echo "Roomies e2e shell: browser automation tooling (system Chromium)"
        '';

        ocr = mkShell (commonPackages ++ ocrPackages) ''
          echo "Roomies OCR shell: image and text extraction tools"
        '';

        export = mkShell (commonPackages ++ exportPackages) ''
          echo "Roomies export shell: document and archive tooling"
        '';

        container = mkShell (commonPackages ++ containerPackages) ''
          echo "Roomies container shell: podman, compose, kubectl, and kustomize"
        '';
      };
    };
}
