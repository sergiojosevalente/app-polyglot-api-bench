<?php

declare(strict_types=1);

namespace App;

/** Model domain. json_encode otomatis memakai property public (urutan: id, name, price). */
final readonly class Item
{
    public function __construct(
        public int $id,
        public string $name,
        public float $price,
    ) {
    }

    /** Nilai awal item 1..1000, identik dengan Seed() di Go dan seed() di Rust. */
    public static function seeded(int $id): self
    {
        return new self($id, "item-{$id}", $id * 1.5);
    }
}
