<?php
declare(strict_types=1);

namespace App\Controllers;

use App\Exception\ForbiddenException;
use App\Exception\HttpException;
use App\Exception\NotFoundException;
use App\Exception\ValidationException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

class ActivityController extends BaseController
{
    public function index(Request $request, Response $response, array $args): Response
    {
        $id     = $this->resolveItineraryId($args['id']);
        $item   = $this->getItinerary($id);
        $userId = $request->getAttribute('userId');

        if (!$item['isPublic'] && ($userId === null || $item['ownerId'] !== (int) $userId)) {
            throw new ForbiddenException();
        }

        return $this->json($response, $this->listActivities($id));
    }

    public function create(Request $request, Response $response, array $args): Response
    {
        $userId = $request->getAttribute('userId');
        if ($userId === null) {
            throw new HttpException('unauthorized', 401);
        }

        $id = $this->resolveItineraryId($args['id']);
        $this->checkOwnership($id, (int) $userId);

        $body      = $this->getBody($request);
        $payload   = $this->normalizePayload($body);
        $this->validatePayload($payload);

        [$datePart, $timePart] = $this->parseDatetime($payload['datetime']);
        $sortOrder = $payload['sortOrder'] ?? $this->nextSortOrder($id);

        $stmt = $this->pdo->prepare('
            INSERT INTO activities
                (itinerary_id, activity_type, identifier, location_name, location_address,
                 latitude, longitude, activity_date, start_time, cost, ticket_status, details, sort_order)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ');
        $stmt->execute([
            $id,
            $payload['type'],
            $payload['identification'],
            $payload['location']['name'],
            $payload['location']['address'],
            $payload['location']['lat'],
            $payload['location']['lng'],
            $datePart,
            $timePart,
            $payload['cost'],
            $payload['ticketStatus'],
            $payload['details'],
            $sortOrder,
        ]);

        $actId = (int) $this->pdo->lastInsertId();
        $this->syncDerivedFields($id);
        return $this->json($response, $this->getActivity($id, $actId), 201);
    }

    public function show(Request $request, Response $response, array $args): Response
    {
        $id    = $this->resolveItineraryId($args['id']);
        $actId = (int) $args['actId'];
        return $this->json($response, $this->getActivity($id, $actId));
    }

    public function update(Request $request, Response $response, array $args): Response
    {
        $userId = $request->getAttribute('userId');
        if ($userId === null) {
            throw new HttpException('unauthorized', 401);
        }

        $id    = $this->resolveItineraryId($args['id']);
        $actId = (int) $args['actId'];
        $this->checkOwnership($id, (int) $userId);

        $current = $this->getActivity($id, $actId);
        $body    = $this->getBody($request);
        $payload = $this->normalizePayload($this->mergePayload($current, $body));
        $this->validatePayload($payload);

        [$datePart, $timePart] = $this->parseDatetime($payload['datetime']);
        $sortOrder = $payload['sortOrder'] ?? $current['sortOrder'];

        $this->pdo->prepare('
            UPDATE activities
            SET activity_type = ?, identifier = ?, location_name = ?, location_address = ?,
                latitude = ?, longitude = ?, activity_date = ?, start_time = ?, cost = ?,
                ticket_status = ?, details = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
            WHERE itinerary_id = ? AND id = ?
        ')->execute([
            $payload['type'],
            $payload['identification'],
            $payload['location']['name'],
            $payload['location']['address'],
            $payload['location']['lat'],
            $payload['location']['lng'],
            $datePart,
            $timePart,
            $payload['cost'],
            $payload['ticketStatus'],
            $payload['details'],
            $sortOrder,
            $id,
            $actId,
        ]);

        $this->syncDerivedFields($id);
        return $this->json($response, $this->getActivity($id, $actId));
    }

    public function destroy(Request $request, Response $response, array $args): Response
    {
        $userId = $request->getAttribute('userId');
        if ($userId === null) {
            throw new HttpException('unauthorized', 401);
        }

        $id    = $this->resolveItineraryId($args['id']);
        $actId = (int) $args['actId'];
        $this->checkOwnership($id, (int) $userId);

        $this->pdo->prepare('DELETE FROM activities WHERE itinerary_id = ? AND id = ?')
             ->execute([$id, $actId]);
        $this->syncDerivedFields($id);

        return $this->json($response, ['deleted' => true]);
    }

