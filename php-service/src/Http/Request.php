<?php

declare(strict_types=1);

namespace App\Http;

/** Request immutable (readonly class, PHP 8.2+) dibangun dari superglobal. */
final readonly class Request
{
    /** @param array<string, mixed> $query */
    public function __construct(
        public string $method,
        public string $path,
        public array $query,
        public string $body,
    ) {
    }

    public static function fromGlobals(): self
    {
        $path = parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH);

        return new self(
            $_SERVER['REQUEST_METHOD'] ?? 'GET',
            is_string($path) && $path !== '' ? $path : '/',
            $_GET,
            (string) file_get_contents('php://input'),
        );
    }

    /** Ambil query string sebagai string; array (?n[]=1) dianggap tidak valid -> null. */
    public function queryString(string $key): ?string
    {
        $v = $this->query[$key] ?? null;
        return is_string($v) ? $v : null;
    }

    /** Decode body JSON menjadi array asosiatif; JSON rusak -> 400. */
    public function json(): mixed
    {
        try {
            return json_decode($this->body, true, 64, JSON_THROW_ON_ERROR);
        } catch (\JsonException $e) {
            throw HttpError::badRequest('invalid JSON: ' . $e->getMessage());
        }
    }
}
