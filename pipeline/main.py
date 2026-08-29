"""Entry point for batch pipeline jobs. Run manually or via cron/Celery inside
the `pipeline` container (see docker-compose.yml — the service just sleeps by
default, jobs are invoked explicitly):

    docker compose exec pipeline python main.py extract-struk --station-id ... --job-id ... --images-dir ...
    docker compose exec pipeline python main.py extract-properti --station-id ... --job-id ... --images-dir ... [--manifest m.json]
    docker compose exec pipeline python main.py extract-gerai --station-id ... --job-id ... --images-dir ... [--manifest m.json]
    docker compose exec pipeline python main.py simulate --station-id ... --job-id ... --time-slot morning

Heavy processing (OCR, spatial analysis, Monte Carlo) stays fully separate from
the always-on Go API, per Brainstorm.md's batch pipeline design.
"""

from __future__ import annotations

import argparse
import glob
import sys

IMAGE_GLOBS = ("*.jpg", "*.jpeg", "*.png")


def _images(images_dir: str) -> list[str]:
    found: list[str] = []
    for pattern in IMAGE_GLOBS:
        found.extend(glob.glob(f"{images_dir}/{pattern}"))
    return sorted(found)


def _add_extract_args(parser: argparse.ArgumentParser, *, with_manifest: bool) -> None:
    parser.add_argument("--station-id", required=True)
    parser.add_argument("--job-id", required=True)
    parser.add_argument("--images-dir", required=True)
    if with_manifest:
        parser.add_argument("--manifest", default=None, help="JSON mapping filename -> {gerai_id|plot_id}")


def main() -> None:
    parser = argparse.ArgumentParser(description="Isi Stasiun batch pipeline")
    sub = parser.add_subparsers(dest="command", required=True)

    _add_extract_args(sub.add_parser("extract-struk", help="OCR batch for Struk Go photos"), with_manifest=False)
    _add_extract_args(sub.add_parser("extract-properti", help="OCR batch for Properti Go banners"), with_manifest=True)
    _add_extract_args(sub.add_parser("extract-gerai", help="Visual classification of gerai photos"), with_manifest=True)

    p_mc = sub.add_parser("simulate", help="Run Monte Carlo spending-gap simulation")
    p_mc.add_argument("--station-id", required=True)
    p_mc.add_argument("--job-id", required=True)
    p_mc.add_argument("--time-slot", required=True, choices=["morning", "midday", "evening", "night"])

    args = parser.parse_args()

    if args.command == "extract-struk":
        from extraction.struk_ocr import run_batch

        run_batch(_images(args.images_dir), args.station_id, args.job_id)

    elif args.command == "extract-properti":
        from extraction.properti_ocr import run_batch
        from shared.manifest import load_manifest

        run_batch(_images(args.images_dir), args.station_id, args.job_id, load_manifest(args.manifest))

    elif args.command == "extract-gerai":
        from extraction.gerai_classifier import run_batch
        from shared.manifest import load_manifest

        run_batch(_images(args.images_dir), args.station_id, args.job_id, load_manifest(args.manifest))

    elif args.command == "simulate":
        from analysis.monte_carlo import run_from_db

        result = run_from_db(args.job_id, args.station_id, args.time_slot)
        print(
            f"pushed spending gap for {args.station_id}/{args.time_slot}: "
            f"potential P10-P90 {result['potential_low_p10']:.0f}-{result['potential_high_p90']:.0f}, "
            f"captured P10-P90 {result['captured_low_p10']:.0f}-{result['captured_high_p90']:.0f}",
            file=sys.stderr,
        )


if __name__ == "__main__":
    main()
