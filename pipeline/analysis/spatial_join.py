"""Spatial join between transaction points (Struk Go), business points (Menu
Go), and catchment isochrones to build the category composition of demand per
station (section 3.3).

The geopandas join and the pure composition maths are split so the latter can
be unit-tested without GDAL.
"""

from __future__ import annotations

from collections.abc import Iterable

from shared.categories import BAKU_CATEGORIES, normalize_category


def join_points_to_catchment(points_gdf, catchment_gdf, *, how: str = "inner"):
    """Return the subset of ``points_gdf`` that falls inside any catchment
    polygon, with the catchment attributes attached. Both frames must share a
    CRS. ``geopandas`` is imported lazily so this module stays importable
    without it."""
    import geopandas as gpd  # deferred: needs GDAL

    if points_gdf.crs != catchment_gdf.crs:
        raise ValueError(f"CRS mismatch: {points_gdf.crs} vs {catchment_gdf.crs}")
    return gpd.sjoin(points_gdf, catchment_gdf, how=how, predicate="within")


def demand_composition(categories: Iterable[str]) -> dict[str, float]:
    """Proportion of transactions in each of the five baku categories.

    Input is the raw category label of every transaction that landed in the
    catchment; labels are normalised here. Always returns all five keys, so a
    category with no demand reads as 0.0 rather than being absent.
    """
    counts = dict.fromkeys(BAKU_CATEGORIES, 0)
    total = 0
    for raw in categories:
        counts[normalize_category(raw)] += 1
        total += 1
    if total == 0:
        return {c: 0.0 for c in BAKU_CATEGORIES}
    return {c: counts[c] / total for c in BAKU_CATEGORIES}


def missing_categories(
    demand: dict[str, float],
    available_in_station: Iterable[str],
    *,
    demand_threshold: float = 0.05,
) -> list[str]:
    """Categories with real demand in the catchment (share >= threshold) that
    no in-station gerai currently serves — the 'kategori hilang' output."""
    available = {normalize_category(c) for c in available_in_station}
    return [
        category
        for category, share in demand.items()
        if share >= demand_threshold and category not in available and category != "lainnya"
    ]
