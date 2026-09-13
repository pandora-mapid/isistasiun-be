from shared.categories import BAKU_CATEGORIES, normalize_category


def test_known_keywords_map_to_baku_categories():
    assert normalize_category("Kopi Kenangan") == "makanan_minuman"
    assert normalize_category("Kimia Farma Apotek") == "apotek_kesehatan"
    assert normalize_category("Indomaret Point") == "ritel_kemasan"
    assert normalize_category("Laundry Express") == "jasa"


def test_baku_category_passthrough():
    for category in BAKU_CATEGORIES:
        assert normalize_category(category) == category


def test_unknown_and_empty_fall_back_to_lainnya():
    assert normalize_category("Toko Bunga Melati") == "lainnya"
    assert normalize_category("") == "lainnya"
    assert normalize_category("   ") == "lainnya"


def test_case_and_whitespace_insensitive():
    assert normalize_category("  ALFAMART   ") == "ritel_kemasan"
