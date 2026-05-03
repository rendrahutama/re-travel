<?php
declare(strict_types=1);

namespace App\Controllers;

use App\Exception\ForbiddenException;
use App\Exception\NotFoundException;
use App\Exception\ValidationException;
use PDO;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

abstract class BaseController
{
    protected const ACTIVITY_TYPES = [
        'Attraction', 'Beach', 'Bus', 'Car', 'Culinary',
        'Culture', 'Cycling', 'Event', 'Explore', 'Ferry',
        'Flight', 'Hiking', 'Motorscooter', 'Nature', 'Other',
        'Shopping', 'Spa', 'Sport', 'Stay', 'Taxi', 'Train',
    ];

    public function __construct(protected readonly PDO $pdo) {}

    protected function json(Response $response, mixed $data, int $status = 200): Response
    {
        $response->getBody()->write(json_encode($data));
        return $response
            ->withStatus($status)
            ->withHeader('Content-Type', 'application/json');
    }

    protected function resolveItineraryId(string $segment): int
    {
        if (ctype_digit($segment)) {
            return (int) $segment;
        }

        $stmt = $this->pdo->prepare('SELECT id FROM itineraries WHERE slug = ?');
        $stmt->execute([$segment]);
        $row = $stmt->fetch();

        if (!$row) {
            throw new NotFoundException('itinerary');
        }

        return (int) $row['id'];
    }

