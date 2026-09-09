{
  lib,
  fetchCrate,
  rustPlatform,
  nodejs_latest,
  pkg-config,
  openssl,
  stdenv,
  curl,
}:

rustPlatform.buildRustPackage rec {
  pname = "wasm-bindgen-cli";
  version = "0.2.127";

  src = fetchCrate {
    inherit pname version;
    hash = "sha256-di+qBAdd7pENLiIB9CoZoab+W5xeDoByMREcCGTSzWo=";
  };

  cargoHash = "sha256-FTv2GZIAQs0ePdIZXIXil7JbZ6kIT05VG6vqC1qNFxQ=";
  nativeBuildInputs = [ pkg-config ];
  buildInputs = [ openssl ] ++ lib.optionals stdenv.hostPlatform.isDarwin [ curl ];
  nativeCheckInputs = [ nodejs_latest ];

  # Tests require the wasm-bindgen monorepo.
  doCheck = false;

  meta = {
    description = "Facilitating high-level interactions between WASM modules and JavaScript";
    homepage = "https://wasm-bindgen.github.io/wasm-bindgen/";
    license = with lib.licenses; [ asl20 mit ];
    mainProgram = "wasm-bindgen";
  };
}
