"""Station-level rollup behind the free-tier Compare View.

One tier above `monte_carlo.run_station_simulation`, which answers "what does
one station look like on one time slot" and reports P10/P90 only. This module
answers "what does this station look like across the whole day", which the
`station_summary` table needs and the per-slot callback cannot express:

* P50 as well as P10/P90;
* a gap simulated in its own right — `potential_i - captured_i` per iteration,
  not `potential_p10 - captured_p90`. Subtracting percentiles of two
  distributions is not the percentile of their difference; it widens the range
  and can even invert the P10/P50/P90 ordering;
* the day rollup as the per-iteration sum across slots, so the slots' spread
  compounds the way the model says it does rather than being added after the
  fact;
* capture rate, confidence band, peak point and category composition.

Results are pushed to POST /pipeline/simulations/station-summary.
"""

from __future__ import annotations

import numpy as np

from analysis.monte_carlo import (
    N_ITERATIONS,
    CategoryDists,
    _sum_across_categories,
    load_distributions_from_db,
)
from shared.backend_client import push_station_summary
from shared.categories import BAKU_CATEGORIES
from shared.config import PURCHASE_CONVERSION, settings

# The four measured slots, in the order the day runs.
TIME_SLOTS = ("morning", "midday", "evening", "night")

# A category with fewer than this many gerai inside the station counts as
# missing. Mirrors AMBANG_GERAI in the frontend (lib/analytics/select.ts) so
# the map and the API agree on which categories are "kategori hilang"; the two
# are independent constants on purpose, but they must not drift apart silently.
AMBANG_GERAI = 3

BASIS_MONTE_CARLO = "monte-carlo-simpul"


def summarize_p10_p50_p90(samples: np.ndarray) -> dict[str, float]:
    p10, p50, p90 = np.percentile(samples, [10, 50, 90])
    return {"p10": float(p10), "p50": float(p50), "p90": float(p90)}


def _simulate_day(
    per_slot: list[tuple[CategoryDists, CategoryDists]], n_iterations: int
) -> tuple[np.ndarray, np.ndarray]:
    """Sum potential and captured across slots, iteration by iteration."""
    potential = np.zeros(n_iterations)
    captured = np.zeros(n_iterations)
    for potential_dists, captured_dists in per_slot:
        if potential_dists:
            potential += _sum_across_categories(potential_dists, n_iterations)
        if captured_dists:
            captured += _sum_across_categories(captured_dists, n_iterations)
    return potential, captured


_CONFIDENCE_SQL = """
    SELECT min(confidence_score), max(confidence_score)
    FROM confidence_layer WHERE station_id = %s
"""
_STRUK_SQL = """
    SELECT count(*) FROM struk_extractions
    WHERE station_id = %s AND is_ambiguous = false
"""
# Doors actually counted vs. doors that exist. `pintu_ditahan` is the honest
# half of the pair: doors the survey could not cover, which is why the station
# figure is a lower bound rather than a complete one.
_PINTU_SQL = """
    SELECT
      (SELECT count(DISTINCT entrance_id) FROM flow_observations WHERE station_id = %(sid)s),
      (SELECT count(*) FROM station_entrances WHERE station_id = %(sid)s)
"""
# Busiest counted door on each slot — the peak point's label.
_PEAK_DOOR_SQL = """
    SELECT se.label, f.time_slot, sum(f.pedestrian_count) AS flow
    FROM flow_observations f
    JOIN station_entrances se ON se.id = f.entrance_id
    WHERE f.station_id = %s AND f.direction = 'in'
    GROUP BY se.label, f.time_slot
    ORDER BY flow DESC
    LIMIT 1
"""
# Category composition: how many gerai the station has per category, and how
# much of the measured entering traffic each category draws (demand share).
_COMPOSITION_SQL = """
    SELECT category, count(DISTINCT gerai_id) AS gerai, sum(entered_count) AS entered
    FROM entry_conversion_observations
    WHERE station_id = %s
    GROUP BY category
"""
_PEAK_VARS_SQL = """
    SELECT avg(passers_by)::float, avg(entered_count::float / NULLIF(passers_by, 0))
    FROM entry_conversion_observations
    WHERE station_id = %s AND time_slot = %s
"""
_PEAK_V_SQL = """
    SELECT avg(final_amount)::float FROM struk_extractions
    WHERE station_id = %s AND is_ambiguous = false AND final_amount > 0
"""


