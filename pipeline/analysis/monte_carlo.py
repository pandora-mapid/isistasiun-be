"""Monte Carlo simulation of Potensi belanja = Σ category (F × E × C × V),
10,000 iterations, reporting P10-P90 per section 3.3. Results are pushed to
the Go API via shared.backend_client.push_monte_carlo_result.
"""

from dataclasses import dataclass

import numpy as np

from shared.backend_client import push_monte_carlo_result

N_ITERATIONS = 10_000


@dataclass
class VariableDistribution:
    """Empirical distribution for one variable (F, E, C, or V), built from
    repeated observation blocks (section 3.3: 'sebaran yang diperoleh dari
    pengulangan blok pengamatan')."""
    samples: np.ndarray

    def draw(self, n: int) -> np.ndarray:
        return np.random.choice(self.samples, size=n, replace=True)


def simulate_category_spending(
    f_dist: VariableDistribution,
    e_dist: VariableDistribution,
    c_dist: VariableDistribution,
    v_dist: VariableDistribution,
    n_iterations: int = N_ITERATIONS,
) -> np.ndarray:
    f = f_dist.draw(n_iterations)
    e = e_dist.draw(n_iterations)
    c = c_dist.draw(n_iterations)
    v = v_dist.draw(n_iterations)
    return f * e * c * v


def summarize_p10_p90(samples: np.ndarray) -> tuple[float, float]:
    return float(np.percentile(samples, 10)), float(np.percentile(samples, 90))


def run_station_simulation(job_id: str, station_id: str, time_slot: str, category_dists: dict) -> None:
    """category_dists: {category_name: (f_dist, e_dist, c_dist, v_dist)} for
    potential, plus a matching set scoped to in-station gerai for captured.
    Sums across categories per 'Potensi belanja simpul = Σ kategori(...)'.
    """
    # TODO: wire in real per-category distributions from survey +
    # struk_extractions tables, run potential vs captured separately, then
    # push the combined result.
    raise NotImplementedError
