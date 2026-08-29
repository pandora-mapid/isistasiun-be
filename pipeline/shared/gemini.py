"""Thin wrapper around the Gemini vision-language API used for batch extraction
(OCR of struk/spanduk, visual classification of gerai). One entry point,
``extract_json``, sends an image + instruction and returns parsed structured
output. Confidence and ambiguity handling live in the calling module, not here.

The google-generativeai import is deferred so that pure-logic modules (and
their tests) can import the extraction package without the SDK installed.
"""

from __future__ import annotations

import json
import mimetypes
from pathlib import Path
from typing import Any

from shared.config import settings

DEFAULT_MODEL = "gemini-1.5-flash"


class GeminiError(RuntimeError):
    pass


def _model(model_name: str):
    if not settings.gemini_api_key:
        raise GeminiError("GEMINI_API_KEY is not set")
    import google.generativeai as genai  # deferred: heavy, network-only path

    genai.configure(api_key=settings.gemini_api_key)
    return genai.GenerativeModel(model_name)


def extract_json(
    instruction: str,
    image_path: str,
    *,
    model_name: str = DEFAULT_MODEL,
) -> dict[str, Any]:
    """Run the VLM on one image and return its JSON object response.

    ``instruction`` must tell the model to answer with a single JSON object and
    describe every field. Raises GeminiError on a missing key, an empty
    response, or output that is not a JSON object.
    """
    path = Path(image_path)
    data = path.read_bytes()
    mime = mimetypes.guess_type(path.name)[0] or "image/jpeg"

    model = _model(model_name)
    response = model.generate_content(
        [instruction, {"mime_type": mime, "data": data}],
        generation_config={"response_mime_type": "application/json", "temperature": 0.0},
    )

    text = (getattr(response, "text", "") or "").strip()
    if not text:
        raise GeminiError(f"empty response for {path.name}")
    try:
        parsed = json.loads(text)
    except json.JSONDecodeError as exc:
        raise GeminiError(f"non-JSON response for {path.name}: {exc}") from exc
    if not isinstance(parsed, dict):
        raise GeminiError(f"expected a JSON object for {path.name}, got {type(parsed).__name__}")
    return parsed
