#!/usr/bin/env python3
"""Validate the rendered Roomies workload security and endpoint contract."""

from __future__ import annotations

import argparse
from pathlib import Path
import sys
from typing import Any

import yaml


Identity = tuple[str, str]
DEFAULT_NAMESPACE = "default"


def load_documents(path: Path) -> list[Any]:
    with path.open(encoding="utf-8") as stream:
        return list(yaml.safe_load_all(stream))


def mapping(value: Any, path: str, errors: list[str]) -> dict[str, Any]:
    if not isinstance(value, dict):
        errors.append(f"{path} must be a mapping")
        return {}
    return value


def positive_int(value: Any) -> bool:
    return isinstance(value, int) and not isinstance(value, bool) and value > 0


def resource_identity(
    document: dict[str, Any], kind: str, index: int, errors: list[str]
) -> Identity | None:
    metadata = document.get("metadata")
    if not isinstance(metadata, dict):
        errors.append(f"{kind} document {index}: metadata must be a mapping")
        return None
    name = metadata.get("name")
    namespace = metadata.get("namespace", DEFAULT_NAMESPACE)
    if not isinstance(name, str) or not name:
        errors.append(f"{kind} document {index}: metadata.name must be a nonempty string")
        return None
    if not isinstance(namespace, str) or not namespace:
        errors.append(
            f"{kind}/{name}: metadata.namespace must be a nonempty string"
        )
        return None
    return namespace, name


def deployment_documents(
    documents: list[dict[str, Any]], errors: list[str]
) -> list[tuple[Identity, dict[str, Any]]]:
    deployments: list[tuple[Identity, dict[str, Any]]] = []
    seen: set[Identity] = set()
    for index, document in enumerate(documents):
        if document.get("kind") != "Deployment":
            continue
        identity = resource_identity(document, "Deployment", index, errors)
        if identity is None:
            continue
        if identity in seen:
            errors.append(
                f"Deployment/{identity[0]}/{identity[1]}: duplicate resource identity"
            )
        seen.add(identity)
        deployments.append((identity, document))
    return deployments


def service_documents(
    documents: list[dict[str, Any]], errors: list[str]
) -> list[tuple[Identity, dict[str, Any]]]:
    services: list[tuple[Identity, dict[str, Any]]] = []
    seen: set[Identity] = set()
    for index, document in enumerate(documents):
        if document.get("kind") != "Service":
            continue
        identity = resource_identity(document, "Service", index, errors)
        if identity is None:
            continue
        if identity in seen:
            errors.append(
                f"Service/{identity[0]}/{identity[1]}: duplicate resource identity"
            )
        seen.add(identity)
        services.append((identity, document))
    return services


def config_map(
    documents: list[dict[str, Any]], name: str, errors: list[str]
) -> dict[str, Any] | None:
    matches: list[dict[str, Any]] = []
    for index, document in enumerate(documents):
        if document.get("kind") != "ConfigMap":
            continue
        identity = resource_identity(document, "ConfigMap", index, errors)
        if identity is not None and identity[1] == name:
            matches.append(document)
    if len(matches) > 1:
        errors.append(f"ConfigMap/{name}: duplicate name")
    return matches[0] if matches else None


def find_resource(
    resources: list[tuple[Identity, dict[str, Any]]],
    namespace: str,
    name: str,
    kind: str,
    errors: list[str],
) -> dict[str, Any] | None:
    identity = (namespace, name)
    matches = [document for resource_id, document in resources if resource_id == identity]
    if not matches:
        errors.append(f"missing required {kind}/{namespace}/{name}")
        return None
    return matches[0]


