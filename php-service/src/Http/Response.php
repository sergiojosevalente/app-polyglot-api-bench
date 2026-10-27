<?php

declare(strict_types=1);

namespace App\Http;

final readonly class Response
{
    /** @param array<mixed>|object $data */
    public function __construct(
        public int $status,
        public array|object $data,
    ) {
    }

    /** @param array<mixed>|object $data */
    public static function json(array|object $data, int $status = 200): self
    {
        return new self($status, $data);
    }

    public function send(): void
    {
        http_response_code($this->status);
        header('Content-Type: application/json');
        echo json_encode($this->data, JSON_THROW_ON_ERROR | JSON_PRESERVE_ZERO_FRACTION | JSON_UNESCAPED_SLASHES), "\n";
    }
}