    public function move(Request $request, Response $response, array $args): Response
    {
        $userId = $request->getAttribute('userId');
        if ($userId === null) {
            throw new HttpException('unauthorized', 401);
        }

        $id        = $this->resolveItineraryId($args['id']);
        $actId     = (int) $args['actId'];
        $direction = (string) ($this->getBody($request)['direction'] ?? '');

        $this->checkOwnership($id, (int) $userId);

        if (!in_array($direction, ['up', 'down'], true)) {
            throw new ValidationException("direction must be either 'up' or 'down'");
        }

        $items = $this->listActivities($id);
        $index = -1;
        foreach ($items as $i => $item) {
            if ((int) $item['id'] === $actId) {
                $index = $i;
                break;
            }
        }

        if ($index === -1) {
            throw new NotFoundException('activity');
        }

        $targetIndex = $direction === 'up' ? $index - 1 : $index + 1;
        if ($targetIndex < 0 || $targetIndex >= count($items)) {
            return $this->json($response, $this->getActivity($id, $actId));
        }

        $currentSort = $items[$index]['sortOrder'];
        $swapSort    = $items[$targetIndex]['sortOrder'];
        $swapId      = (int) $items[$targetIndex]['id'];

        $this->pdo->beginTransaction();
        try {
            $this->pdo->prepare('UPDATE activities SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND itinerary_id = ?')
                 ->execute([$swapSort, $actId, $id]);
            $this->pdo->prepare('UPDATE activities SET sort_order = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND itinerary_id = ?')
                 ->execute([$currentSort, $swapId, $id]);
            $this->pdo->commit();
        } catch (\Throwable $e) {
            $this->pdo->rollBack();
            throw $e;
        }

        return $this->json($response, $this->getActivity($id, $actId));
    }

    private function normalizePayload(array $body): array
    {
        $type = trim((string) ($body['type'] ?? ''));
        $loc  = $body['location'] ?? [];

        return [
            'datetime'       => trim((string) ($body['datetime'] ?? '')),
            'type'           => $type !== '' ? $type : 'Other',
            'identification' => trim((string) ($body['identification'] ?? '')),
            'location'       => [
                'name'    => trim((string) ($loc['name'] ?? '')),
                'address' => trim((string) ($loc['address'] ?? '')),
                'lat'     => isset($loc['lat']) && $loc['lat'] !== null ? (float) $loc['lat'] : null,
                'lng'     => isset($loc['lng']) && $loc['lng'] !== null ? (float) $loc['lng'] : null,
            ],
            'cost'         => (float) ($body['cost'] ?? 0),
            'ticketStatus' => isset($body['ticketStatus']) && trim((string) $body['ticketStatus']) !== ''
                               ? trim((string) $body['ticketStatus'])
                               : null,
            'details'      => trim((string) ($body['details'] ?? '')),
            'sortOrder'    => isset($body['sortOrder']) ? (int) $body['sortOrder'] : null,
        ];
    }

    private function mergePayload(array $current, array $body): array
    {
        return [
            'datetime'       => ($body['datetime'] ?? '') !== '' ? $body['datetime'] : $current['datetime'],
            'type'           => ($body['type'] ?? '') !== '' ? $body['type'] : $current['type'],
            'identification' => ($body['identification'] ?? '') !== '' ? $body['identification'] : $current['identification'],
            'location'       => !empty($body['location']) ? $body['location'] : $current['location'],
            'cost'           => array_key_exists('cost', $body) ? $body['cost'] : $current['cost'],
            'ticketStatus'   => array_key_exists('ticketStatus', $body) ? $body['ticketStatus'] : $current['ticketStatus'],
            'details'        => ($body['details'] ?? '') !== '' ? $body['details'] : $current['details'],
            'sortOrder'      => array_key_exists('sortOrder', $body) ? $body['sortOrder'] : $current['sortOrder'],
        ];
    }

    private function validatePayload(array $payload): void
    {
        if ($payload['datetime'] === '') {
            throw new ValidationException('datetime is required');
        }
        $this->parseDatetime($payload['datetime']);
        if ($payload['type'] === '') {
            throw new ValidationException('type is required');
        }
        if (!in_array($payload['type'], self::ACTIVITY_TYPES, true)) {
            throw new ValidationException('type is not supported');
        }
    }

    private function nextSortOrder(int $itineraryId): int
    {
        $stmt = $this->pdo->prepare('SELECT COALESCE(MAX(sort_order), -1) + 1 FROM activities WHERE itinerary_id = ?');
        $stmt->execute([$itineraryId]);
        return (int) $stmt->fetchColumn();
    }
}
