import numpy as np
import pytest

from analysis import monte_carlo
from analysis.monte_carlo import (
    VariableDistribution,
    run_station_simulation,
    simulate_category_spending,
    summarize_p10_p90,
)


def _const(value: float) -> VariableDistribution:
    return VariableDistribution.from_values([value, value])


def test_simulate_category_spending_is_product_of_draws():
    out = simulate_category_spending(_const(100), _const(0.5), _const(0.4), _const(20000), n_iterations=256)
    assert out.shape == (256,)
    assert np.allclose(out, 100 * 0.5 * 0.4 * 20000)


def test_summarize_orders_low_then_high():
    low, high = summarize_p10_p90(np.arange(0, 100.0))
    assert low < high


def test_from_values_rejects_empty():
    with pytest.raises(ValueError):
        VariableDistribution.from_values([])


def test_run_station_simulation_pushes_gap_between_potential_and_captured(monkeypatch):
    captured_payloads = []
    monkeypatch.setattr(monte_carlo, "push_monte_carlo_result", captured_payloads.append)

    dists = {"makanan_minuman": (_const(1000), _const(0.5), _const(0.4), _const(25000))}
    smaller = {"makanan_minuman": (_const(200), _const(0.5), _const(0.4), _const(25000))}

    result = run_station_simulation("job1", "st1", "morning", dists, smaller, n_iterations=512)

    assert captured_payloads == [result]
    assert result["iterations"] == 512
    assert result["potential_low_p10"] > result["captured_high_p90"]
    assert result["potential_low_p10"] == pytest.approx(1000 * 0.5 * 0.4 * 25000)


def test_run_station_simulation_requires_potential():
    with pytest.raises(ValueError):
        run_station_simulation("j", "s", "morning", {}, {}, n_iterations=10)
