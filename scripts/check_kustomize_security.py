#!/usr/bin/env python3
"""Validate the rendered Roomies workload security and endpoint contract."""

from __future__ import annotations

import argparse
from pathlib import Path
import sys
from typing import Any

import yaml


def load_documents(path: Path) -> list[dict[str, Any]]:
    with path.open(encoding="utf-8") as stream:
        return [doc for doc in yaml.safe_load_all(stream) if isinstance(doc, dict)]


def mapping(value: Any, path: str, errors: list[str]) -> dict[str, Any]:
    if value is None:
        return {}
    if not isinstance(value, dict):
        errors.append(f"{path} must be a mapping")
        return {}
    return value


def positive_int(value: Any) -> bool:
    return isinstance(value, int) and not isinstance(value, bool) and value > 0


def deployment_documents(documents: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    return {
        str(doc.get("metadata", {}).get("name")): doc
        for doc in documents
        if doc.get("kind") == "Deployment"
    }


def service_documents(documents: list[dict[str, Any]]) -> dict[str, dict[str, Any]]:
    return {
        str(doc.get("metadata", {}).get("name")): doc
        for doc in documents
        if doc.get("kind") == "Service"
    }


def config_map(documents: list[dict[str, Any]], name: str) -> dict[str, Any] | None:
    for doc in documents:
        if doc.get("kind") == "ConfigMap" and doc.get("metadata", {}).get("name") == name:
            return doc
    return None


def find_container(
    deployment: dict[str, Any], name: str, errors: list[str]
) -> dict[str, Any] | None:
    deployment_name = deployment.get("metadata", {}).get("name", "<unnamed>")
    pod = mapping(
        mapping(deployment.get("spec"), f"Deployment/{deployment_name}.spec", errors).get(
            "template"
        ),
        f"Deployment/{deployment_name}.spec.template",
        errors,
    )
    pod_spec = mapping(pod.get("spec"), f"Deployment/{deployment_name}.spec.template.spec", errors)
    for container in pod_spec.get("containers", []):
        if isinstance(container, dict) and container.get("name") == name:
            return container
    errors.append(f"Deployment/{deployment_name}: missing container {name}")
    return None


def port_number(container: dict[str, Any], name: str, errors: list[str], context: str) -> int | None:
    ports = container.get("ports", [])
    if not isinstance(ports, list):
        errors.append(f"{context}.ports must be a list")
        return None
    matches = [port for port in ports if isinstance(port, dict) and port.get("name") == name]
    if len(matches) != 1 or not isinstance(matches[0].get("containerPort"), int):
        errors.append(f"{context}: expected one named port {name}")
        return None
    return matches[0]["containerPort"]


def assert_probe(
    container: dict[str, Any], probe_name: str, expected_path: str, expected_port: str,
    errors: list[str], context: str,
) -> None:
    probe = container.get(probe_name)
    http_get = probe.get("httpGet") if isinstance(probe, dict) else None
    if not isinstance(http_get, dict):
        errors.append(f"{context}.{probe_name}: missing httpGet")
        return
    if http_get.get("path") != expected_path or http_get.get("port") != expected_port:
        errors.append(
            f"{context}.{probe_name}: expected {expected_path} on port {expected_port}"
        )


def assert_service(
    service: dict[str, Any] | None,
    name: str,
    service_port: int,
    target_port: str,
    target_container: dict[str, Any] | None,
    errors: list[str],
) -> None:
    if service is None:
        errors.append(f"missing Service/{name}")
        return
    ports = mapping(service.get("spec"), f"Service/{name}.spec", errors).get("ports", [])
    if not isinstance(ports, list):
        errors.append(f"Service/{name}.spec.ports must be a list")
        return
    matches = [port for port in ports if isinstance(port, dict) and port.get("name") == "http"]
    if len(matches) != 1:
        errors.append(f"Service/{name}: expected one named http port")
        return
    port = matches[0]
    if port.get("port") != service_port or port.get("targetPort") != target_port:
        errors.append(
            f"Service/{name}: expected port {service_port} targeting named port {target_port}"
        )
    if target_container is not None:
        resolved = port_number(target_container, target_port, errors, f"Service/{name} target")
        if resolved is None or resolved <= 0:
            errors.append(f"Service/{name}: target port {target_port} does not resolve")


def validate_security(documents: list[dict[str, Any]]) -> list[str]:
    errors: list[str] = []
    deployments = deployment_documents(documents)
    if not deployments:
        errors.append("rendered manifest contains no Deployment")
        return errors

    inventory: list[str] = []
    for deployment_name, deployment in deployments.items():
        deployment_path = f"Deployment/{deployment_name}"
        spec = mapping(deployment.get("spec"), f"{deployment_path}.spec", errors)
        template = mapping(spec.get("template"), f"{deployment_path}.spec.template", errors)
        pod_spec = mapping(
            template.get("spec"), f"{deployment_path}.spec.template.spec", errors
        )
        pod_security = mapping(
            pod_spec.get("securityContext"), f"{deployment_path}.pod.securityContext", errors
        )

        volumes = pod_spec.get("volumes", [])
        if not isinstance(volumes, list):
            errors.append(f"{deployment_path}.pod.volumes must be a list")
        else:
            for index, volume in enumerate(volumes):
                if isinstance(volume, dict) and "hostPath" in volume:
                    errors.append(f"{deployment_path}.pod.volumes[{index}] uses hostPath")

        containers: list[tuple[str, Any]] = []
        for field in ("initContainers", "containers"):
            value = pod_spec.get(field, [])
            if not isinstance(value, list):
                errors.append(f"{deployment_path}.pod.{field} must be a list")
                continue
            containers.extend((field, item) for item in value)

        if not any(field == "containers" for field, _ in containers):
            errors.append(f"{deployment_path}: no app containers")

        for field, raw_container in containers:
            if not isinstance(raw_container, dict):
                errors.append(f"{deployment_path}.pod.{field} contains a non-mapping")
                continue
            name = raw_container.get("name", f"{field}[unnamed]")
            context = f"{deployment_path}.pod.{field}/{name}"
            inventory.append(context)
            container_security = mapping(
                raw_container.get("securityContext"), f"{context}.securityContext", errors
            )

            def effective(key: str) -> Any:
                if key in container_security:
                    return container_security[key]
                return pod_security.get(key)

            if effective("runAsNonRoot") is not True:
                errors.append(f"{context}: effective runAsNonRoot must be true")
            if not positive_int(effective("runAsUser")):
                errors.append(f"{context}: effective runAsUser must be a nonzero positive integer")
            if not positive_int(effective("runAsGroup")):
                errors.append(f"{context}: effective runAsGroup must be a nonzero positive integer")
            if container_security.get("allowPrivilegeEscalation") is not False:
                errors.append(f"{context}: allowPrivilegeEscalation must be false")
            if container_security.get("readOnlyRootFilesystem") is not True:
                errors.append(f"{context}: readOnlyRootFilesystem must be true")
            if container_security.get("privileged") is True:
                errors.append(f"{context}: privileged containers are forbidden")

            capabilities = container_security.get("capabilities")
            drops = capabilities.get("drop", []) if isinstance(capabilities, dict) else []
            if not isinstance(drops, list) or "ALL" not in drops:
                errors.append(f"{context}: capabilities.drop must include ALL")

            seccomp = container_security.get("seccompProfile", pod_security.get("seccompProfile"))
            seccomp_type = seccomp.get("type") if isinstance(seccomp, dict) else None
            if seccomp_type != "RuntimeDefault":
                errors.append(f"{context}: effective seccompProfile.type must be RuntimeDefault")

    # Endpoint checks are semantic checks over the rendered object model, not text counts.
    backend = deployments.get("roomies-backend")
    worker = deployments.get("roomies-backend-worker")
    frontend = deployments.get("roomies-frontend")
    services = service_documents(documents)
    backend_container = find_container(backend, "backend", errors) if backend else None
    worker_container = find_container(worker, "backend-worker", errors) if worker else None
    caddy_container = find_container(frontend, "caddy", errors) if frontend else None

    if backend_container is not None:
        if port_number(backend_container, "http", errors, "Deployment/roomies-backend/backend") != 8080:
            errors.append("Deployment/roomies-backend/backend: http must listen on 8080")
        assert_probe(backend_container, "readinessProbe", "/readyz", "http", errors, "Deployment/roomies-backend/backend")
        assert_probe(backend_container, "livenessProbe", "/healthz", "http", errors, "Deployment/roomies-backend/backend")
    if worker_container is not None:
        if port_number(worker_container, "health", errors, "Deployment/roomies-backend-worker/backend-worker") != 8081:
            errors.append("Deployment/roomies-backend-worker/backend-worker: health must listen on 8081")
        assert_probe(worker_container, "readinessProbe", "/readyz", "health", errors, "Deployment/roomies-backend-worker/backend-worker")
        assert_probe(worker_container, "livenessProbe", "/healthz", "health", errors, "Deployment/roomies-backend-worker/backend-worker")
    if caddy_container is not None:
        if port_number(caddy_container, "http", errors, "Deployment/roomies-frontend/caddy") != 8080:
            errors.append("Deployment/roomies-frontend/caddy: http must listen on 8080")
        assert_probe(caddy_container, "readinessProbe", "/", "http", errors, "Deployment/roomies-frontend/caddy")
        assert_probe(caddy_container, "livenessProbe", "/", "http", errors, "Deployment/roomies-frontend/caddy")

    assert_service(services.get("roomies-backend"), "roomies-backend", 8080, "http", backend_container, errors)
    assert_service(services.get("roomies-frontend"), "roomies-frontend", 80, "http", caddy_container, errors)

    caddy_config = config_map(documents, "roomies-frontend-caddyfile")
    caddyfile = caddy_config.get("data", {}).get("Caddyfile") if caddy_config else None
    listeners = {
        line.strip()
        for line in caddyfile.splitlines()
        if isinstance(caddyfile, str) and line.strip() and not line.strip().startswith("#")
    } if isinstance(caddyfile, str) else set()
    if ":8080" not in listeners:
        errors.append("roomies-frontend-caddyfile: Caddy must listen on :8080")
    if ":80" in listeners:
        errors.append("roomies-frontend-caddyfile: Caddy must not listen on privileged :80")

    return errors


def validate_manifest(path: Path) -> list[str]:
    return validate_security(load_documents(path))


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("manifest", type=Path)
    args = parser.parse_args(argv)
    errors = validate_manifest(args.manifest)
    if errors:
        for error in errors:
            print(f"ERROR: {error}", file=sys.stderr)
        return 1
    documents = load_documents(args.manifest)
    deployments = deployment_documents(documents)
    count = 0
    for deployment in deployments.values():
        spec = deployment.get("spec", {})
        template = spec.get("template", {}) if isinstance(spec, dict) else {}
        pod = template.get("spec", {}) if isinstance(template, dict) else {}
        if isinstance(pod, dict):
            for field in ("initContainers", "containers"):
                value = pod.get(field, [])
                count += len(value) if isinstance(value, list) else 0
    print(f"{args.manifest}: validated {len(deployments)} Deployments and {count} containers/initContainers")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
