"""Spatial join between transaction points (Struk Go), business points (Menu
Go), and catchment isochrones to build category-composition-of-demand per
station (section 3.3).
"""

import geopandas as gpd


def join_transactions_to_catchment(
    transactions_gdf: gpd.GeoDataFrame, catchment_gdf: gpd.GeoDataFrame
) -> gpd.GeoDataFrame:
    """TODO: gpd.sjoin(transactions_gdf, catchment_gdf, predicate='within')
    then group by category to get demand composition.
    """
    raise NotImplementedError
