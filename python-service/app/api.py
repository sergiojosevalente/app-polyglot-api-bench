"""Lapisan HTTP. Padanannya: handler.go, handlers.rs, Handler.php, handlers.ts, ApiController.java"""

import asyncio
import gc
import platform
import sys
import threading
from typing import Annotated

import fastapi
from fastapi import APIRouter, Depends, Request

from app import compute
from app.errors import ApiError
from app.metrics import Metrics, read_proc
from app.schemas import (
    CpuResponse,
    HealthResponse,
    HelloResponse,
    ItemResponse,
    MetricsResponse,
    StatsRequest,
    StatsResponse,
    UpsertItemRequest,
)
from app.store import Item, ItemStore

LANG = "python"
DEFAULT_FIB_N = 25

router = APIRouter()


# Dependency injection ala FastAPI: Depends() memanggil fungsi ini per request.
def get_store(request: Request) -> ItemStore:
    return request.app.state.store


def get_metrics(request: Request) -> Metrics:
    return request.app.state.metrics


StoreDep = Annotated[ItemStore, Depends(get_store)]
MetricsDep = Annotated[Metrics, Depends(get_metrics)]


def to_response(item: Item) -> ItemResponse:
    return ItemResponse(id=item.id, name=item.name, price=item.price)


@router.get("/health")
async def health() -> HealthResponse:
    return HealthResponse(status="ok", lang=LANG)


@router.get("/api/hello")
async def hello(name: str = "") -> HelloResponse:
    return HelloResponse(message=f"Hello, {name or 'World'}!", lang=LANG)


# SENGAJA `def`, bukan `async def`: FastAPI menjalankannya di threadpool supaya
# fib() tidak memblokir event loop seperti di Node. Tetapi karena GIL, hanya SATU
# thread yang mengeksekusi bytecode Python pada satu waktu, sehingga CPU tetap ~1 core.
@router.get("/api/cpu")
def cpu(n: int = DEFAULT_FIB_N) -> CpuResponse:
    if n < 0:
        raise ApiError.bad_request("n must be a non-negative integer")
    if n > compute.MAX_FIB:
        raise ApiError.bad_request(f"n must be <= {compute.MAX_FIB}")
    return CpuResponse(n=n, result=compute.fib(n), lang=LANG)


@router.post("/api/json")
async def json_stats(req: StatsRequest) -> StatsResponse:
    stats = compute.summarize(req.numbers)
    if stats is None:  # sudah dijamin min_length=1, tapi type checker minta dicek
        raise ApiError.bad_request("numbers must not be empty")
    return StatsResponse(
        count=stats.count, sum=stats.sum, min=stats.min, max=stats.max, mean=stats.mean, lang=LANG
    )


@router.get("/api/items/{item_id}")
async def get_item(item_id: int, store: StoreDep) -> ItemResponse:
    if item_id < 0:
        raise ApiError.bad_request("id must be a non-negative integer")
    item = store.get(item_id)
    if item is None:
        raise ApiError.not_found(f"item {item_id} not found")
    return to_response(item)


@router.put("/api/items/{item_id}")
async def put_item(item_id: int, req: UpsertItemRequest, store: StoreDep) -> ItemResponse:
    if item_id < 0:
        raise ApiError.bad_request("id must be a non-negative integer")
    return to_response(store.upsert(Item(id=item_id, name=req.name, price=req.price)))


@router.get("/metrics")
async def metrics(store: StoreDep, m: MetricsDep) -> MetricsResponse:
    ps = read_proc()
    gil = getattr(sys, "_is_gil_enabled", lambda: True)()  # 3.13+: False di build free-threaded
    return MetricsResponse(
        lang=LANG,
        version=f"Python {platform.python_version()} · FastAPI {fastapi.__version__}",
        uptime_sec=m.uptime_sec,
        requests_total=m.requests,
        rss_bytes=ps.rss_bytes,
        cpu_seconds=ps.cpu_seconds,
        threads=ps.threads,
        items=store.count(),
        runtime={
            "gil": gil,
            "python_threads": threading.active_count(),
            "asyncio_tasks": len(asyncio.all_tasks()),
            "gc_collections": sum(s["collections"] for s in gc.get_stats()),
        },
    )
