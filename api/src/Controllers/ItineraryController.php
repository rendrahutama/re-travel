<?php
declare(strict_types=1);

namespace App\Controllers;

use App\Exception\ForbiddenException;
use App\Exception\HttpException;
use App\Exception\ValidationException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

class ItineraryController extends BaseController
{
    public function index(Request $request, Response $response): Response
    {
        $userEmail = $request->getAttribute('userEmail');

        if ($userEmail !== null) {
            $stmt = $this->pdo->prepare('
                SELECT id FROM itineraries
                WHERE user_email = ?
                ORDER BY start_date ASC, id ASC
            ');
            $stmt->execute([$userEmail]);
        } else {
            $stmt = $this->pdo->query('
                SELECT id FROM itineraries
                WHERE is_public = 1
                ORDER BY start_date ASC, id ASC
            ');
        }

        $items = [];
        foreach ($stmt->fetchAll() as $row) {
            $items[] = $this->getItinerary((int) $row['id']);
        }

        return $this->json($response, $items);
    }

    public function create(Request $request, Response $response): Response
    {
        $userEmail = $request->getAttribute('userEmail');
        if ($userEmail === null) {
            throw new HttpException('unauthorized', 401);
        }

        $body    = $this->getBody($request);
        $payload = $this->normalizePayload($body);
        $this->validatePayload($payload);

        $slug = $this->generateUniqueSlug($this->toSlug($payload['name']));

        $stmt = $this->pdo->prepare('
            INSERT INTO itineraries
                (user_email, name, description, start_date, end_date, currency, cover_image_url, estimated_cost, is_public, slug)
            VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
        ');
        $stmt->execute([
            $userEmail,
            $payload['name'],
            $payload['description'],
            $payload['startDate'],
            $payload['startDate'],
            $payload['currency'],
            $payload['image'],
            $payload['isPublic'] ? 1 : 0,
            $slug,
        ]);

        $id = (int) $this->pdo->lastInsertId();
        return $this->json($response, $this->getItinerary($id), 201);
    }

    public function show(Request $request, Response $response, array $args): Response
    {
        $id     = $this->resolveItineraryId($args['id']);
        $item   = $this->getItinerary($id);
        $userEmail = $request->getAttribute('userEmail');

        if (!$item['isPublic'] && ($userEmail === null || $item['ownerEmail'] !== $userEmail)) {
            throw new ForbiddenException();
        }

        return $this->json($response, $item);
    }

    public function update(Request $request, Response $response, array $args): Response
    {
        $userEmail = $request->getAttribute('userEmail');
        if ($userEmail === null) {
            throw new HttpException('unauthorized', 401);
        }

        $id = $this->resolveItineraryId($args['id']);
        $this->checkOwnership($id, $userEmail);

        $current = $this->getItinerary($id);
        $body    = $this->getBody($request);
        $payload = $this->normalizePayload($this->mergePayload($current, $body));
        $this->validatePayload($payload);

        if ($payload['image'] !== $current['image']) {
            $this->deleteUploadedImage($current['image']);
        }

        $this->pdo->prepare('
            UPDATE itineraries
            SET name = ?, description = ?, start_date = ?, end_date = ?, currency = ?,
                cover_image_url = ?, estimated_cost = 0, is_public = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        ')->execute([
            $payload['name'],
            $payload['description'],
            $payload['startDate'],
            $payload['startDate'],
            $payload['currency'],
            $payload['image'],
            $payload['isPublic'] ? 1 : 0,
            $id,
        ]);

        $this->syncDerivedFields($id);
        return $this->json($response, $this->getItinerary($id));
    }

    public function destroy(Request $request, Response $response, array $args): Response
    {
        $userEmail = $request->getAttribute('userEmail');
        if ($userEmail === null) {
            throw new HttpException('unauthorized', 401);
        }

        $id = $this->resolveItineraryId($args['id']);
        $this->checkOwnership($id, $userEmail);

        $row = $this->pdo->prepare('SELECT cover_image_url FROM itineraries WHERE id = ?');
        $row->execute([$id]);
        $imageUrl = $row->fetchColumn();

        $this->pdo->prepare('DELETE FROM itineraries WHERE id = ?')->execute([$id]);

        $this->deleteUploadedImage($imageUrl ?: null);

        return $this->json($response, ['deleted' => true]);
    }

    private function normalizePayload(array $body): array
    {
        $currency = strtoupper(trim((string) ($body['currency'] ?? '')));
        return [
            'name'        => trim((string) ($body['name'] ?? '')),
            'description' => trim((string) ($body['description'] ?? '')),
            'startDate'   => trim((string) ($body['startDate'] ?? '')),
            'currency'    => $currency !== '' ? $currency : 'IDR',
            'image'       => isset($body['image']) && trim((string) $body['image']) !== ''
                              ? trim((string) $body['image'])
                              : null,
            'isPublic'    => (bool) ($body['isPublic'] ?? false),
        ];
    }

    private function mergePayload(array $current, array $body): array
    {
        return [
            'name'        => ($body['name'] ?? '') !== '' ? $body['name'] : $current['name'],
            'description' => ($body['description'] ?? '') !== '' ? $body['description'] : $current['description'],
            'startDate'   => ($body['startDate'] ?? '') !== '' ? $body['startDate'] : $current['startDate'],
            'currency'    => ($body['currency'] ?? '') !== '' ? $body['currency'] : $current['currency'],
            'image'       => array_key_exists('image', $body) ? $body['image'] : $current['image'],
            'isPublic'    => array_key_exists('isPublic', $body) ? $body['isPublic'] : $current['isPublic'],
        ];
    }

    private function deleteUploadedImage(?string $url): void
    {
        if ($url === null || !str_contains($url, '/uploads/')) {
            return;
        }

        $filename = basename(parse_url($url, PHP_URL_PATH) ?? '');
        if ($filename === '') {
            return;
        }

        $path = __DIR__ . '/../../public/uploads/' . $filename;
        if (is_file($path)) {
            @unlink($path);
        }
    }

    private function validatePayload(array $payload): void
    {
        if ($payload['name'] === '') {
            throw new ValidationException('name is required');
        }
        if ($payload['startDate'] === '') {
            throw new ValidationException('startDate is required');
        }
        if (!\DateTime::createFromFormat('Y-m-d', $payload['startDate'])) {
            throw new ValidationException('startDate must use YYYY-MM-DD');
        }
    }
}
