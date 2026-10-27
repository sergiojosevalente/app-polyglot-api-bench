<?php

declare(strict_types=1);

namespace App\Http;

/**
 * Router minimal: method + pattern dengan parameter {name}.
 * Go punya ServeMux di stdlib, Rust memakai axum; di PHP produksi biasanya
 * memakai framework (Laravel/Symfony) atau library (FastRoute). Di sini ditulis
 * sendiri supaya 0 dependency runtime dan mudah dibaca.
 */
final class Router
{
    /** @var array<string, array<string, \Closure>> pattern => method => handler */
    private array $routes = [];

    public function add(string $method, string $pattern, \Closure $handler): void
    {
        $this->routes[$pattern][$method] = $handler;
    }

    public function dispatch(Request $request): Response
    {
        foreach ($this->routes as $pattern => $methods) {
            $regex = '#^' . preg_replace('#\{(\w+)\}#', '(?P<$1>[^/]+)', $pattern) . '$#';
            if (preg_match($regex, $request->path, $m) !== 1) {
                continue;
            }
            $handler = $methods[$request->method] ?? null;
            if ($handler === null) {
                return Response::json(['error' => 'method not allowed'], 405);
            }
            $params = array_filter($m, 'is_string', ARRAY_FILTER_USE_KEY);
            return $handler($request, $params);
        }
        return Response::json(['error' => 'not found'], 404);
    }
}