def load_context_from_db(station_id: str) -> dict:
    """Everything in the rollup that is counted rather than simulated."""
    import psycopg  # deferred, same as monte_carlo

    with psycopg.connect(settings.database_url) as conn:
        conf_min, conf_max = conn.execute(_CONFIDENCE_SQL, (station_id,)).fetchone()
        (struk_terbaca,) = conn.execute(_STRUK_SQL, (station_id,)).fetchone()
        dicacah, total_pintu = conn.execute(_PINTU_SQL, {"sid": station_id}).fetchone()
        peak_door = conn.execute(_PEAK_DOOR_SQL, (station_id,)).fetchone()
        composition_rows = conn.execute(_COMPOSITION_SQL, (station_id,)).fetchall()

        peak_vars = (None, None)
        if peak_door:
            peak_vars = conn.execute(_PEAK_VARS_SQL, (station_id, peak_door[1])).fetchone()
        (peak_v,) = conn.execute(_PEAK_V_SQL, (station_id,)).fetchone()

    confidence = None
    if conf_min is not None and conf_max is not None:
        confidence = {"min": float(conf_min), "max": float(conf_max)}

    return {
        "confidence": confidence,
        "struk_terbaca": int(struk_terbaca or 0),
        "pintu_dicacah": int(dicacah or 0),
        "pintu_ditahan": max(int(total_pintu or 0) - int(dicacah or 0), 0),
        "peak_door": peak_door,
        "peak_vars": peak_vars,
        "peak_v": peak_v,
        "composition_rows": composition_rows,
    }


def build_composition(composition_rows) -> list[dict]:
    """One entry per baku category, including the ones with no gerai at all —
    those are exactly the 'kategori hilang' the map is meant to surface, so
    leaving them out would hide the finding."""
    by_category = {c: (0, 0.0) for c in BAKU_CATEGORIES}
    for category, gerai, entered in composition_rows:
        if category in by_category:
            by_category[category] = (int(gerai or 0), float(entered or 0))

    total_entered = sum(entered for _, entered in by_category.values())
    return [
        {
            "category": category,
            "demand_share": round(entered / total_entered, 4) if total_entered else 0.0,
            "gerai_count": gerai,
            "is_missing": gerai < AMBANG_GERAI,
        }
        for category, (gerai, entered) in by_category.items()
    ]


def build_peak(context: dict, gap_by_slot: dict[str, dict[str, float]]) -> dict | None:
    """The busiest counted door, carrying that slot's F/E/C/V and its own gap."""
    peak_door = context["peak_door"]
    if not peak_door:
        return None
    label, time_slot, flow = peak_door
    avg_passers, avg_entry = context["peak_vars"]
    return {
        "point_label": label,
        "time_slot": time_slot,
        "f": float(flow or avg_passers or 0),
        "e": round(float(avg_entry or 0), 4),
        "c": PURCHASE_CONVERSION,
        "v": float(context["peak_v"] or 0),
        "gap": gap_by_slot.get(time_slot, {"p10": 0.0, "p50": 0.0, "p90": 0.0}),
    }


def run_station_summary(
    job_id: str,
    station_id: str,
    day_type: str = "weekday",
    n_iterations: int = N_ITERATIONS,
    time_slots: tuple[str, ...] = TIME_SLOTS,
) -> dict:
    """Simulate the whole-station, whole-day rollup and push it to the API.

    Slots with no observations are skipped rather than counted as zero: a slot
    that was never surveyed is unknown, not empty, and folding it in as zero
    would quietly drag the station's potential down.
    """
    per_slot: list[tuple[CategoryDists, CategoryDists]] = []
    gap_by_slot: dict[str, dict[str, float]] = {}

    for time_slot in time_slots:
        try:
            potential_dists, captured_dists = load_distributions_from_db(station_id, time_slot)
        except ValueError:
            continue  # no flow observations for this slot — not surveyed
        if not potential_dists:
            continue
        per_slot.append((potential_dists, captured_dists))

        slot_potential = _sum_across_categories(potential_dists, n_iterations)
        slot_captured = (
            _sum_across_categories(captured_dists, n_iterations)
            if captured_dists
            else np.zeros(n_iterations)
        )
        gap_by_slot[time_slot] = summarize_p10_p50_p90(slot_potential - slot_captured)

    if not per_slot:
        raise ValueError(f"no estimable time slot for station {station_id}")

    potential, captured = _simulate_day(per_slot, n_iterations)
    gap = potential - captured

    potensi = summarize_p10_p50_p90(potential)
    tertangkap = summarize_p10_p50_p90(captured)
    capture_rate = (
        round(tertangkap["p50"] / potensi["p50"], 4) if potensi["p50"] > 0 else None
    )

    context = load_context_from_db(station_id)
    payload = {
        "job_id": job_id,
        "station_id": station_id,
        "day_type": day_type,
        "iterations": n_iterations,
        "potensi": potensi,
        "tertangkap": tertangkap,
        "gap": summarize_p10_p50_p90(gap),
        "capture_rate": capture_rate,
        "confidence": context["confidence"],
        "struk_terbaca": context["struk_terbaca"],
        "pintu_dicacah": context["pintu_dicacah"],
        "pintu_ditahan": context["pintu_ditahan"],
        "peak": build_peak(context, gap_by_slot),
        "composition": build_composition(context["composition_rows"]),
        "basis": BASIS_MONTE_CARLO,
    }
    push_station_summary(payload)
    return payload
