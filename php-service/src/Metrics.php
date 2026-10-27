<?php

declare(strict_types=1);

namespace App;

/**
 * Counter request global. Di Go/Rust cukup variabel atomic di memori proses;
 * di PHP harus di shared memory (APCu) karena tiap request & worker terpisah.
 */
final class Metrics
{
    private const REQUESTS_KEY = 'metrics:requests';

    public function countRequest(): void
    {
        apcu_inc(self::REQUESTS_KEY);
    }

    public function requests(): int
    {
        return (int) apcu_fetch(self::REQUESTS_KEY);
    }
}
