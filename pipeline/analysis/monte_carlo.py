"""Monte Carlo simulation of Potensi belanja = Σ category (F × E × C × V),
10,000 iterations, reporting P10-P90 per section 3.3. Results are pushed to the
Go API via shared.backend_client.push_monte_carlo_result.

Uncertainty is *propagated, not estimated*: F, E and V each carry an empirical
distribution built from repeated observation blocks, and they are drawn
together per iteration. C is the exception — it is the fixed
`PURCHASE_CONVERSION` product constant (0.95), not a distribution, so it
scales every draw instead of adding spread of its own. See shared/config.py
for why.
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

from shared.backend_client import push_monte_carlo_result
from shared.categories import BAKU_CATEGORIES
from shared.config import PURCHASE_CONVERSION, V_DEFAULT_BY_CATEGORY, settings

N_ITERATIONS = 10_000

# One (F, E, V) tuple of distributions per category. C is not in here: it is
# the PURCHASE_CONVERSION constant, applied inside simulate_category_spending.
CategoryDists = dict[str, tuple["VariableDistribution", ...]]


@dataclass
class VariableDistribution:
    """Empirical distribution for one variable (F, E, or V), built from
    repeated observation blocks (section 3.3: 'sebaran yang diperoleh dari
    pengulangan blok pengamatan')."""

    samples: np.ndarray

    @classmethod
    def from_values(cls, values) -> "VariableDistribution":
        arr = np.asarray(list(values), dtype=float)
        if arr.size == 0:
            raise ValueError("cannot build a distribution from zero samples")
        return cls(arr)

    def draw(self, n: int) -> np.ndarray:
        return np.random.choice(self.samples, size=n, replace=True)


def simulate_category_spending(
    f_dist: VariableDistribution,
    e_dist: VariableDistribution,
    v_dist: VariableDistribution,
    n_iterations: int = N_ITERATIONS,
) -> np.ndarray:
    """One category's F x E x C x V, with C held at PURCHASE_CONVERSION."""
    f = f_dist.draw(n_iterations)
    e = e_dist.draw(n_iterations)
    v = v_dist.draw(n_iterations)
    return f * e * PURCHASE_CONVERSION * v


def summarize_p10_p90(samples: np.ndarray) -> tuple[float, float]:
    return float(np.percentile(samples, 10)), float(np.percentile(samples, 90))


def _sum_across_categories(dists: CategoryDists, n_iterations: int) -> np.ndarray:
    total = np.zeros(n_iterations)
    for f, e, v in dists.values():
        total += simulate_category_spending(f, e, v, n_iterations)
    return total


def run_station_simulation(
    job_id: str,
    station_id: str,
    time_slot: str,
    potential_dists: CategoryDists,
    captured_dists: CategoryDists,
    n_iterations: int = N_ITERATIONS,
) -> dict:
    """Simulate potential vs captured spending for one station/time-slot and
    push the P10-P90 of each side. Both sides use the identical F×E×C×V
    instrument (methodology 3.3), so the API can read their difference as a
    real gap rather than a measurement artefact."""
    if not potential_dists:
        raise ValueError("no potential distributions — nothing to simulate")

    potential = _sum_across_categories(potential_dists, n_iterations)
    captured = (
        _sum_across_categories(captured_dists, n_iterations)
        if captured_dists
        else np.zeros(n_iterations)
    )

    p_low, p_high = summarize_p10_p90(potential)
    c_low, c_high = summarize_p10_p90(captured)

    payload = {
        "job_id": job_id,
        "station_id": station_id,
        "time_slot": time_slot,
        "potential_low_p10": p_low,
        "potential_high_p90": p_high,
        "captured_low_p10": c_low,
        "captured_high_p90": c_high,
        "iterations": n_iterations,
    }
    push_monte_carlo_result(payload)
    return payload


# --- Distribution loading from the operational DB -------------------------------
#
# F is a per-door flow shared by every category; E, C, V are per category.
# Potential uses entrance flow (flow_observations); captured uses the flow
# measured in front of the in-station gerai (passers_by in
# entry_conversion_observations). There is no in/out-of-station marker in the
# schema yet, so every entry_conversion row is treated as an in-station gerai
# per the field protocol (methodology 5.2). Refine once that flag exists.

_FLOW_SQL = """
    SELECT pedestrian_count FROM flow_observations
    WHERE station_id = %s AND time_slot = %s AND direction = 'in'
"""
_EC_SQL = """
    SELECT category, passers_by, entered_count, completed_purchase_count
    FROM entry_conversion_observations
    WHERE station_id = %s AND time_slot = %s AND entered_count > 0
"""
_V_SQL = """
    SELECT category, final_amount FROM struk_extractions
    WHERE station_id = %s AND is_ambiguous = false AND final_amount > 0
"""


def load_distributions_from_db(station_id: str, time_slot: str) -> tuple[CategoryDists, CategoryDists]:
    import psycopg  # deferred

    with psycopg.connect(settings.database_url) as conn:
        flow = [r[0] for r in conn.execute(_FLOW_SQL, (station_id, time_slot)).fetchall()]
        ec_rows = conn.execute(_EC_SQL, (station_id, time_slot)).fetchall()
        v_rows = conn.execute(_V_SQL, (station_id,)).fetchall()

    if not flow:
        raise ValueError(f"no flow observations for station {station_id} / {time_slot}")

    entrance_flow = VariableDistribution.from_values(flow)

    e_by_cat: dict[str, list[float]] = {c: [] for c in BAKU_CATEGORIES}
    gerai_flow_by_cat: dict[str, list[float]] = {c: [] for c in BAKU_CATEGORIES}
    # `completed_purchase_count` is selected and kept in the table on purpose —
    # it is still the field-measured comparison for the 0.95 assumption — but it
    # no longer feeds a C distribution.
    for category, passers, entered, _completed in ec_rows:
        if category not in e_by_cat:
            continue
        gerai_flow_by_cat[category].append(float(passers))
        e_by_cat[category].append(entered / passers if passers else 0.0)

    v_by_cat: dict[str, list[float]] = {c: [] for c in BAKU_CATEGORIES}
    for category, amount in v_rows:
        if category in v_by_cat:
            v_by_cat[category].append(float(amount))

    # Receipt OCR is parked ("datanya tidak ada"), so most categories have no
    # struk at all. Fall back to the documented, sourced default for the two
    # categories that have one rather than dropping them: without this the
    # whole station is unestimable and the pipeline publishes nothing. A
    # category with neither struk nor a default stays out — better absent than
    # invented.
    for category, default in V_DEFAULT_BY_CATEGORY.items():
        if category in v_by_cat and not v_by_cat[category]:
            v_by_cat[category].append(default)

    potential: CategoryDists = {}
    captured: CategoryDists = {}
    for category in BAKU_CATEGORIES:
        if not (e_by_cat[category] and v_by_cat[category]):
            continue  # category lacks the samples to simulate — contributes 0
        e = VariableDistribution.from_values(e_by_cat[category])
        v = VariableDistribution.from_values(v_by_cat[category])
        potential[category] = (entrance_flow, e, v)
        if gerai_flow_by_cat[category]:
            captured[category] = (
                VariableDistribution.from_values(gerai_flow_by_cat[category]), e, v,
            )

    return potential, captured


def run_from_db(job_id: str, station_id: str, time_slot: str, n_iterations: int = N_ITERATIONS) -> dict:
    potential, captured = load_distributions_from_db(station_id, time_slot)
    return run_station_simulation(job_id, station_id, time_slot, potential, captured, n_iterations)
