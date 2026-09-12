import numpy as np
import pytest

from analysis import station_summary
from analysis.monte_carlo import VariableDistribution
from analysis.station_summary import (
    AMBANG_GERAI,
    build_composition,
    build_peak,
    run_station_summary,
    summarize_p10_p50_p90,
)
from shared.categories import BAKU_CATEGORIES
from shared.config import PURCHASE_CONVERSION


def _const(value: float) -> VariableDistribution:
    return VariableDistribution.from_values([value, value])


def _dists(f: float, v: float = 20000.0):
    return {"makanan_minuman": (_const(f), _const(0.5), _const(v))}


_CONTEXT = {
    "confidence": {"min": 0.3, "max": 0.8},
    "struk_terbaca": 7,
    "pintu_dicacah": 2,
    "pintu_ditahan": 1,
    "peak_door": ("Pintu Bawah", "evening", 120.0),
    "peak_vars": (300.0, 0.04),
    "peak_v": 22000.0,
    "composition_rows": [("makanan_minuman", 4, 60), ("ritel_kemasan", 1, 40)],
}


def _patch_db(monkeypatch, by_slot: dict):
    pushed = []
    monkeypatch.setattr(station_summary, "push_station_summary", pushed.append)
    monkeypatch.setattr(station_summary, "load_context_from_db", lambda _sid: dict(_CONTEXT))

    def fake_load(_station_id, time_slot):
        if time_slot not in by_slot:
            raise ValueError("no flow observations")
        return by_slot[time_slot]

    monkeypatch.setattr(station_summary, "load_distributions_from_db", fake_load)
    return pushed


def test_summarize_returns_ordered_percentiles():
    out = summarize_p10_p50_p90(np.arange(0.0, 100.0))
    assert out["p10"] < out["p50"] < out["p90"]


def test_run_station_summary_pushes_a_full_payload(monkeypatch):
    pushed = _patch_db(monkeypatch, {"morning": (_dists(1000), _dists(200))})

    result = run_station_summary("job1", "st1", n_iterations=10_000)

    assert pushed == [result]
    assert result["basis"] == "monte-carlo-simpul"
    assert result["iterations"] == 10_000
    assert result["potensi"]["p50"] == pytest.approx(1000 * 0.5 * PURCHASE_CONVERSION * 20000)
    assert result["tertangkap"]["p50"] == pytest.approx(200 * 0.5 * PURCHASE_CONVERSION * 20000)
    assert result["gap"]["p50"] == pytest.approx(
        result["potensi"]["p50"] - result["tertangkap"]["p50"]
    )
    assert result["capture_rate"] == pytest.approx(0.2)
    assert result["confidence"] == {"min": 0.3, "max": 0.8}
    assert result["pintu_dicacah"] == 2 and result["pintu_ditahan"] == 1


# The whole reason this module exists rather than reusing the per-slot
# callback: the day figure must be the sum across slots, not one slot.
def test_day_rollup_sums_across_slots(monkeypatch):
    _patch_db(monkeypatch, {"morning": (_dists(1000), {}), "evening": (_dists(500), {})})

    result = run_station_summary("job1", "st1", n_iterations=10_000)

    assert result["potensi"]["p50"] == pytest.approx(1500 * 0.5 * PURCHASE_CONVERSION * 20000)


# A slot nobody surveyed is unknown, not zero — counting it as zero would drag
# the station's potential down and understate the gap.
def test_unsurveyed_slots_are_skipped_not_zeroed(monkeypatch):
    _patch_db(monkeypatch, {"morning": (_dists(1000), {})})

    result = run_station_summary("job1", "st1", n_iterations=10_000)

    assert result["potensi"]["p50"] == pytest.approx(1000 * 0.5 * PURCHASE_CONVERSION * 20000)


def test_run_station_summary_requires_one_estimable_slot(monkeypatch):
    _patch_db(monkeypatch, {})
    with pytest.raises(ValueError):
        run_station_summary("job1", "st1", n_iterations=10_000)


# Percentiles of a difference, not a difference of percentiles: with spread on
# both sides the naive p10 - p90 subtraction reports a wider (and lower) floor
# than the simulated gap actually has.
def test_gap_is_simulated_not_subtracted_percentiles(monkeypatch):
    spread = {
        "makanan_minuman": (
            VariableDistribution.from_values(np.linspace(500, 1500, 101)), _const(0.5), _const(20000),
        )
    }
    smaller = {
        "makanan_minuman": (
            VariableDistribution.from_values(np.linspace(100, 300, 101)), _const(0.5), _const(20000),
        )
    }
    _patch_db(monkeypatch, {"morning": (spread, smaller)})

    result = run_station_summary("job1", "st1", n_iterations=20_000)

    naive_p10 = result["potensi"]["p10"] - result["tertangkap"]["p90"]
    assert result["gap"]["p10"] > naive_p10
    assert result["gap"]["p10"] <= result["gap"]["p50"] <= result["gap"]["p90"]


def test_composition_lists_every_baku_category_including_absent_ones():
    out = build_composition([("makanan_minuman", 4, 60), ("ritel_kemasan", 1, 40)])

    assert [c["category"] for c in out] == list(BAKU_CATEGORIES)
    by_cat = {c["category"]: c for c in out}
    assert by_cat["makanan_minuman"]["demand_share"] == pytest.approx(0.6)
    assert by_cat["makanan_minuman"]["is_missing"] is False
    # 1 gerai is below AMBANG_GERAI, so it still counts as a missing category.
    assert AMBANG_GERAI == 3
    assert by_cat["ritel_kemasan"]["is_missing"] is True
    assert by_cat["apotek_kesehatan"]["gerai_count"] == 0
    assert by_cat["apotek_kesehatan"]["is_missing"] is True


def test_composition_without_traffic_reports_zero_share_not_a_crash():
    out = build_composition([("makanan_minuman", 2, 0)])
    assert all(c["demand_share"] == 0.0 for c in out)


def test_peak_carries_the_locked_c_and_its_own_slot_gap():
    gap_by_slot = {"evening": {"p10": 10.0, "p50": 20.0, "p90": 30.0}}
    peak = build_peak(dict(_CONTEXT), gap_by_slot)

    assert peak["point_label"] == "Pintu Bawah"
    assert peak["time_slot"] == "evening"
    assert peak["c"] == PURCHASE_CONVERSION
    assert peak["gap"] == gap_by_slot["evening"]


def test_peak_is_none_when_no_door_was_counted():
    context = dict(_CONTEXT, peak_door=None)
    assert build_peak(context, {}) is None
