FROM docker.io/rust:1.88-alpine

RUN apk add --no-cache \
    git \
    curl \
    make \
    gcc \
    musl-dev \
    sqlite-dev \
    postgresql-client \
    go \
    openssl-dev \
    pkgconf \
    nodejs \
    npm \
    && rustup target add wasm32-unknown-unknown \
    && cargo install dioxus-cli --version 0.7.9 --locked

WORKDIR /workspace
CMD ["sleep", "infinity"]
