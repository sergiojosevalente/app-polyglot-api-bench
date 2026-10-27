<?php

declare(strict_types=1);

namespace App;

/**
 * Store berbasis APCu (shared memory antar proses FPM worker).
 * Padanannya: store.go (map + RWMutex) dan store.rs (RwLock<HashMap>).
 *
 * Kenapa tidak array biasa? Karena shared-nothing: array di memori PHP hilang
 * setelah request selesai, dan setiap worker adalah PROSES terpisah.
 *
 * "Lazy seed": item 1..1000 tidak benar-benar disimpan; kalau belum pernah
 * di-PUT, nilainya dihitung dari Item::seeded(). Ini menghindari race saat
 * seeding karena tidak ada titik "startup" tunggal di PHP-FPM.
 * Tidak perlu mutex: tiap operasi APCu atomic per key.
 */
final class ItemStore
{
    private const SEED_COUNT = 1000;
    private const PREFIX = 'item:';
    private const EXTRA_KEY = 'items:extra';

    public function get(int $id): ?Item
    {
        $row = apcu_fetch(self::PREFIX . $id, $ok);
        if ($ok && is_array($row)) {
            return new Item($id, $row[0], $row[1]);
        }
        return $this->isSeeded($id) ? Item::seeded($id) : null;
    }

    public function upsert(Item $item): Item
    {
        $key = self::PREFIX . $item->id;
        $row = [$item->name, $item->price]; // array lebih murah di-serialize APCu daripada objek
        // apcu_add atomic: true hanya untuk penulis pertama -> hitungan item akurat tanpa lock.
        if (apcu_add($key, $row)) {
            if (!$this->isSeeded($item->id)) {
                apcu_inc(self::EXTRA_KEY);
            }
        } else {
            apcu_store($key, $row);
        }
        return $item;
    }

    public function count(): int
    {
        return self::SEED_COUNT + (int) apcu_fetch(self::EXTRA_KEY);
    }

    private function isSeeded(int $id): bool
    {
        return $id >= 1 && $id <= self::SEED_COUNT;
    }
}
