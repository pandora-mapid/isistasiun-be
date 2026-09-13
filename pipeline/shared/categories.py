"""The five fixed (baku) business categories and normalisation of free-text
merchant/product names onto them, per methodology 3.3-3.4:

    "Penyeragaman kategori juga tidak realistis dikerjakan manual pada ratusan
     record ... padahal keseragaman itu prasyarat agar komposisi antar-simpul
     dapat diperbandingkan."

normalize_category is deliberately rule-based and offline so it stays cheap,
deterministic, and testable. The VLM is only asked for a raw label; mapping it
to a baku category happens here.
"""

from __future__ import annotations

MAKANAN_MINUMAN = "makanan_minuman"
RITEL_KEMASAN = "ritel_kemasan"
APOTEK_KESEHATAN = "apotek_kesehatan"
JASA = "jasa"
LAINNYA = "lainnya"

BAKU_CATEGORIES: tuple[str, ...] = (
    MAKANAN_MINUMAN,
    RITEL_KEMASAN,
    APOTEK_KESEHATAN,
    JASA,
    LAINNYA,
)

# Keyword -> baku category. Checked as substrings against a lowercased,
# whitespace-collapsed label. Order matters only for readability; the first
# matching category wins in _match.
_KEYWORDS: dict[str, tuple[str, ...]] = {
    MAKANAN_MINUMAN: (
        "makan", "minum", "food", "beverage", "cafe", "kopi", "coffee", "resto",
        "restaurant", "bakery", "roti", "snack", "jajan", "kuliner", "warung",
        "kedai", "boba", "juice", "es ", "teh", "ayam", "nasi", "mie", "bakso",
        "donut", "donat", "ice cream", "gelato", "milk", "susu",
    ),
    APOTEK_KESEHATAN: (
        "apotek", "apotik", "pharmacy", "farma", "obat", "kimia farma", "guardian",
        "watson", "century", "klinik", "clinic", "health", "kesehatan", "optik",
        "vitamin", "medical",
    ),
    RITEL_KEMASAN: (
        "minimarket", "supermarket", "swalayan", "retail", "ritel", "grocery",
        "indomaret", "alfamart", "alfamidi", "circle k", "lawson", "familymart",
        "convenience", "kemasan", "toko kelontong", "kelontong", "mart",
    ),
    JASA: (
        "jasa", "service", "servis", "laundry", "salon", "barber", "cukur",
        "spa", "atm", "bank", "money changer", "tukar uang", "print", "fotokopi",
        "fotocopy", "travel", "tour", "pijat", "reflexology", "reflexi",
        "phone", "pulsa", "konter", "counter", "repair", "reparasi", "loker",
        "locker", "charging",
    ),
}


def _clean(label: str) -> str:
    return " ".join((label or "").strip().lower().split())


def normalize_category(label: str) -> str:
    """Map a free-text merchant/product/business label to one of the five baku
    categories. Anything unrecognised falls back to ``lainnya`` — never guessed.
    An input that is already a baku category is returned unchanged."""
    cleaned = _clean(label)
    if cleaned in BAKU_CATEGORIES:
        return cleaned
    if not cleaned:
        return LAINNYA
    for category, keywords in _KEYWORDS.items():
        if any(kw in cleaned for kw in keywords):
            return category
    return LAINNYA


def is_baku(category: str) -> bool:
    return category in BAKU_CATEGORIES