    protected function getItinerary(int $id): array
    {
        $stmt = $this->pdo->prepare('
            SELECT i.id, i.user_id, i.name,
                   COALESCE(i.description, \'\') AS description,
                   i.start_date, i.end_date,
                   COALESCE(i.currency, \'IDR\') AS currency,
                   COALESCE(i.estimated_cost, 0) AS estimated_cost,
                   i.cover_image_url, i.created_at, i.is_public,
                   COALESCE((SELECT name FROM users WHERE id = i.user_id), \'Unknown\') AS owner_name,
                   COALESCE(i.slug, \'\') AS slug
            FROM itineraries i
            WHERE i.id = ?
        ');
        $stmt->execute([$id]);
        $row = $stmt->fetch();

        if (!$row) {
            throw new NotFoundException('itinerary');
        }

        return [
            'id'            => (string) $row['id'],
            'ownerId'       => (int) $row['user_id'],
            'ownerName'     => $row['owner_name'],
            'slug'          => $row['slug'],
            'name'          => $row['name'],
            'description'   => $row['description'],
            'startDate'     => substr($row['start_date'], 0, 10),
            'endDate'       => substr($row['end_date'], 0, 10),
            'currency'      => $row['currency'],
            'estimatedCost' => (float) $row['estimated_cost'],
            'image'         => $row['cover_image_url'] ?: null,
            'isPublic'      => (bool) $row['is_public'],
            'createdAt'     => $row['created_at'],
            'activities'    => $this->listActivities($id),
        ];
    }

    protected function listActivities(int $itineraryId): array
    {
        $stmt = $this->pdo->prepare('
            SELECT id, activity_date, start_time, activity_type,
                   COALESCE(identifier, \'\') AS identifier,
                   COALESCE(location_name, \'\') AS location_name,
                   COALESCE(location_address, \'\') AS location_address,
                   latitude, longitude,
                   COALESCE(cost, 0) AS cost,
                   ticket_status,
                   COALESCE(details, \'\') AS details,
                   COALESCE(sort_order, 0) AS sort_order
            FROM activities
            WHERE itinerary_id = ?
            ORDER BY sort_order ASC, activity_date ASC, start_time ASC, id ASC
        ');
        $stmt->execute([$itineraryId]);

        return array_map(fn($r) => $this->formatActivity($r), $stmt->fetchAll());
    }

    protected function getActivity(int $itineraryId, int $activityId): array
    {
        $stmt = $this->pdo->prepare('
            SELECT id, activity_date, start_time, activity_type,
                   COALESCE(identifier, \'\') AS identifier,
                   COALESCE(location_name, \'\') AS location_name,
                   COALESCE(location_address, \'\') AS location_address,
                   latitude, longitude,
                   COALESCE(cost, 0) AS cost,
                   ticket_status,
                   COALESCE(details, \'\') AS details,
                   COALESCE(sort_order, 0) AS sort_order
            FROM activities
            WHERE itinerary_id = ? AND id = ?
        ');
        $stmt->execute([$itineraryId, $activityId]);
        $row = $stmt->fetch();

        if (!$row) {
            throw new NotFoundException('activity');
        }

        return $this->formatActivity($row);
    }

    private function formatActivity(array $row): array
    {
        $date = substr($row['activity_date'], 0, 10);
        $time = substr($row['start_time'], 0, 5);

        return [
            'id'             => (string) $row['id'],
            'datetime'       => "{$date}T{$time}",
            'type'           => $row['activity_type'],
            'identification' => $row['identifier'],
            'location'       => [
                'name'    => $row['location_name'],
                'address' => $row['location_address'],
                'lat'     => $row['latitude'] !== null ? (float) $row['latitude'] : null,
                'lng'     => $row['longitude'] !== null ? (float) $row['longitude'] : null,
            ],
            'cost'         => (float) $row['cost'],
            'ticketStatus' => $row['ticket_status'] ?: null,
            'details'      => $row['details'],
            'sortOrder'    => (int) $row['sort_order'],
        ];
    }

    protected function syncDerivedFields(int $itineraryId): void
    {
        $this->pdo->prepare('
            UPDATE itineraries
            SET end_date = COALESCE(
                    (SELECT CASE
                        WHEN MAX(a.activity_date) IS NULL OR MAX(a.activity_date) < itineraries.start_date
                            THEN itineraries.start_date
                        ELSE MAX(a.activity_date)
                     END
                     FROM activities a
                     WHERE a.itinerary_id = itineraries.id),
                    start_date
                ),
                estimated_cost = COALESCE(
                    (SELECT SUM(a.cost) FROM activities a WHERE a.itinerary_id = itineraries.id),
                    0
                ),
                updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        ')->execute([$itineraryId]);
    }

    protected function checkOwnership(int $itineraryId, int $userId): void
    {
        $stmt = $this->pdo->prepare('SELECT user_id FROM itineraries WHERE id = ?');
        $stmt->execute([$itineraryId]);
        $row = $stmt->fetch();

        if (!$row) {
            throw new NotFoundException('itinerary');
        }
        if ((int) $row['user_id'] !== $userId) {
            throw new ForbiddenException();
        }
    }

    protected function generateUniqueSlug(string $base): string
    {
        if ($base === '') {
            $base = 'itinerary';
        }

        $slug = $base;
        $i    = 2;
        while (true) {
            $stmt = $this->pdo->prepare('SELECT COUNT(*) FROM itineraries WHERE slug = ?');
            $stmt->execute([$slug]);
            if ((int) $stmt->fetchColumn() === 0) {
                return $slug;
            }
            $slug = "{$base}-{$i}";
            $i++;
        }
    }

    protected static function toSlug(string $name): string
    {
        $s           = strtolower(trim($name));
        $result      = '';
        $prevHyphen  = true;

        foreach (mb_str_split($s) as $char) {
            if (preg_match('/[a-z0-9]/', $char)) {
                $result    .= $char;
                $prevHyphen = false;
            } elseif (!$prevHyphen) {
                $result    .= '-';
                $prevHyphen = true;
            }
        }

        return rtrim($result, '-');
    }

    protected function parseDatetime(string $value): array
    {
        $dt = \DateTime::createFromFormat('Y-m-d\TH:i', $value);
        if (!$dt) {
            throw new ValidationException('datetime must use YYYY-MM-DDTHH:MM');
        }
        return [$dt->format('Y-m-d'), $dt->format('H:i:s')];
    }

    protected function getBody(Request $request): array
    {
        $parsed = $request->getParsedBody();
        if (is_array($parsed)) {
            return $parsed;
        }

        $raw = (string) $request->getBody();
        if ($raw !== '') {
            $decoded = json_decode($raw, true);
            if (is_array($decoded)) {
                return $decoded;
            }
        }

        return [];
    }
}
