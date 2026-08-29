"""OCR extraction for Properti Go spanduk (rental banner) photos: offered rent
+ area (sqm), per section 3.2. Feeds the rent-per-flow index for the area
*around* a station (not in-station plots — see methodology 3.5).
"""

from __future__ import annotations

import re
import sys
from dataclasses import dataclass
from pathlib import Path

from shared.backend_client import push_properti_extraction
from shared.gemini import extract_json
from shared.manifest import Manifest, resolve

_PROMPT = """You are reading one property rental banner (spanduk sewa) photo in Indonesia.
Return ONE JSON object, no prose, with exactly these fields:
- "offered_rent": number - the asking rent in rupiah per year, 0 if not shown
- "rent_period": one of "year", "month", "" - the period the rent figure refers to
- "area_sqm": number - floor area in square metres, 0 if not shown
- "readable": boolean - false if the banner is torn, blurred, or largely illegible
Convert "15 jt" to 15000000, "1,5 M" to 1500000000. Never invent numbers you cannot see."""

_MONTHS_PER_YEAR = 12


@dataclass
class PropertiExtractionResult:
    job_id: str
    source_ref: str  # R2 object key
    station_id: str
    plot_id: str
    offered_rent: float  # normalised to rupiah / year
    area_sqm: float
    confidence: float
    photo_url: str = ""  # redacted public URL; set to also write a transparency card


def _plot_id_from_path(image_path: str) -> str:
    """Fallback plot id when no manifest entry exists: the leading token of the
    filename, e.g. ``P-014_front.jpg`` -> ``P-014``."""
    stem = Path(image_path).stem
    return re.split(r"[_\s]", stem, maxsplit=1)[0] or stem


def _annualise(rent: float, period: str) -> float:
    if period == "month":
        return rent * _MONTHS_PER_YEAR
    return rent


def extract_properti(image_path: str, station_id: str, plot_id: str, job_id: str) -> PropertiExtractionResult:
    raw = extract_json(_PROMPT, image_path)

    rent = _annualise(float(raw.get("offered_rent") or 0.0), raw.get("rent_period", ""))
    area = float(raw.get("area_sqm") or 0.0)
    readable = bool(raw.get("readable", True))
    confidence = 0.85 if (readable and rent > 0 and area > 0) else 0.4

    return PropertiExtractionResult(
        job_id=job_id,
        source_ref=Path(image_path).name,
        station_id=station_id,
        plot_id=plot_id,
        offered_rent=rent,
        area_sqm=area,
        confidence=confidence,
    )


def run_batch(
    image_paths: list[str],
    station_id: str,
    job_id: str,
    manifest: Manifest | None = None,
) -> None:
    manifest = manifest or {}
    skipped = 0
    for path in image_paths:
        plot_id = resolve(manifest, path, "plot_id", _plot_id_from_path(path))
        result = extract_properti(path, station_id, plot_id, job_id)
        # The callback requires a positive rent and area; a banner missing either
        # carries no usable signal for the rent-per-flow index, so drop it here
        # rather than push a row that would 400.
        if result.offered_rent <= 0 or result.area_sqm <= 0:
            skipped += 1
            print(f"skip {Path(path).name}: incomplete (rent={result.offered_rent}, area={result.area_sqm})", file=sys.stderr)
            continue
        push_properti_extraction(result.__dict__)
    if skipped:
        print(f"{skipped}/{len(image_paths)} banners skipped as incomplete", file=sys.stderr)
