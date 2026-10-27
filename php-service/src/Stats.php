<?php

declare(strict_types=1);

namespace App;

/** Value object immutable. Constructor promotion (PHP 8) = deklarasi + assign sekaligus. */
final readonly class Stats
{
    public function __construct(
        public int $count,
        public float $sum,
        public float $min,
        public float $max,
        public float $mean,
    ) {
    }

    /** @return array<string, int|float> */
    public function toArray(): array
    {
        return get_object_vars($this);
    }
}
