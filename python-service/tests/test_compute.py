import pytest

from app.compute import Stats, fib, summarize
from app.store import Item, ItemStore


# parametrize = padanan table-driven test di Go.
@pytest.mark.parametrize(("n", "want"), [(0, 0), (1, 1), (2, 1), (10, 55), (25, 75025)])
def test_fib(n: int, want: int) -> None:
    assert fib(n) == want


def test_summarize() -> None:
    assert summarize([1, 2, 3, 4.5, -2]) == Stats(count=5, sum=8.5, min=-2, max=4.5, mean=1.7)


def test_summarize_empty() -> None:
    assert summarize([]) is None


def test_store_seed_and_get() -> None:
    s = ItemStore()
    s.seed(10)
    assert s.get(7) == Item(id=7, name="item-7", price=10.5)
    assert s.get(99) is None
