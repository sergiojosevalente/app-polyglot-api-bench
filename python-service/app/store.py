"""Repository in-memory. Padanannya: store.go, store.rs, ItemStore.java, store.ts

Tanpa lock: semua handler yang memakai store adalah `async def`, sehingga berjalan
di SATU thread event loop (sama seperti Node). Kalau handler diubah menjadi `def`
biasa, FastAPI menjalankannya di threadpool dan Anda perlu threading.Lock. Ini wajib
terutama di Python free-threaded (3.13t, tanpa GIL).
"""

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class Item:
    id: int
    name: str
    price: float


class ItemStore:
    def __init__(self) -> None:
        self._items: dict[int, Item] = {}  # prefix _ = "private" (hanya konvensi)

    def seed(self, n: int) -> None:
        for i in range(1, n + 1):
            self._items[i] = Item(id=i, name=f"item-{i}", price=i * 1.5)

    def get(self, item_id: int) -> Item | None:
        return self._items.get(item_id)

    def upsert(self, item: Item) -> Item:
        self._items[item.id] = item
        return item

    def count(self) -> int:
        return len(self._items)
