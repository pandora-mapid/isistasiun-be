"""OCR + nominal extraction for Struk Go receipt photos.

Rule per methodology 3.4: take the *final amount paid* by the customer, not
subtotal/tax/change. Receipts that are unreadable, cut off, or have more than
one candidate final value are flagged is_ambiguous and excluded from
downstream Monte Carlo simulation (still logged for coverage reporting).
"""

from dataclasses import dataclass

from shared.backend_client import push_struk_extraction


@dataclass
class StrukExtractionResult:
    job_id: str
    source_ref: str
    station_id: str
    category: str
    final_amount: float
    payment_method: str
    transacted_at: str  # ISO 8601
    confidence: float
    is_ambiguous: bool


def extract_struk(image_path: str, station_id: str, job_id: str) -> StrukExtractionResult:
    """Run the vision-language model on one receipt photo.

    TODO: call Gemini (or chosen VLM) with a prompt constrained to the fields
    in StrukExtractionResult; parse + validate the structured output; flag
    is_ambiguous per the rule above. See section 3.4 / Lampiran for the
    100-photo manual-label validation set to benchmark against.
    """
    raise NotImplementedError


def run_batch(image_paths: list[str], station_id: str, job_id: str) -> None:
    for path in image_paths:
        result = extract_struk(path, station_id, job_id)
        push_struk_extraction(result.__dict__)
