"""Verify every reachable descriptor in an extracted local OCI image layout."""
import hashlib
import json
from pathlib import Path
import re
import sys


def validate(layout):
    def descriptor(item):
        digest = item["digest"]
        if not re.fullmatch(r"sha256:[0-9a-f]{64}", digest):
            raise ValueError(f"Unsupported digest: {digest}")
        data = (layout / "blobs" / "sha256" / digest[7:]).read_bytes()
        if type(item["size"]) is not int or len(data) != item["size"]:
            raise ValueError(f"Descriptor size mismatch: {digest}")
        if hashlib.sha256(data).hexdigest() != digest[7:]:
            raise ValueError(f"Descriptor digest mismatch: {digest}")
        media = item["mediaType"]
        if media in ("application/vnd.oci.image.index.v1+json",
                     "application/vnd.docker.distribution.manifest.list.v2+json"):
            for child in json.loads(data)["manifests"]:
                descriptor(child)
        elif media in ("application/vnd.oci.image.manifest.v1+json",
                       "application/vnd.docker.distribution.manifest.v2+json"):
            manifest = json.loads(data)
            descriptor(manifest["config"])
            for layer in manifest["layers"]:
                descriptor(layer)
        elif media not in ("application/vnd.oci.image.config.v1+json",
                            "application/vnd.docker.container.image.v1+json") and not (
            media.startswith("application/vnd.oci.image.layer.v1.tar") or
            media.startswith("application/vnd.docker.image.rootfs.diff.tar")
        ):
            raise ValueError(f"Unsupported descriptor media type: {media}")

    for manifest in json.loads((layout / "index.json").read_bytes())["manifests"]:
        descriptor(manifest)


if __name__ == "__main__":
    validate(Path(sys.argv[1]))
