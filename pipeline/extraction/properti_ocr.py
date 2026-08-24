"""OCR extraction for Properti Go spanduk (rental banner) photos:
offered rent + area (sqm), per section 3.2.
"""

from dataclasses import dataclass

from shared.backend_client import push_properti_extraction


@dataclass
class PropertiExtractionResult:
    job_id: str
    source_ref: str
    station_id: str
    plot_id: str
    offered_rent: float
    area_sqm: float
    confidence: float


def extract_properti(image_path: str, station_id: str, plot_id: str, job_id: str) -> PropertiExtractionResult:
    """TODO: VLM OCR for rent price + area from spanduk photos."""
    raise NotImplementedError


def run_batch(image_paths: list[str], station_id: str, job_id: str) -> None:
    for path in image_paths:
        # plot_id typically derived from filename/metadata convention agreed
        # with the survey team — placeholder here.
        plot_id = path
        result = extract_properti(path, station_id, plot_id, job_id)
        push_properti_extraction(result.__dict__)
