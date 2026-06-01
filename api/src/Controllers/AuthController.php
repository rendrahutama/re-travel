<?php
declare(strict_types=1);

namespace App\Controllers;

use App\Db\SsoPdo;
use App\Exception\NotFoundException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

class AuthController
{
    public function __construct(private readonly SsoPdo $pdo) {}

    public function me(Request $request, Response $response): Response
    {
        $userId = (int) $request->getAttribute('userId');

        $stmt = $this->pdo->prepare('SELECT id, name, email FROM users WHERE id = ?');
        $stmt->execute([$userId]);
        $user = $stmt->fetch();

        if (!$user) {
            throw new NotFoundException('user');
        }

        $response->getBody()->write(json_encode([
            'id'    => (int) $user['id'],
            'name'  => $user['name'],
            'email' => $user['email'],
        ]));
        return $response->withHeader('Content-Type', 'application/json');
    }

    public function logout(Request $request, Response $response): Response
    {
        $cookies = $request->getCookieParams();
        $token   = $cookies['auth_token'] ?? '';

        if ($token !== '') {
            $this->pdo->prepare('DELETE FROM sessions WHERE token = ?')->execute([$token]);
        }

        // Clear the SSO cookie from this subdomain as well
        $domain   = $_ENV['COOKIE_DOMAIN'] ?? '';
        $isSecure = $domain !== '' && $domain !== 'localhost';

        $parts = [
            'auth_token=',
            'Path=/',
            'Expires=' . gmdate('D, d M Y H:i:s T', time() - 3600),
            'HttpOnly',
            'SameSite=Lax',
        ];
        if ($domain !== '') {
            $parts[] = 'Domain=' . $domain;
        }
        if ($isSecure) {
            $parts[] = 'Secure';
        }

        $response->getBody()->write(json_encode(['ok' => true]));
        return $response
            ->withHeader('Content-Type', 'application/json')
            ->withAddedHeader('Set-Cookie', implode('; ', $parts));
    }
}
