//! Repository in-memory yang aman dipakai banyak thread.
//! Padanannya: go-service/internal/store/store.go

use std::{collections::HashMap, sync::RwLock};

use serde::Serialize;

#[derive(Debug, Clone, PartialEq, Serialize)]
pub struct Item {
    pub id: u64,
    pub name: String,
    pub price: f64,
}

/// Beda penting dengan Go: di Rust data berada DI DALAM lock (`RwLock<HashMap>`),
/// jadi mustahil mengakses map tanpa lock — dicek compiler, bukan `-race` saat runtime.
/// std `RwLock` aman dipakai di handler async selama lock tidak dipegang melewati `.await`.
#[derive(Default)]
pub struct Store {
    items: RwLock<HashMap<u64, Item>>,
}

impl Store {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn seed(&self, n: u64) {
        let mut items = self.items.write().expect("lock poisoned");
        for i in 1..=n {
            items.insert(i, Item { id: i, name: format!("item-{i}"), price: i as f64 * 1.5 });
        }
    }

    /// `.cloned()` menyalin Item keluar supaya read-lock bisa segera dilepas.
    pub fn get(&self, id: u64) -> Option<Item> {
        self.items.read().expect("lock poisoned").get(&id).cloned()
    }

    pub fn upsert(&self, item: Item) -> Item {
        self.items.write().expect("lock poisoned").insert(item.id, item.clone());
        item
    }

    pub fn count(&self) -> usize {
        self.items.read().expect("lock poisoned").len()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::{sync::Arc, thread};

    #[test]
    fn seed_and_get() {
        let s = Store::new();
        s.seed(10);
        let item = s.get(7).expect("item 7 exists");
        assert_eq!(item.name, "item-7");
        assert_eq!(item.price, 10.5);
        assert!(s.get(99).is_none());
    }

    #[test]
    fn concurrent_upsert() {
        // Arc = reference-counted pointer yang boleh dibagi lintas thread.
        let s = Arc::new(Store::new());
        let handles: Vec<_> = (1..=100)
            .map(|id| {
                let s = Arc::clone(&s);
                thread::spawn(move || {
                    s.upsert(Item { id, name: "x".into(), price: 1.0 });
                    s.get(id);
                })
            })
            .collect();
        for h in handles {
            h.join().unwrap();
        }
        assert_eq!(s.count(), 100);
    }
}
