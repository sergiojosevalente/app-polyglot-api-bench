"""Model request/response (Pydantic). Validasi bersifat DEKLARATIF: tipe dan aturan
ditulis di model; FastAPI menolak request yang tidak cocok sebelum handler dipanggil.
Mirip serde di Rust, tetapi dicek saat runtime, bukan saat compile.
"""

from pydantic import BaseModel, ConfigDict, Field, field_validator


class StrictModel(BaseModel):
    # Default Pydantic "lax": "12" (string) diterima sebagai angka. Strict = seperti Go/Rust.
    model_config = ConfigDict(strict=True)


class HealthResponse(BaseModel):
    status: str
    lang: str


class HelloResponse(BaseModel):
    message: str
    lang: str


class CpuResponse(BaseModel):
    n: int
    result: int
    lang: str


class StatsRequest(StrictModel):
    numbers: list[float] = Field(min_length=1)


class StatsResponse(BaseModel):
    count: int
    sum: float
    min: float
    max: float
    mean: float
    lang: str


class UpsertItemRequest(StrictModel):
    name: str
    price: float = Field(ge=0)

    @field_validator("name")
    @classmethod
    def name_not_blank(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("name must not be empty")
        return v


class ItemResponse(BaseModel):
    id: int
    name: str
    price: float


class MetricsResponse(BaseModel):
    lang: str
    version: str
    uptime_sec: float
    requests_total: int
    rss_bytes: int
    cpu_seconds: float
    threads: int
    items: int
    runtime: dict[str, object]
