<?php
declare(strict_types=1);

namespace App\Middleware;

use App\Db\SsoPdo;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface as Handler;

class OptionalAuthMiddleware implements MiddlewareInterface
{
    public function __construct(private readonly SsoPdo $pdo) {}

    public function process(Request $request, Handler $handler): Response
    {
        $token = $this->extractToken($request);

        if ($token !== '') {
            $stmt = $this->pdo->prepare('
                SELECT u.email, s.expires_at
                FROM sessions s
                JOIN users u ON u.id = s.user_id
                WHERE s.token = ?
            ');
            $stmt->execute([$token]);
            $row = $stmt->fetch();

            if ($row && new \DateTime() <= new \DateTime($row['expires_at'])) {
                $request = $request->withAttribute('userEmail', $row['email']);
            }
        }

        return $handler->handle($request);
    }

    private function extractToken(Request $request): string
    {
        $cookies = $request->getCookieParams();
        if (!empty($cookies['auth_token'])) {
            return $cookies['auth_token'];
        }

        $auth = $request->getHeaderLine('Authorization');
        return str_starts_with($auth, 'Bearer ') ? substr($auth, 7) : '';
    }
}
