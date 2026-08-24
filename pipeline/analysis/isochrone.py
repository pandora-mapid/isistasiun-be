"""Pedestrian isochrone (3, 5, 10 min) from each station entrance, forming
the catchment area boundary used for spatial join against transaction/POI
points (section 3.3).
"""

import geopandas as gpd
from shapely.geometry import Point

WALKING_SPEED_M_PER_MIN = 80  # ~4.8 km/h, conservative pedestrian speed


def build_isochrone(entrance_point: Point, minutes: int, network_gdf: gpd.GeoDataFrame) -> gpd.GeoDataFrame:
    """TODO: network-based isochrone using the pedestrian/road graph
    (network_gdf) rather than a naive buffer, once GEO MAPID's jaringan
    jalan dataset is loaded. Naive buffer left as a placeholder fallback.
    """
    radius_m = WALKING_SPEED_M_PER_MIN * minutes
    buffer = entrance_point.buffer(radius_m / 111_320)  # rough deg conversion
    return gpd.GeoDataFrame({"minutes": [minutes]}, geometry=[buffer], crs="EPSG:4326")
