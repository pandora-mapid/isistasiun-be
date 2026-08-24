"""Entry point for batch pipeline jobs. Run manually or via cron/Celery
inside the `pipeline` container (see docker-compose.yml — the service just
sleeps by default, jobs are invoked explicitly):

    docker compose exec pipeline python main.py extract-struk --station-id ... --job-id ...

This keeps heavy processing (OCR, spatial analysis, Monte Carlo) fully
separate from the always-on Go API, per Brainstorm.md's batch pipeline design.
"""

import argparse
import sys


def main() -> None:
    parser = argparse.ArgumentParser(description="Isi Stasiun batch pipeline")
    sub = parser.add_subparsers(dest="command", required=True)

    p_struk = sub.add_parser("extract-struk", help="OCR batch for Struk Go photos")
    p_struk.add_argument("--station-id", required=True)
    p_struk.add_argument("--job-id", required=True)
    p_struk.add_argument("--images-dir", required=True)

    p_mc = sub.add_parser("simulate", help="Run Monte Carlo spending-gap simulation")
    p_mc.add_argument("--station-id", required=True)
    p_mc.add_argument("--job-id", required=True)
    p_mc.add_argument("--time-slot", required=True, choices=["morning", "midday", "evening", "night"])

    args = parser.parse_args()

    if args.command == "extract-struk":
        from extraction.struk_ocr import run_batch
        import glob

        images = glob.glob(f"{args.images_dir}/*.jpg") + glob.glob(f"{args.images_dir}/*.jpeg")
        run_batch(images, args.station_id, args.job_id)

    elif args.command == "simulate":
        print("TODO: wire run_station_simulation with real distributions", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
