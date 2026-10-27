<?php

declare(strict_types=1);

namespace App;

use App\Http\HttpError;
use App\Http\Request;
use App\Http\Response;

/**
 * Lapisan HTTP: validasi, panggil business logic, bentuk response.
 * Padanannya: handler.go dan handlers.rs.
 */
final readonly class Handler
{
    private const LANG = 'php';
    private const DEFAULT_FIB_N = 25;

    public function __construct(
        private ItemStore $store,
        private Metrics $metrics,
    ) {
    }

    public function health(Request $r): Response
    {
        return Response::json(['status' => 'ok', 'lang' => self::LANG]);
    }

    public function hello(Request $r): Response
    {
        $name = $r->queryString('name');
        if ($name === null || $name === '') {
            $name = 'World';
        }
        return Response::json(['message' => "Hello, {$name}!", 'lang' => self::LANG]);
    }

    public function cpu(Request $r): Response
    {
        $n = self::DEFAULT_FIB_N;
        if (array_key_exists('n', $r->query) && $r->query['n'] !== '') {
            $raw = $r->queryString('n');
            if ($raw === null || !ctype_digit($raw) || strlen($raw) > 10) {
                throw HttpError::badRequest('n must be a non-negative integer');
            }
            $n = (int) $raw;
        }
        if ($n > Compute::MAX_FIB) {
            throw HttpError::badRequest(sprintf('n must be <= %d', Compute::MAX_FIB));
        }
        // Satu worker FPM = satu request pada satu waktu. Selama fib berjalan, worker ini
        // tidak melayani apa pun; konkurensi PHP = jumlah worker (PHP_WORKERS).
        return Response::json(['n' => $n, 'result' => Compute::fib($n), 'lang' => self::LANG]);
    }

    public function jsonStats(Request $r): Response
    {
        $body = $r->json();
        $numbers = is_array($body) ? ($body['numbers'] ?? []) : null;
        // PHP dinamis: tipe data dari JSON harus divalidasi manual
        // (Go/Rust mendapatkannya dari deklarasi struct).
        if (!is_array($numbers) || !array_is_list($numbers)) {
            throw HttpError::badRequest('invalid JSON: numbers must be an array of numbers');
        }
        foreach ($numbers as $x) {
            if (!is_int($x) && !is_float($x)) {
                throw HttpError::badRequest('invalid JSON: numbers must be an array of numbers');
            }
        }
        $stats = Compute::summarize($numbers)
            ?? throw HttpError::badRequest('numbers must not be empty');

        return Response::json([...$stats->toArray(), 'lang' => self::LANG]);
    }

    /** @param array<string, string> $params */
    public function getItem(Request $r, array $params): Response
    {
        $id = self::parseId($params['id'] ?? '');
        $item = $this->store->get($id)
            ?? throw HttpError::notFound("item {$id} not found");

        return Response::json($item);
    }

    /** @param array<string, string> $params */
    public function putItem(Request $r, array $params): Response
    {
        $id = self::parseId($params['id'] ?? '');
        $body = $r->json();
        $name = is_array($body) ? ($body['name'] ?? null) : null;
        $price = is_array($body) ? ($body['price'] ?? null) : null;

        if (!is_string($name) || trim($name) === '') {
            throw HttpError::badRequest('name must not be empty');
        }
        if ($price === null) {
            throw HttpError::badRequest('price is required');
        }
        if (!is_int($price) && !is_float($price)) {
            throw HttpError::badRequest('invalid JSON: price must be a number');
        }
        if ($price < 0) {
            throw HttpError::badRequest('price must be >= 0');
        }

        return Response::json($this->store->upsert(new Item($id, $name, (float) $price)));
    }

    public function metrics(Request $r): Response
    {
        $ps = ProcStats::read();
        $opcache = function_exists('opcache_get_status') ? opcache_get_status(false) : false;
        $jit = is_array($opcache) && !empty($opcache['jit']['on']);
        $apcu = apcu_sma_info(true);

        return Response::json([
            'lang' => self::LANG,
            'version' => 'PHP ' . PHP_VERSION . ' (FPM' . ($jit ? ' + JIT' : '') . ')',
            'uptime_sec' => $ps->uptimeSec,
            'requests_total' => $this->metrics->requests(),
            'rss_bytes' => $ps->memoryBytes,
            'cpu_seconds' => $ps->cpuSeconds,
            'threads' => $ps->threads,
            'items' => $this->store->count(),
            'runtime' => [
                'fpm_workers' => $ps->fpmWorkers,
                'processes' => $ps->processes,
                'opcache' => is_array($opcache) && !empty($opcache['opcache_enabled']),
                'jit' => $jit,
                'apcu_used_bytes' => is_array($apcu) ? (int) ($apcu['seg_size'] * $apcu['num_seg'] - $apcu['avail_mem']) : 0,
                'memory_source' => $ps->memorySource,
            ],
        ]);
    }

    private static function parseId(string $raw): int
    {
        if (!ctype_digit($raw) || strlen($raw) > 18) {
            throw HttpError::badRequest('id must be a non-negative integer');
        }
        return (int) $raw;
    }
}
