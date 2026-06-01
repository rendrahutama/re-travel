<?php
declare(strict_types=1);

namespace App\Middleware;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface as Handler;
use Slim\Psr7\Factory\ResponseFactory;

class CorsMiddleware implements MiddlewareInterface
{
    public function process(Request $request, Handler $handler): Response
    {
        if ($request->getMethod() === 'OPTIONS') {
            return $this->addHeaders((new ResponseFactory())->createResponse(204), $request);
        }

        return $this->addHeaders($handler->handle($request), $request);
    }

    private function addHeaders(Response $response, Request $request): Response
    {
        $origin  = $request->getHeaderLine('Origin');
        $allowed = array_filter(array_map('trim', explode(',', $_ENV['ALLOWED_ORIGINS'] ?? '')));

        if (in_array($origin, $allowed, true)) {
            $response = $response->withHeader('Access-Control-Allow-Origin', $origin);
        } elseif (!empty($allowed)) {
            $response = $response->withHeader('Access-Control-Allow-Origin', $allowed[0]);
        } else {
            // Fallback for local dev when ALLOWED_ORIGINS is not set
            $response = $response->withHeader('Access-Control-Allow-Origin', $origin ?: '*');
        }

        return $response
            ->withHeader('Access-Control-Allow-Credentials', 'true')
            ->withHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, PATCH, DELETE, OPTIONS')
            ->withHeader('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    }
}
