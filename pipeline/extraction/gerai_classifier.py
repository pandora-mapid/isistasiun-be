"""Visual classification of gerai (stall/shop) photos from field survey into
the 5 baku categories, plus a storefront-visibility read, per section 3.3.
"""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from shared.backend_client import push_gerai_classification
from shared.categories import BAKU_CATEGORIES, normalize_category
from shared.gemini import extract_json
from shared.manifest import Manifest, resolve

_VISIBILITY = ("high", "medium", "low")

_PROMPT = """You are looking at one photo of a shop/stall (gerai) in or near an Indonesian transit station.
Return ONE JSON object, no prose, with exactly these fields:
- "business_label": string - what this business sells or does, in a few words (e.g. "coffee shop",
  "pharmacy", "minimarket", "phone accessories", "shoe repair")
- "visibility": one of "high", "medium", "low" - how prominent the storefront is to passing pedestrians
  (signage size, frontage width, lighting, whether the entrance faces the main flow)
- "readable": boolean - false if the photo is too dark, blurred, or obstructed to judge
Never invent a business type you cannot infer from the image."""


@dataclass
class GeraiClassificationResult:
    job_id: str
    source_ref: str  # R2 object key
    station_id: str
    gerai_id: str
    category: str
    visibility: str  # "high" | "medium" | "low"
    confidence: float
    photo_url: str = ""  # redacted public URL; set to also write a transparency card




def classify_gerai(image_path: str, station_id: str, gerai_id: str, job_id: str) -> GeraiClassificationResult:
    raw = extract_json(_PROMPT, image_path)

    category = normalize_category(raw.get("business_label", ""))
    visibility = raw.get("visibility", "")
    if visibility not in _VISIBILITY:
        visibility = "medium"

    readable = bool(raw.get("readable", True))
    confident = readable and category in BAKU_CATEGORIES and category != "lainnya"
    confidence = 0.8 if confident else 0.45

    return GeraiClassificationResult(
        job_id=job_id,
        source_ref=Path(image_path).name,
        station_id=station_id,
        gerai_id=gerai_id,
        category=category,
        visibility=visibility,
        confidence=confidence,
    )


def run_batch(
    image_paths: list[str],
    station_id: str,
    job_id: str,
    manifest: Manifest | None = None,
) -> None:
    manifest = manifest or {}
    for path in image_paths:
        gerai_id = resolve(manifest, path, "gerai_id", Path(path).stem)
        result = classify_gerai(path, station_id, gerai_id, job_id)
        push_gerai_classification(result.__dict__)
