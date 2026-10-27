<?php

declare(strict_types=1);

namespace App\Http;

/** Exception yang membawa status HTTP. Padanan enum AppError di rust-service/src/error.rs. */
final class HttpError extends \RuntimeException
{
    public function __construct(public readonly int $status, string $message)
    {
        parent::__construct($message);
    }

    public static function badRequest(string $message): self
    {
        return new self(400, $message);
    }

    public static function notFound(string $message): self
    {
        return new self(404, $message);
    }
}
