package com.gorust.javaservice.store;

import com.gorust.javaservice.domain.Item;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import org.springframework.stereotype.Repository;

/**
 * Repository in-memory. Padanannya: store.go (map + RWMutex), store.rs (RwLock<HashMap>).
 *
 * ConcurrentHashMap tidak memakai satu lock global: baca tanpa lock, tulis hanya
 * mengunci satu bucket. Untuk beban banyak-baca seperti skenario "store" ini
 * biasanya lebih skalabel daripada RWMutex.
 */
@Repository
public class ItemStore {

    private static final int SEED_COUNT = 1000;

    private final ConcurrentHashMap<Long, Item> items = new ConcurrentHashMap<>();

    public ItemStore() {
        seed(SEED_COUNT);
    }

    public void seed(int n) {
        for (long i = 1; i <= n; i++) {
            items.put(i, new Item(i, "item-" + i, i * 1.5));
        }
    }

    public Optional<Item> get(long id) {
        return Optional.ofNullable(items.get(id));
    }

    public Item upsert(Item item) {
        items.put(item.id(), item);
        return item;
    }

    public int count() {
        return items.size();
    }
}
