"""Optional per-image manifest that maps a photo filename to the survey-sheet
identifiers the backend callbacks require as UUIDs (gerai_id, plot_id).

The field team records these when doing the gerai inventory; without a manifest
the batch commands fall back to the filename stem, which only works if the team
names files by the id directly. Format (JSON):

    {
      "G-07.jpg": {"gerai_id": "3f2b...-uuid"},
      "P-014_front.jpg": {"plot_id": "a91c...-uuid"}
    }
"""

from __future__ import annotations

import json
from pathlib import Path

Manifest = dict[str, dict[str, str]]


def load_manifest(path: str | None) -> Manifest:
    if not path:
        return {}
    data = json.loads(Path(path).read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise ValueError("manifest must be a JSON object keyed by filename")
    return data


def resolve(manifest: Manifest, image_path: str, key: str, fallback: str) -> str:
    entry = manifest.get(Path(image_path).name) or {}
    return entry.get(key) or fallback
