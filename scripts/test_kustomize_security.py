#!/usr/bin/env python3
"""Mutation tests for the rendered Kubernetes security contract."""

from __future__ import annotations

from copy import deepcopy
import os
from pathlib import Path
import subprocess
import sys
import tempfile
from typing import Any, Callable

import yaml

from check_kustomize_security import load_documents, validate_security


def deployment(documents: list[dict[str, Any]], name: str) -> dict[str, Any]:
    for document in documents:
        if document.get("kind") == "Deployment" and document.get("metadata", {}).get("name") == name:
            return document
    raise AssertionError(f"missing Deployment/{name}")


def backend_container(documents: list[dict[str, Any]]) -> dict[str, Any]:
    pod = deployment(documents, "roomies-backend")["spec"]["template"]["spec"]
    return next(container for container in pod["containers"] if container["name"] == "backend")


def assert_rejected(
    original: list[dict[str, Any]], description: str, mutate: Callable[[list[dict[str, Any]]], None],
    expected: str,
) -> None:
    documents = deepcopy(original)
    mutate(documents)
    errors = validate_security(documents)
    if not errors:
        raise AssertionError(f"mutation unexpectedly passed: {description}")
    if not any(expected in error for error in errors):
        raise AssertionError(
            f"mutation failed for an unexpected reason: {description}; expected {expected!r}, got {errors}"
        )


def remove_deployment(documents: list[dict[str, Any]], name: str) -> None:
    documents[:] = [
        document
        for document in documents
        if not (
            document.get("kind") == "Deployment"
            and document.get("metadata", {}).get("name") == name
        )
    ]


def assert_cli_rejected(contents: str, description: str, expected: str) -> None:
    checker = Path(
        os.environ.get(
            "SECURITY_PARSER", str(Path(__file__).with_name("check_kustomize_security.py"))
        )
    )
    with tempfile.NamedTemporaryFile("w", suffix=".yaml", encoding="utf-8") as fixture:
        fixture.write(contents)
        fixture.flush()
        result = subprocess.run(
            [sys.executable, str(checker), fixture.name],
            capture_output=True,
            text=True,
            check=False,
        )
    if result.returncode == 0:
        raise AssertionError(f"CLI mutation unexpectedly passed: {description}")
    if expected not in result.stderr:
        raise AssertionError(
            f"CLI mutation failed for an unexpected reason: {description}; "
            f"expected {expected!r}, got {result.stderr!r}"
        )


def main(path: Path) -> int:
    original = load_documents(path)
    assert_rejected(
        original,
        "extra unsecured container",
        lambda documents: deployment(documents, "roomies-backend")["spec"]["template"]["spec"]["containers"].append(
            {"name": "unsecured", "image": "example.invalid/unsecured:mutation"}
        ),
        "unsecured",
    )

    duplicate_identity = deepcopy(deployment(original, "roomies-backend"))
    assert_rejected(
        original,
        "duplicate Deployment identity",
        lambda documents: documents.append(deepcopy(duplicate_identity)),
        "duplicate resource identity",
    )

    same_name = {
        "apiVersion": "apps/v1",
        "kind": "Deployment",
        "metadata": {"name": "roomies-backend", "namespace": "other"},
        "spec": {
            "template": {
                "spec": {
                    "containers": [
                        {"name": "other-unsecured", "image": "example.invalid/mutation"}
                    ]
                }
            }
        },
    }
    for position in ("front", "back"):
        def add_same_name(documents: list[dict[str, Any]], position: str = position) -> None:
            if position == "front":
                documents.insert(0, deepcopy(same_name))
            else:
                documents.append(deepcopy(same_name))

        assert_rejected(
            original,
            f"unsecured same-name Deployment in another namespace ({position})",
            add_same_name,
            "Deployment/other/roomies-backend",
        )

    for name in ("roomies-backend", "roomies-backend-worker", "roomies-frontend"):
        assert_rejected(
            original,
            f"missing baseline {name}",
            lambda documents, name=name: remove_deployment(documents, name),
            f"missing required Deployment/default/{name}",
        )

    assert_rejected(
        original,
        "UID zero",
        lambda documents: backend_container(documents)["securityContext"].update({"runAsUser": 0}),
        "runAsUser must be a nonzero positive integer",
    )
    assert_rejected(
        original,
        "missing drop ALL",
        lambda documents: backend_container(documents)["securityContext"]["capabilities"].update({"drop": []}),
        "capabilities.drop must include ALL",
    )
    assert_rejected(
        original,
        "Unconfined seccomp",
        lambda documents: deployment(documents, "roomies-backend")["spec"]["template"]["spec"]["securityContext"][
            "seccompProfile"
        ].update({"type": "Unconfined"}),
        "seccompProfile.type must be RuntimeDefault",
    )

    assert_cli_rejected("just-a-scalar\n", "scalar YAML document", "document[0] must be a mapping")
    privileged_documents = deepcopy(original)
    backend_container(privileged_documents)["securityContext"]["privileged"] = "true"
    assert_cli_rejected(
        yaml.safe_dump_all(privileged_documents),
        "string privileged field",
        "privileged must be a boolean",
    )

    print(
        "Kubernetes security mutation tests passed: duplicate identity, duplicate-name namespace, baseline removal, "
        "unsecured container, UID 0, drop ALL, Unconfined seccomp, scalar document, string privileged"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main(Path(sys.argv[1])))