def find_container(
    deployment: dict[str, Any], identity: Identity, name: str, errors: list[str]
) -> dict[str, Any] | None:
    deployment_path = f"Deployment/{identity[0]}/{identity[1]}"
    spec = mapping(deployment.get("spec"), f"{deployment_path}.spec", errors)
    template = mapping(spec.get("template"), f"{deployment_path}.spec.template", errors)
    pod_spec = mapping(
        template.get("spec"), f"{deployment_path}.spec.template.spec", errors
    )
    containers = pod_spec.get("containers", [])
    if not isinstance(containers, list):
        errors.append(f"{deployment_path}.pod.containers must be a list")
        return None
    for container in containers:
        if isinstance(container, dict) and container.get("name") == name:
            return container
    errors.append(f"{deployment_path}: missing required container {name}")
    return None


def port_number(
    container: dict[str, Any], name: str, errors: list[str], context: str
) -> int | None:
    ports = container.get("ports", [])
    if not isinstance(ports, list):
        errors.append(f"{context}.ports must be a list")
        return None
    matches: list[dict[str, Any]] = []
    for index, port in enumerate(ports):
        if not isinstance(port, dict):
            errors.append(f"{context}.ports[{index}] must be a mapping")
            continue
        if port.get("name") == name:
            matches.append(port)
    if len(matches) != 1 or not positive_int(matches[0].get("containerPort")):
        errors.append(f"{context}: expected one named port {name} with a positive integer")
        return None
    return matches[0]["containerPort"]


def assert_probe(
    container: dict[str, Any],
    probe_name: str,
    expected_path: str,
    expected_port: str,
    errors: list[str],
    context: str,
) -> None:
    probe = container.get(probe_name)
    http_get = probe.get("httpGet") if isinstance(probe, dict) else None
    if not isinstance(http_get, dict):
        errors.append(f"{context}.{probe_name}: missing httpGet mapping")
        return
    if http_get.get("path") != expected_path or http_get.get("port") != expected_port:
        errors.append(
            f"{context}.{probe_name}: expected {expected_path} on port {expected_port}"
        )


def assert_service(
    service: dict[str, Any] | None,
    identity: Identity,
    service_port: int,
    target_port: str,
    target_container: dict[str, Any] | None,
    errors: list[str],
) -> None:
    service_path = f"Service/{identity[0]}/{identity[1]}"
    if service is None:
        return
    spec = mapping(service.get("spec"), f"{service_path}.spec", errors)
    ports = spec.get("ports", [])
    if not isinstance(ports, list):
        errors.append(f"{service_path}.spec.ports must be a list")
        return
    matches: list[dict[str, Any]] = []
    for index, port in enumerate(ports):
        if not isinstance(port, dict):
            errors.append(f"{service_path}.ports[{index}] must be a mapping")
            continue
        if port.get("name") == "http":
            matches.append(port)
    if len(matches) != 1:
        errors.append(f"{service_path}: expected one named http port")
        return
    port = matches[0]
    if port.get("port") != service_port or port.get("targetPort") != target_port:
        errors.append(
            f"{service_path}: expected port {service_port} targeting named port {target_port}"
        )
    if target_container is not None:
        resolved = port_number(target_container, target_port, errors, f"{service_path} target")
        if resolved is None or resolved <= 0:
            errors.append(f"{service_path}: target port {target_port} does not resolve")


def security_context(
    container: dict[str, Any], context: str, errors: list[str]
) -> dict[str, Any]:
    if "securityContext" not in container:
        errors.append(f"{context}.securityContext must be a mapping")
        return {}
    return mapping(container["securityContext"], f"{context}.securityContext", errors)


def typed_security_fields(
    security: dict[str, Any], context: str, errors: list[str]
) -> None:
    bool_fields = (
        "runAsNonRoot",
        "allowPrivilegeEscalation",
        "readOnlyRootFilesystem",
        "privileged",
    )
    for field in bool_fields:
        if field in security and not isinstance(security[field], bool):
            errors.append(f"{context}.{field} must be a boolean")
    for field in ("runAsUser", "runAsGroup"):
        if field in security and not positive_int(security[field]):
            errors.append(f"{context}.{field} must be a nonzero positive integer")

    if "capabilities" in security:
        capabilities = security["capabilities"]
        if not isinstance(capabilities, dict):
            errors.append(f"{context}.capabilities must be a mapping")
        elif "drop" in capabilities:
            drops = capabilities["drop"]
            if not isinstance(drops, list):
                errors.append(f"{context}.capabilities.drop must be a list")
            else:
                for index, drop in enumerate(drops):
                    if not isinstance(drop, str):
                        errors.append(
                            f"{context}.capabilities.drop[{index}] must be a string"
                        )

    if "seccompProfile" in security:
        seccomp = security["seccompProfile"]
        if not isinstance(seccomp, dict):
            errors.append(f"{context}.seccompProfile must be a mapping")
        elif "type" in seccomp and not isinstance(seccomp["type"], str):
            errors.append(f"{context}.seccompProfile.type must be a string")


