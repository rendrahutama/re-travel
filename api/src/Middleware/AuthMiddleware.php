<?php
declare(strict_types=1);

namespace App\Middleware;

use App\Db\SsoPdo;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface as Handler;
use Slim\Psr7\Factory\ResponseFactory;

class AuthMiddleware implements MiddlewareInterface
{
    public function __construct(private readonly SsoPdo $pdo) {}

    public function process(Request $request, Handler $handler): Response
    {
        $token  = $this->extractToken($request);
        $userId = $this->validateSession($token);

        if ($userId === null) {
            $response = (new ResponseFactory())->createResponse(401);
            $response->getBody()->write(json_encode(['error' => 'unauthorized']));
            return $response->withHeader('Content-Type', 'application/json');
        }

        return $handler->handle($request->withAttribute('userId', $userId));
    }

    private function extractToken(Request $request): string
    {
        // Cookie is the primary SSO mechanism
        $cookies = $request->getCookieParams();
        if (!empty($cookies['auth_token'])) {
            return $cookies['auth_token'];
        }

        // Fallback: Authorization header for local dev / API clients
        $auth = $request->getHeaderLine('Authorization');
        return str_starts_with($auth, 'Bearer ') ? substr($auth, 7) : '';
    }

    private function validateSession(string $token): ?int
    {
        if ($token === '') {
            return null;
        }

        $stmt = $this->pdo->prepare('SELECT user_id, expires_at FROM sessions WHERE token = ?');
        $stmt->execute([$token]);
        $row = $stmt->fetch();

        if (!$row || new \DateTime() > new \DateTime($row['expires_at'])) {
            return null;
        }

        return (int) $row['user_id'];
    }
}
