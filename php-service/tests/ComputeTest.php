<?php

declare(strict_types=1);

namespace App\Tests;

use App\Compute;
use App\Stats;
use PHPUnit\Framework\Attributes\DataProvider;
use PHPUnit\Framework\TestCase;

final class ComputeTest extends TestCase
{
    /** @return list<array{int, int}> */
    public static function fibCases(): array
    {
        return [[0, 0], [1, 1], [2, 1], [10, 55], [25, 75025]];
    }

    // Data provider = padanan table-driven test di Go.
    #[DataProvider('fibCases')]
    public function testFib(int $n, int $want): void
    {
        self::assertSame($want, Compute::fib($n));
    }

    public function testSummarize(): void
    {
        self::assertEquals(new Stats(5, 8.5, -2.0, 4.5, 1.7), Compute::summarize([1, 2, 3, 4.5, -2]));
    }

    public function testSummarizeEmpty(): void
    {
        self::assertNull(Compute::summarize([]));
    }
}
