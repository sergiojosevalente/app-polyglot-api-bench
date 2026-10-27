// Repository in-memory. Padanannya: store.go, store.rs, ItemStore.php
//
// TIDAK perlu mutex: JavaScript berjalan di SATU thread (event loop). Kode
// sinkron antara dua `await` tidak pernah disela, jadi Map aman tanpa lock.
// Harga yang dibayar: satu proses Node hanya memakai ~1 core untuk kode JS.

export interface Item {
  id: number;
  name: string;
  price: number;
}

export class ItemStore {
  readonly #items = new Map<number, Item>(); // `#` = private field asli JavaScript

  seed(n: number): void {
    for (let i = 1; i <= n; i++) {
      this.#items.set(i, { id: i, name: `item-${i}`, price: i * 1.5 });
    }
  }

  get(id: number): Item | undefined {
    return this.#items.get(id);
  }

  upsert(item: Item): Item {
    this.#items.set(item.id, item);
    return item;
  }

  count(): number {
    return this.#items.size;
  }
}
