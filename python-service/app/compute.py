"""Business logic murni tanpa pengetahuan HTTP.

Padanannya: compute.go, compute.rs, Compute.php, compute.ts, Compute.java
"""

from dataclasses import dataclass

MAX_FIB = 35


def fib(n: int) -> int:
    """Sengaja rekursif naif: benchmark CPU. CPython adalah interpreter bytecode
    tanpa JIT (default), jadi di sini Python paling terasa lambat."""
    return n if n < 2 else fib(n - 1) + fib(n - 2)


# dataclass = class data dengan __init__/__eq__/__repr__ otomatis (mirip record Java).
@dataclass(frozen=True, slots=True)
class Stats:
    count: int
    sum: float
    min: float
    max: float
    mean: float


def summarize(nums: list[float]) -> Stats | None:
    """`Stats | None` = Option<Stats>. Type hint TIDAK dicek saat runtime;
    hanya oleh tool seperti mypy/pyright."""
    if not nums:
        return None
    # Loop eksplisit, bukan sum(): sejak Python 3.12 sum() untuk float memakai
    # penjumlahan terkompensasi (Neumaier) sehingga hasilnya bisa beda tipis dari
    # bahasa lain yang menjumlah berurutan.
    total = 0.0
    lo = float("inf")
    hi = float("-inf")
    for n in nums:
        total += n
        lo = min(lo, n)
        hi = max(hi, n)
    return Stats(count=len(nums), sum=total, min=lo, max=hi, mean=total / len(nums))
