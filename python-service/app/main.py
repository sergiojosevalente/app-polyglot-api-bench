"""Entry point: membuat aplikasi ASGI. Uvicorn (lihat Dockerfile) yang menjalankan
server HTTP + event loop dan menangani SIGTERM untuk graceful shutdown.

Padanannya: main.go, main.rs, main.ts, JavaServiceApplication.java
"""

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse
from starlette.exceptions import HTTPException as StarletteHTTPException

from app.api import router
from app.errors import ApiError
from app.metrics import Metrics, RequestCounterMiddleware
from app.store import ItemStore


def create_app() -> FastAPI:
    metrics = Metrics()

    # lifespan = kode startup/shutdown aplikasi (padanan wiring di main()).
    @asynccontextmanager
    async def lifespan(app: FastAPI) -> AsyncIterator[None]:
        store = ItemStore()
        store.seed(1000)
        app.state.store = store
        app.state.metrics = metrics
        print("python-service ready", flush=True)
        yield
        print("python-service stopped", flush=True)

    # docs_url=None: matikan Swagger UI bawaan (berguna saat dev, tidak perlu di benchmark).
    app = FastAPI(lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)
    app.include_router(router)
    app.add_middleware(RequestCounterMiddleware, metrics=metrics)

    # Error handling terpusat: semua error menjadi {"error": "..."} seperti bahasa lain.
    @app.exception_handler(ApiError)
    async def api_error(_: Request, exc: ApiError) -> JSONResponse:
        return JSONResponse({"error": exc.message}, status_code=exc.status)

    # Default FastAPI: 422 + {"detail": [...]}. Disamakan menjadi 400.
    @app.exception_handler(RequestValidationError)
    async def validation_error(_: Request, exc: RequestValidationError) -> JSONResponse:
        first = exc.errors()[0] if exc.errors() else {}
        loc = ".".join(str(p) for p in first.get("loc", ()))
        return JSONResponse({"error": f"invalid request: {loc}: {first.get('msg', '')}"}, status_code=400)

    @app.exception_handler(StarletteHTTPException)
    async def http_error(_: Request, exc: StarletteHTTPException) -> JSONResponse:
        return JSONResponse({"error": str(exc.detail).lower()}, status_code=exc.status_code)

    return app


app = create_app()
