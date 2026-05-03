<?php
declare(strict_types=1);

namespace App\Controllers;

use App\Exception\ValidationException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

class ImportController extends BaseController
{
    public function fromLocalStorage(Request $request, Response $response): Response
    {
        $userId = (int) $request->getAttribute('userId');
        $parsed = $request->getParsedBody();

        if (empty($parsed)) {
            throw new ValidationException('import payload is empty');
        }

        if (array_is_list($parsed)) {
            $itineraries     = $parsed;
            $replaceExisting = true;
        } else {
            $itineraries     = $parsed['itineraries'] ?? [];
            $replaceExisting = (bool) ($parsed['replaceExisting'] ?? false);
        }

        if (empty($itineraries)) {
            throw new ValidationException('itineraries is required and must not be empty');
        }

        $this->pdo->beginTransaction();
        try {
            if ($replaceExisting) {
                $this->pdo->prepare('DELETE FROM itineraries WHERE user_id = ?')->execute([$userId]);
            }

            $importedIds = [];
            $batchSlugs  = [];

            foreach ($itineraries as $source) {
                $payload = $this->normalizeItineraryPayload($source);
                $this->validateItineraryPayload($payload);

                $createdAt = $this->normalizeCreatedAt($source['createdAt'] ?? '');

                $base = $this->toSlug($payload['name']);
                if ($base === '') {
                    $base = 'itinerary';
                }
                $slug = $base;
                $i    = 2;
                while (true) {
                    if (!isset($batchSlugs[$slug])) {
                        $stmt = $this->pdo->prepare('SELECT COUNT(*) FROM itineraries WHERE slug = ?');
                        $stmt->execute([$slug]);
                        if ((int) $stmt->fetchColumn() === 0) {
                            break;
                        }
                    }
                    $slug = "{$base}-{$i}";
                    $i++;
                }
                $batchSlugs[$slug] = true;

                $stmt = $this->pdo->prepare('
                    INSERT INTO itineraries
                        (user_id, name, description, start_date, end_date, currency,
                         cover_image_url, estimated_cost, is_public, slug, created_at, updated_at)
                    VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?)
                ');
                $stmt->execute([
                    $userId,
                    $payload['name'],
                    $payload['description'],
                    $payload['startDate'],
                    $payload['endDate'] !== '' ? $payload['endDate'] : $payload['startDate'],
                    $payload['currency'],
                    $payload['image'],
                    $payload['estimatedCost'],
                    $slug,
                    $createdAt,
                    $createdAt,
                ]);

                $itineraryId   = (int) $this->pdo->lastInsertId();
                $importedIds[] = $itineraryId;

                foreach (($source['activities'] ?? []) as $sourceActivity) {
                    $ap = $this->normalizeActivityPayload($sourceActivity);
                    $this->validateActivityPayload($ap);

                    [$datePart, $timePart] = $this->parseDatetime($ap['datetime']);

                    $this->pdo->prepare('
                        INSERT INTO activities
                            (itinerary_id, activity_type, identifier, location_name, location_address,
                             latitude, longitude, activity_date, start_time, cost, ticket_status,
                             details, sort_order, created_at, updated_at)
                        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                    ')->execute([
                        $itineraryId,
                        $ap['type'],
                        $ap['identification'],
                        $ap['location']['name'],
                        $ap['location']['address'],
                        $ap['location']['lat'],
                        $ap['location']['lng'],
                        $datePart,
                        $timePart,
                        $ap['cost'],
                        $ap['ticketStatus'],
                        $ap['details'],
                        $ap['sortOrder'],
                        $createdAt,
                        $createdAt,
                    ]);
                }

                $this->syncDerivedFields($itineraryId);
            }

            $this->pdo->commit();
        } catch (\Throwable $e) {
            $this->pdo->rollBack();
            throw $e;
        }

        $items = [];
        foreach ($importedIds as $id) {
            $items[] = $this->getItinerary($id);
        }

        return $this->json($response, [
            'importedCount' => count($items),
            'itineraries'   => $items,
        ], 201);
    }

    private function normalizeItineraryPayload(array $body): array
    {
        $currency = strtoupper(trim((string) ($body['currency'] ?? '')));
        return [
            'name'          => trim((string) ($body['name'] ?? '')),
            'description'   => trim((string) ($body['description'] ?? '')),
            'startDate'     => trim((string) ($body['startDate'] ?? '')),
            'endDate'       => trim((string) ($body['endDate'] ?? '')),
            'currency'      => $currency !== '' ? $currency : 'IDR',
            'image'         => isset($body['image']) && trim((string) $body['image']) !== ''
                                ? trim((string) $body['image'])
                                : null,
            'estimatedCost' => (float) ($body['estimatedCost'] ?? 0),
        ];
    }

    private function validateItineraryPayload(array $payload): void
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

    private function normalizeActivityPayload(array $body): array
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
            'sortOrder'    => (int) ($body['sortOrder'] ?? 0),
        ];
    }

    private function validateActivityPayload(array $payload): void
    {
        if ($payload['datetime'] === '') {
            throw new ValidationException('datetime is required');
        }
        $this->parseDatetime($payload['datetime']);
        if (!in_array($payload['type'], self::ACTIVITY_TYPES, true)) {
            throw new ValidationException('type is not supported');
        }
    }

    private function normalizeCreatedAt(string $value): string
    {
        $value = trim($value);
        if ($value === '') {
            return (new \DateTime())->format('Y-m-d H:i:s');
        }
        foreach (['Y-m-d\TH:i:sP', 'Y-m-d\TH:i:s\Z', 'Y-m-d\TH:i:s', 'Y-m-d H:i:s'] as $format) {
            $dt = \DateTime::createFromFormat($format, $value);
            if ($dt !== false) {
                return $dt->format('Y-m-d H:i:s');
            }
        }
        return (new \DateTime())->format('Y-m-d H:i:s');
    }
}
