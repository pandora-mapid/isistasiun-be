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
