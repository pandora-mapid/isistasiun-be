from extraction.struk_ocr import select_final_amount


def test_single_total_line_is_unambiguous():
    lines = [
        {"label": "Subtotal", "amount": 45000},
        {"label": "PPN 11%", "amount": 4950},
        {"label": "TOTAL", "amount": 49950},
        {"label": "Tunai", "amount": 50000},
        {"label": "Kembali", "amount": 50},
    ]
    amount, ambiguous = select_final_amount(lines)
    assert amount == 49950
    assert ambiguous is False


def test_subtotal_and_tender_are_not_treated_as_final():
    lines = [
        {"label": "Sub Total", "amount": 20000},
        {"label": "Cash", "amount": 20000},
    ]
    amount, ambiguous = select_final_amount(lines)
    assert amount is None
    assert ambiguous is True


def test_conflicting_totals_are_ambiguous():
    lines = [
        {"label": "Total Belanja", "amount": 30000},
        {"label": "Total Pembayaran", "amount": 27000},
    ]
    amount, ambiguous = select_final_amount(lines)
    assert amount == 30000  # best guess returned, but flagged
    assert ambiguous is True


def test_no_lines_at_all_is_ambiguous():
    amount, ambiguous = select_final_amount([])
    assert amount is None
    assert ambiguous is True


def test_garbled_amounts_are_skipped():
    lines = [
        {"label": "Total", "amount": "??"},
        {"label": "Grand Total", "amount": 15000},
    ]
    amount, ambiguous = select_final_amount(lines)
    assert amount == 15000
    assert ambiguous is False
