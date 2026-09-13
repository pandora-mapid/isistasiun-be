import os
from dataclasses import dataclass

from dotenv import load_dotenv

load_dotenv()


@dataclass(frozen=True)
class Settings:
    geo_mapid_api_key: str = os.getenv("GEO_MAPID_API_KEY", "")
    geo_mapid_base_url: str = os.getenv("GEO_MAPID_BASE_URL", "https://api.geo.mapid.io")

    gemini_api_key: str = os.getenv("GEMINI_API_KEY", "")

    r2_account_id: str = os.getenv("R2_ACCOUNT_ID", "")
    r2_access_key_id: str = os.getenv("R2_ACCESS_KEY_ID", "")
    r2_secret_access_key: str = os.getenv("R2_SECRET_ACCESS_KEY", "")
    r2_bucket_name: str = os.getenv("R2_BUCKET_NAME", "isi-stasiun-media")

    backend_callback_base_url: str = os.getenv("BACKEND_CALLBACK_BASE_URL", "http://localhost:8080/api/v1")
    backend_callback_api_key: str = os.getenv("BACKEND_CALLBACK_API_KEY", "")

    database_url: str = os.getenv("DATABASE_URL", "")


settings = Settings()


# Konversi beli (C) di F x E x C x V. Keputusan produk, final: 95% pengunjung
# yang masuk ke gerai menyelesaikan pembelian. Bukan env var dan bukan
# distribusi yang ditarik dari `entry_conversion_observations` — dikunci di
# sini supaya satu angka yang sama dipakai pipeline, fixture AI, dan panel
# transparansi. `completed_purchase_count` tetap dikumpulkan di lapangan dan
# tetap tersimpan, tapi sebagai pembanding/QA, bukan sumber distribusi.
PURCHASE_CONVERSION = 0.95


# Nilai transaksi (V) di F x E x C x V, rupiah per pembelian.
#
# Sumber utama V adalah OCR struk (`struk_extractions`). Jalur itu berstatus
# PARKIR — "datanya tidak ada" (AI/isi-stasiun-ai-integration.md 1) — jadi
# angka di bawah dipakai sebagai cadangan untuk kategori yang belum punya
# satu pun struk terbaca. Dokumen yang sama sudah mengunci keduanya sebagai
# "default yang tampak & bisa disetel", dengan dasar publik:
#
#   makanan_minuman  ~Rp25.000  nilai transaksi grab-and-go transit
#   ritel_kemasan    ~Rp30.000  band minimarket transit Rp20-40rb
#                               (BUKAN Rp426rb Populix = belanja stok mingguan)
#
# Kategori lain sengaja tidak diberi default: tidak ada angka bersumber untuk
# apotek/jasa/lainnya, dan menebaknya akan memalsukan sumber. Kategori tanpa
# struk dan tanpa default di sini tidak ikut disimulasikan.
#
# Setiap rupiah yang lahir dari angka ini WAJIB tampil bertanda estimasi di
# UI (FE mengeksposnya sebagai slider), bukan sebagai hasil pengukuran.
V_DEFAULT_BY_CATEGORY: dict[str, float] = {
    "makanan_minuman": 25_000.0,
    "ritel_kemasan": 30_000.0,
}
