"""OCR + nominal extraction for Struk Go receipt photos.

Rule per methodology 3.4: take the *final amount paid* by the customer, not
subtotal / tax / cash tendered / change. Receipts that are unreadable, cut off,
or expose more than one candidate final value are flagged is_ambiguous and
excluded from the Monte Carlo simulation downstream (still pushed, so they
count toward the reported data-coverage rate).

The VLM only reads pixels into structured lines; picking the final amount and
deciding ambiguity is done here in plain Python so it is deterministic and
unit-tested (see tests/test_struk_ocr.py).
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

from shared.backend_client import push_struk_extraction
from shared.categories import normalize_category
from shared.gemini import extract_json

_PROMPT = """You are reading one retail receipt (struk) photo from a transit-station outlet in Indonesia.
Return ONE JSON object, no prose, with exactly these fields:
- "merchant_name": string - shop or brand name, "" if unreadable
- "amount_lines": array of {"label": string, "amount": number} for EVERY monetary line you can read
  (item prices, subtotal, discount, tax, total, amount paid, cash, change). Amounts in rupiah as plain
  numbers, no thousand separators or currency symbols.
- "payment_method": one of "cash", "qris", "debit", "credit", "ewallet", "" (use "" if not shown)
- "transacted_at": ISO-8601 timestamp printed on the receipt, or "" if none is legible
- "readable": boolean - false if the receipt is torn, blurred, or largely illegible
Never invent values you cannot see. Use "", 0, [] instead of guessing."""

# Label fragments (matched on a lowercased, whitespace-collapsed label) that mark
# the grand total actually paid, and fragments that must NOT be treated as it.
_FINAL_LABELS = (
    "grand total", "total akhir", "total bayar", "total pembayaran",
    "total dibayar", "jumlah dibayar", "total belanja", "total",
)
_NOT_FINAL = (
    "sub total", "subtotal", "sub-total",  # pre-tax
    "tunai", "cash", "kembali", "change", "kembalian",  # tendered / change
    "ppn", "pajak", "tax", "pb1", "service", "diskon", "discount", "hemat",
)


@dataclass
class StrukExtractionResult:
    job_id: str
    source_ref: str  # R2 object key
    station_id: str
    category: str
    final_amount: float
    payment_method: str
    transacted_at: str  # ISO 8601
    confidence: float
    is_ambiguous: bool
    photo_url: str = ""  # redacted public URL; set to also write a transparency card


def _clean(label: str) -> str:
    return " ".join((label or "").strip().lower().split())


def select_final_amount(amount_lines: list[dict]) -> tuple[float | None, bool]:
    """Pick the final amount paid from the VLM's monetary lines.

    Returns (amount, is_ambiguous). Ambiguous when no total-like line is present
    or when several total-like lines disagree — in both cases the caller drops
    the row from downstream math.
    """
    parsed: list[tuple[str, float]] = []
    for line in amount_lines or []:
        amount = line.get("amount")
        if amount is None:
            continue
        try:
            parsed.append((_clean(line.get("label", "")), float(amount)))
        except (TypeError, ValueError):
            continue

    totals = {
        round(amount, 2)
        for label, amount in parsed
        if amount > 0
        and any(marker in label for marker in _FINAL_LABELS)
        and not any(bad in label for bad in _NOT_FINAL)
    }

    if len(totals) == 1:
        return totals.pop(), False
    if len(totals) > 1:
        return max(totals), True
    return None, True  # no total-like line found — cannot determine the amount


def _now_iso() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


_PAYMENT_ALIASES = {
    "cash": "cash", "tunai": "cash",
    "qris": "qris", "qr": "qris",
    "debit": "debit", "kartu debit": "debit",
    "credit": "credit", "kredit": "credit", "kartu kredit": "credit",
    "ewallet": "ewallet", "e-wallet": "ewallet", "gopay": "ewallet",
    "ovo": "ewallet", "dana": "ewallet", "shopeepay": "ewallet",
}


def _norm_payment(raw: str) -> str:
    return _PAYMENT_ALIASES.get(_clean(raw), _clean(raw))


def extract_struk(image_path: str, station_id: str, job_id: str) -> StrukExtractionResult:
    raw = extract_json(_PROMPT, image_path)

    final_amount, ambiguous = select_final_amount(raw.get("amount_lines") or [])
    if not raw.get("readable", True):
        ambiguous = True
    ambiguous = ambiguous or final_amount is None

    return StrukExtractionResult(
        job_id=job_id,
        source_ref=Path(image_path).name,
        station_id=station_id,
        category=normalize_category(raw.get("merchant_name", "")),
        final_amount=float(final_amount or 0.0),
        payment_method=_norm_payment(raw.get("payment_method", "")),
        transacted_at=raw.get("transacted_at") or _now_iso(),
        confidence=0.9 if not ambiguous else 0.4,
        is_ambiguous=ambiguous,
    )


def run_batch(image_paths: list[str], station_id: str, job_id: str) -> None:
    for path in image_paths:
        result = extract_struk(path, station_id, job_id)
        push_struk_extraction(result.__dict__)