def validate_security(documents: list[Any]) -> list[str]:
    errors: list[str] = []
    mappings: list[dict[str, Any]] = []
    for index, document in enumerate(documents):
        if not isinstance(document, dict):
            errors.append(f"document[{index}] must be a mapping")
            continue
        mappings.append(document)

    deployments = deployment_documents(mappings, errors)
    if not deployments:
        errors.append("rendered manifest contains no Deployment")
        return errors

    inventory: list[str] = []
    for identity, deployment in deployments:
        deployment_path = f"Deployment/{identity[0]}/{identity[1]}"
        spec = mapping(deployment.get("spec"), f"{deployment_path}.spec", errors)
        template = mapping(spec.get("template"), f"{deployment_path}.spec.template", errors)
        pod_spec = mapping(
            template.get("spec"), f"{deployment_path}.spec.template.spec", errors
        )
        if "securityContext" in pod_spec:
            pod_security = mapping(
                pod_spec["securityContext"],
                f"{deployment_path}.pod.securityContext",
                errors,
            )
            typed_security_fields(pod_security, f"{deployment_path}.pod.securityContext", errors)
        else:
            pod_security = {}
            errors.append(f"{deployment_path}.pod.securityContext must be a mapping")

        volumes = pod_spec.get("volumes", [])
        if not isinstance(volumes, list):
            errors.append(f"{deployment_path}.pod.volumes must be a list")
        else:
            for index, volume in enumerate(volumes):
                if not isinstance(volume, dict):
                    errors.append(f"{deployment_path}.pod.volumes[{index}] must be a mapping")
                elif "hostPath" in volume:
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
            if not isinstance(name, str) or not name:
                errors.append(f"{deployment_path}.pod.{field}.name must be a nonempty string")
                name = f"{field}[unnamed]"
            context = f"{deployment_path}.pod.{field}/{name}"
            inventory.append(context)
            container_security = security_context(raw_container, context, errors)
            typed_security_fields(container_security, context, errors)

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
            if "allowPrivilegeEscalation" not in container_security or container_security["allowPrivilegeEscalation"] is not False:
                errors.append(f"{context}: allowPrivilegeEscalation must be false")
            if "readOnlyRootFilesystem" not in container_security or container_security["readOnlyRootFilesystem"] is not True:
                errors.append(f"{context}: readOnlyRootFilesystem must be true")
            if "privileged" in container_security:
                if container_security["privileged"] is not False:
                    errors.append(f"{context}: privileged must be boolean false")

            capabilities = container_security.get("capabilities")
            drops = capabilities.get("drop", []) if isinstance(capabilities, dict) else []
            if not isinstance(drops, list) or "ALL" not in drops:
                errors.append(f"{context}: capabilities.drop must include ALL")

            seccomp = container_security.get("seccompProfile", pod_security.get("seccompProfile"))
            seccomp_type = seccomp.get("type") if isinstance(seccomp, dict) else None
            if seccomp_type != "RuntimeDefault":
                errors.append(f"{context}: effective seccompProfile.type must be RuntimeDefault")

    # Endpoint checks are semantic checks over the rendered object model, not text counts.
    services = service_documents(mappings, errors)
    service_namespaces = [identity[0] for identity, _ in services if identity[1] == "roomies-backend"]
    baseline_namespace = service_namespaces[0] if len(service_namespaces) == 1 else DEFAULT_NAMESPACE
    if len(service_namespaces) > 1:
        errors.append("roomies-backend Service has ambiguous namespace/name identities")

    backend = find_resource(
        deployments, baseline_namespace, "roomies-backend", "Deployment", errors
    )
    worker = find_resource(
        deployments, baseline_namespace, "roomies-backend-worker", "Deployment", errors
    )
    frontend = find_resource(
        deployments, baseline_namespace, "roomies-frontend", "Deployment", errors
    )
    backend_identity = (baseline_namespace, "roomies-backend")
    worker_identity = (baseline_namespace, "roomies-backend-worker")
    frontend_identity = (baseline_namespace, "roomies-frontend")
    backend_container = (
        find_container(backend, backend_identity, "backend", errors) if backend else None
    )
    worker_container = (
        find_container(worker, worker_identity, "backend-worker", errors) if worker else None
    )
    caddy_container = (
        find_container(frontend, frontend_identity, "caddy", errors) if frontend else None
    )

    backend_service = find_resource(
        services, baseline_namespace, "roomies-backend", "Service", errors
    )
    frontend_service = find_resource(
        services, baseline_namespace, "roomies-frontend", "Service", errors
    )
    if backend_container is not None:
        if port_number(backend_container, "http", errors, f"Deployment/{backend_identity[0]}/{backend_identity[1]}/backend") != 8080:
            errors.append("backend http must listen on 8080")
        assert_probe(backend_container, "readinessProbe", "/readyz", "http", errors, f"Deployment/{backend_identity[0]}/{backend_identity[1]}/backend")
        assert_probe(backend_container, "livenessProbe", "/healthz", "http", errors, f"Deployment/{backend_identity[0]}/{backend_identity[1]}/backend")
    if worker_container is not None:
        if port_number(worker_container, "health", errors, f"Deployment/{worker_identity[0]}/{worker_identity[1]}/backend-worker") != 8081:
            errors.append("worker health must listen on 8081")
        assert_probe(worker_container, "readinessProbe", "/readyz", "health", errors, f"Deployment/{worker_identity[0]}/{worker_identity[1]}/backend-worker")
        assert_probe(worker_container, "livenessProbe", "/healthz", "health", errors, f"Deployment/{worker_identity[0]}/{worker_identity[1]}/backend-worker")
    if caddy_container is not None:
        if port_number(caddy_container, "http", errors, f"Deployment/{frontend_identity[0]}/{frontend_identity[1]}/caddy") != 8080:
            errors.append("Caddy http must listen on 8080")
        assert_probe(caddy_container, "readinessProbe", "/", "http", errors, f"Deployment/{frontend_identity[0]}/{frontend_identity[1]}/caddy")
        assert_probe(caddy_container, "livenessProbe", "/", "http", errors, f"Deployment/{frontend_identity[0]}/{frontend_identity[1]}/caddy")

    assert_service(backend_service, (baseline_namespace, "roomies-backend"), 8080, "http", backend_container, errors)
    assert_service(frontend_service, (baseline_namespace, "roomies-frontend"), 80, "http", caddy_container, errors)

    caddy_config = config_map(mappings, "roomies-frontend-caddyfile", errors)
    caddy_data = caddy_config.get("data") if caddy_config else None
    caddyfile = caddy_data.get("Caddyfile") if isinstance(caddy_data, dict) else None
    if caddy_data is not None and not isinstance(caddy_data, dict):
        errors.append("roomies-frontend-caddyfile.data must be a mapping")
    listeners = {
        line.strip()
        for line in caddyfile.splitlines()
        if isinstance(caddyfile, str) and line.strip() and not line.strip().startswith("#")
    } if isinstance(caddyfile, str) else set()
    if not isinstance(caddyfile, str):
        errors.append("roomies-frontend-caddyfile.data.Caddyfile must be a string")
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
    documents = [document for document in load_documents(args.manifest) if isinstance(document, dict)]
    deployments = deployment_documents(documents, [])
    count = 0
    for _, deployment in deployments:
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
