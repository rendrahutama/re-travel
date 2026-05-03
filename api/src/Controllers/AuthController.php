<?php
declare(strict_types=1);

namespace App\Controllers;

use App\Exception\HttpException;
use App\Exception\ValidationException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

class AuthController extends BaseController
{
    public function register(Request $request, Response $response): Response
    {
        $body     = $this->getBody($request);
        $name     = trim((string) ($body['name'] ?? ''));
        $email    = strtolower(trim((string) ($body['email'] ?? '')));
        $password = trim((string) ($body['password'] ?? ''));

        if ($name === '' || $email === '' || $password === '') {
            throw new ValidationException('name, email, and password are required');
        }
        if (strlen($password) < 8) {
            throw new ValidationException('password must be at least 8 characters');
        }

        $hash = password_hash($password, PASSWORD_BCRYPT);

        try {
            $stmt = $this->pdo->prepare('INSERT INTO users (name, email, password) VALUES (?, ?, ?)');
            $stmt->execute([$name, $email, $hash]);
            $userId = (int) $this->pdo->lastInsertId();
        } catch (\PDOException $e) {
            if (str_contains($e->getMessage(), '1062')) {
                throw new HttpException('email already registered', 409);
            }
            throw $e;
        }

        return $this->json($response, ['id' => $userId, 'name' => $name, 'email' => $email], 201);
    }

    public function login(Request $request, Response $response): Response
    {
        $body  = $this->getBody($request);
        $email = strtolower(trim((string) ($body['email'] ?? '')));
        $pass  = (string) ($body['password'] ?? '');

        $stmt = $this->pdo->prepare('SELECT id, name, password FROM users WHERE email = ?');
        $stmt->execute([$email]);
        $user = $stmt->fetch();

        if (!$user || !password_verify($pass, $user['password'])) {
            throw new HttpException('invalid email or password', 401);
        }

        $token     = bin2hex(random_bytes(32));
        $expiresAt = (new \DateTime())->modify('+30 days')->format('Y-m-d H:i:s');

        $this->pdo->prepare('INSERT INTO sessions (user_id, token, expires_at) VALUES (?, ?, ?)')
             ->execute([(int) $user['id'], $token, $expiresAt]);

        return $this->json($response, [
            'token' => $token,
            'user'  => ['id' => (int) $user['id'], 'name' => $user['name'], 'email' => $email],
        ]);
    }

    public function logout(Request $request, Response $response): Response
    {
        $auth = $request->getHeaderLine('Authorization');
        if (str_starts_with($auth, 'Bearer ')) {
            $token = substr($auth, 7);
            $this->pdo->prepare('DELETE FROM sessions WHERE token = ?')->execute([$token]);
        }

        return $this->json($response, ['ok' => true]);
    }

    public function me(Request $request, Response $response): Response
    {
        $userId = (int) $request->getAttribute('userId');

        $stmt = $this->pdo->prepare('SELECT id, name, email FROM users WHERE id = ?');
        $stmt->execute([$userId]);
        $user = $stmt->fetch();

        if (!$user) {
            throw new \App\Exception\NotFoundException('user');
        }

        return $this->json($response, [
            'id'    => (int) $user['id'],
            'name'  => $user['name'],
            'email' => $user['email'],
        ]);
    }
}
