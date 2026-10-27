<?php

declare(strict_types=1);

namespace App;

use App\Http\HttpError;
use App\Http\Request;
use App\Http\Response;
use App\Http\Router;

/**
 * Wiring: dependency -> router -> dispatch. Padanan main.go / main.rs,
 * tapi dijalankan ulang di SETIAP request (shared-nothing).
 */
final class App
{
    public static function handle(): void
    {
        $metrics = new Metrics();
        $metrics->countRequest();

        $handler = new Handler(new ItemStore(), $metrics);

        $router = new Router();
        $router->add('GET', '/health', $handler->health(...));
        $router->add('GET', '/api/hello', $handler->hello(...));
        $router->add('POST', '/api/json', $handler->jsonStats(...));
        $router->add('GET', '/api/cpu', $handler->cpu(...));
        $router->add('GET', '/api/items/{id}', $handler->getItem(...));
        $router->add('PUT', '/api/items/{id}', $handler->putItem(...));
        $router->add('GET', '/metrics', $handler->metrics(...));

        // Exception = mekanisme error PHP. Satu try/catch terpusat di sini
        // setara dengan IntoResponse for AppError di Rust.
        try {
            $response = $router->dispatch(Request::fromGlobals());
        } catch (HttpError $e) {
            $response = Response::json(['error' => $e->getMessage()], $e->status);
        }
        $response->send();
    }
}
