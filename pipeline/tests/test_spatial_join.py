from analysis.spatial_join import demand_composition, missing_categories


def test_demand_composition_normalises_and_returns_all_five():
    comp = demand_composition(["Kopi Kenangan", "Alfamart", "Alfamart", "Guardian"])
    assert set(comp) == {"makanan_minuman", "ritel_kemasan", "apotek_kesehatan", "jasa", "lainnya"}
    assert comp["ritel_kemasan"] == 0.5
    assert comp["makanan_minuman"] == 0.25
    assert comp["apotek_kesehatan"] == 0.25
    assert comp["jasa"] == 0.0
    assert abs(sum(comp.values()) - 1.0) < 1e-9


def test_demand_composition_empty_is_all_zero():
    comp = demand_composition([])
    assert set(comp.values()) == {0.0}


def test_missing_categories_flags_unserved_demand_above_threshold():
    demand = {
        "makanan_minuman": 0.5,
        "ritel_kemasan": 0.3,
        "apotek_kesehatan": 0.15,
        "jasa": 0.02,
        "lainnya": 0.03,
    }
    missing = missing_categories(demand, available_in_station=["Kopi Kenangan"])
    assert missing == ["ritel_kemasan", "apotek_kesehatan"]


def test_missing_categories_ignores_lainnya_and_below_threshold():
    demand = {"makanan_minuman": 0.9, "ritel_kemasan": 0.0, "apotek_kesehatan": 0.01, "jasa": 0.0, "lainnya": 0.09}
    assert missing_categories(demand, available_in_station=["Kopi Kenangan"]) == []
