<?php
declare(strict_types=1);

namespace App\Middleware;

use PDO;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface as Handler;

class OptionalAuthMiddleware implements MiddlewareInterface
{
    public function __construct(private readonly PDO $pdo) {}

    public function process(Request $request, Handler $handler): Response
    {
        $auth = $request->getHeaderLine('Authorization');
        if (str_starts_with($auth, 'Bearer ')) {
            $token = substr($auth, 7);
            $stmt  = $this->pdo->prepare('SELECT user_id, expires_at FROM sessions WHERE token = ?');
            $stmt->execute([$token]);
            $row = $stmt->fetch();

            if ($row && new \DateTime() <= new \DateTime($row['expires_at'])) {
                $request = $request->withAttribute('userId', (int) $row['user_id']);
            }
        }

        return $handler->handle($request);
    }
}
