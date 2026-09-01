FROM docker.io/rust:1.88-bookworm AS builder

ARG DIOXUS_CLI_VERSION=0.7.10
RUN rustup target add wasm32-unknown-unknown \
    && cargo install dioxus-cli --version "${DIOXUS_CLI_VERSION}" --locked

WORKDIR /workspace
ARG ROOMIES_API_URL=/api
ENV ROOMIES_API_URL=${ROOMIES_API_URL}
COPY . .
WORKDIR /workspace/src/app
RUN dx build --release --web

FROM docker.io/alpine:3.20
WORKDIR /app
COPY --from=builder /workspace/src/app/target/dx/roomies-app/release/web/public/ /app/dist/
CMD ["/bin/true"]
