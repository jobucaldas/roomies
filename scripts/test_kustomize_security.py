#!/usr/bin/env python3
"""Mutation tests for the rendered Kubernetes security contract."""

from __future__ import annotations

from copy import deepcopy
from pathlib import Path
import sys
from typing import Any, Callable

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
    print("Kubernetes security mutation tests passed: unsecured container, UID 0, drop ALL, Unconfined seccomp")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(Path(sys.argv[1])))
