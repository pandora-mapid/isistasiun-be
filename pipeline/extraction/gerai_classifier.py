"""Visual classification of gerai (stall/shop) photos from field survey into
the 5 baku categories, per section 3.3.
"""

from dataclasses import dataclass

from shared.backend_client import push_gerai_classification

CATEGORIES = ["makanan_minuman", "ritel_kemasan", "apotek_kesehatan", "jasa", "lainnya"]


@dataclass
class GeraiClassificationResult:
    job_id: str
    source_ref: str
    station_id: str
    gerai_id: str
    category: str
    visibility: str  # "high" | "medium" | "low"
    confidence: float


def classify_gerai(image_path: str, station_id: str, gerai_id: str, job_id: str) -> GeraiClassificationResult:
    """TODO: VLM classification into CATEGORIES + visibility heuristic."""
    raise NotImplementedError


def run_batch(image_paths: list[str], station_id: str, job_id: str) -> None:
    for path in image_paths:
        gerai_id = path  # placeholder — real ID comes from survey inventory
        result = classify_gerai(path, station_id, gerai_id, job_id)
        push_gerai_classification(result.__dict__)
