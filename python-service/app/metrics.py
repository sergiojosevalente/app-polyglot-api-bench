"""Counter request (middleware ASGI) + statistik proses dari /proc.

Padanannya: metrics.go, metrics.rs
"""

import time
from dataclasses import dataclass
from pathlib import Path

from starlette.types import ASGIApp, Receive, Scope, Send


class Metrics:
    def __init__(self) -> None:
        self.started = time.monotonic()
        # int biasa cukup: middleware berjalan di thread event loop saja.
        self.requests = 0

    @property
    def uptime_sec(self) -> float:
        return time.monotonic() - self.started


class RequestCounterMiddleware:
    """Middleware ASGI "murni". Lebih cepat daripada @app.middleware("http")
    (BaseHTTPMiddleware) yang membungkus setiap request dengan task tambahan."""

    def __init__(self, app: ASGIApp, metrics: Metrics) -> None:
        self.app = app
        self.metrics = metrics

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] == "http":
            self.metrics.requests += 1
        await self.app(scope, receive, send)


@dataclass(frozen=True, slots=True)
class ProcStats:
    rss_bytes: int = 0
    cpu_seconds: float = 0.0
    threads: int = 0


CLOCK_TICKS = 100.0
PAGE_SIZE = 4096


def read_proc() -> ProcStats:
    """Baca /proc/self/stat (Linux). Di OS lain hasilnya nol."""
    try:
        s = Path("/proc/self/stat").read_text()
    except OSError:
        return ProcStats()
    # Nama proses bisa berisi spasi, jadi mulai setelah ')' terakhir; field k = f[k-3].
    f = s[s.rfind(")") + 2 :].split()
    if len(f) < 22:
        return ProcStats()
    return ProcStats(
        rss_bytes=int(f[21]) * PAGE_SIZE,
        cpu_seconds=(int(f[11]) + int(f[12])) / CLOCK_TICKS,
        threads=int(f[17]),
    )
