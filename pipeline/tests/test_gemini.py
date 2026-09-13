import pytest

from shared.gemini import GeminiError, _generate_with_retry


class _FlakyModel:
    """Fails `fail_times` times with a transient error, then succeeds."""

    def __init__(self, fail_times: int):
        self.fail_times = fail_times
        self.calls = 0

    def generate_content(self, *_args, **_kwargs):
        self.calls += 1
        if self.calls <= self.fail_times:
            raise RuntimeError("503 Service Unavailable")
        return "ok"


class _AlwaysFailingModel:
    def __init__(self):
        self.calls = 0

    def generate_content(self, *_args, **_kwargs):
        self.calls += 1
        raise RuntimeError("429 Rate Limited")


def _no_sleep(monkeypatch):
    monkeypatch.setattr("shared.gemini.time.sleep", lambda _seconds: None)


def test_succeeds_after_transient_failures(monkeypatch):
    _no_sleep(monkeypatch)
    model = _FlakyModel(fail_times=2)

    result = _generate_with_retry(
        model, "instruction", "image/jpeg", b"data", "receipt.jpg",
        max_retries=3, backoff_seconds=0.01,
    )

    assert result == "ok"
    assert model.calls == 3


def test_gives_up_after_max_retries(monkeypatch):
    _no_sleep(monkeypatch)
    model = _AlwaysFailingModel()

    with pytest.raises(GeminiError, match="failed after 3 attempts"):
        _generate_with_retry(
            model, "instruction", "image/jpeg", b"data", "receipt.jpg",
            max_retries=2, backoff_seconds=0.01,
        )

    assert model.calls == 3  # initial attempt + 2 retries
