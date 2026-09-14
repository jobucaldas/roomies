{
  lib,
  fetchCrate,
  rustPlatform,
  pkg-config,
  cacert,
  openssl,
  rustfmt,
  installShellFiles,
  makeWrapper,
  esbuild,
  wasm-bindgen-cli,
}:

rustPlatform.buildRustPackage rec {
  pname = "dioxus-cli";
  version = "0.7.10";

  src = fetchCrate {
    inherit pname version;
    hash = "sha256-kPzo5zRSVs46SjiDRKpKxca8kPcWUgqc/LMKQsk0sC8=";
  };

  cargoHash = "sha256-cvBVIkIqBjXFifYNpL2DqZpQcBaX/59Xw0ZJKUvUcIs=";
  buildFeatures = [
    "no-downloads"
    "disable-telemetry"
  ];

  OPENSSL_NO_VENDOR = 1;

  nativeBuildInputs = [
    pkg-config
    cacert
    installShellFiles
    makeWrapper
  ];

  buildInputs = [ openssl ];
  nativeCheckInputs = [ rustfmt ];

  checkFlags = [
    # Requires network access.
    "--skip=serve::proxy::test"
    # Requires the Dioxus monorepo and mobile toolchains.
    "--skip=test_harnesses::run_harness"
  ];

  postInstall = ''
    installShellCompletion --cmd dx \
      --bash <($out/bin/dx completions bash) \
      --fish <($out/bin/dx completions fish) \
      --zsh <($out/bin/dx completions zsh)
  '';

  postFixup = ''
    wrapProgram $out/bin/dx \
      --suffix PATH : ${lib.makeBinPath [ esbuild wasm-bindgen-cli ]}
  '';

  meta = {
    description = "CLI for building fullstack web, desktop, and mobile apps with a single codebase";
    homepage = "https://dioxus.dev";
    changelog = "https://github.com/DioxusLabs/dioxus/releases";
    license = with lib.licenses; [ mit asl20 ];
    mainProgram = "dx";
  };
}
