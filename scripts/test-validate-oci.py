"""Offline descriptor-graph regression tests, without a registry or containers."""
import copy
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("validator", Path(__file__).with_name("validate-oci.py"))
validator = importlib.util.module_from_spec(spec)
spec.loader.exec_module(validator)


class DescriptorTests(unittest.TestCase):
    def test_referenced_content(self):
        with tempfile.TemporaryDirectory() as directory:
            layout = Path(directory)
            blobs = layout / "blobs" / "sha256"
            blobs.mkdir(parents=True)

            def blob(data, media):
                digest = hashlib.sha256(data).hexdigest()
                (blobs / digest).write_bytes(data)
                return {"digest": "sha256:" + digest, "size": len(data), "mediaType": media}

            prefix = "application/vnd.oci.image."
            config = blob(b'{}', prefix + "config.v1+json")
            layer = blob(b'layer', prefix + "layer.v1.tar")

            def index(layer_descriptor):
                manifest = blob(json.dumps({"config": config, "layers": [layer_descriptor]}).encode(), prefix + "manifest.v1+json")
                nested = blob(json.dumps({"manifests": [manifest]}).encode(), prefix + "index.v1+json")
                (layout / "index.json").write_text(json.dumps({"manifests": [nested]}))

            index(layer)
            validator.validate(layout)
            for field, value in (("size", 999), ("digest", "sha256:" + "0" * 64)):
                malformed = copy.deepcopy(layer)
                malformed[field] = value
                index(malformed)
                with self.assertRaises((ValueError, FileNotFoundError)):
                    validator.validate(layout)
            index(layer)
            (blobs / layer["digest"][7:]).write_bytes(b'other')
            with self.assertRaises(ValueError):
                validator.validate(layout)


if __name__ == "__main__":
    unittest.main()
