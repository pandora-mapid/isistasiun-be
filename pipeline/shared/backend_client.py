"""Posts batch-pipeline results back to the Go API's /pipeline/* callback
endpoints (owned by Arzaka — see backend/internal/pipeline/). Every call
carries X-Service-Key so it passes middleware.RequireServiceKey.
"""

import requests

from shared.config import settings


def _post(path: str, payload: dict) -> dict:
    resp = requests.post(
        f"{settings.backend_callback_base_url}{path}",
        json=payload,
        headers={"X-Service-Key": settings.backend_callback_api_key},
        timeout=15,
    )
    resp.raise_for_status()
    return resp.json()


def push_struk_extraction(payload: dict) -> dict:
    return _post("/pipeline/extractions/struk", payload)


def push_properti_extraction(payload: dict) -> dict:
    return _post("/pipeline/extractions/properti", payload)


def push_gerai_classification(payload: dict) -> dict:
    return _post("/pipeline/extractions/gerai", payload)


def push_monte_carlo_result(payload: dict) -> dict:
    return _post("/pipeline/simulations/monte-carlo", payload)
