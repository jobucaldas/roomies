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
  version = "0.2.126";

  src = fetchCrate {
    inherit pname version;
    hash = "sha256-H6Is3fiZVxZCfOMWK5dWMSrtn50VGv0sfdnsT+cTtyk=";
  };

  cargoHash = "sha256-VucqkXbCi4qtQzY/HrXiDnbSURsagPsdNVMn1Tw3UiY=";
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
