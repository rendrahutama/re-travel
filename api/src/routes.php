<?php
declare(strict_types=1);

use App\Controllers\ActivityController;
use App\Controllers\AuthController;
use App\Controllers\ImportController;
use App\Controllers\ItineraryController;
use App\Controllers\SitemapController;
use App\Controllers\UploadController;
use App\Middleware\AuthMiddleware;
use App\Middleware\OptionalAuthMiddleware;
use Slim\Routing\RouteCollectorProxy;

$app->get('/health', function ($request, $response) {
    $response->getBody()->write(json_encode(['status' => 'ok']));
    return $response->withHeader('Content-Type', 'application/json');
});

$app->get('/sitemap.xml', [SitemapController::class, 'index']);

$app->group('/api', function (RouteCollectorProxy $group) {
    $group->post('/auth/logout', [AuthController::class, 'logout']);
    $group->get('/auth/me',     [AuthController::class, 'me'])->add(AuthMiddleware::class);

    $group->post('/upload/image', [UploadController::class, 'image'])->add(AuthMiddleware::class);

    $group->post('/import/local-storage', [ImportController::class, 'fromLocalStorage'])
          ->add(AuthMiddleware::class);

    $group->get('/itineraries',    [ItineraryController::class, 'index'])->add(OptionalAuthMiddleware::class);
    $group->post('/itineraries',   [ItineraryController::class, 'create'])->add(OptionalAuthMiddleware::class);
    $group->get('/itineraries/{id}',    [ItineraryController::class, 'show'])->add(OptionalAuthMiddleware::class);
    $group->map(['PUT', 'PATCH'], '/itineraries/{id}', [ItineraryController::class, 'update'])->add(OptionalAuthMiddleware::class);
    $group->delete('/itineraries/{id}', [ItineraryController::class, 'destroy'])->add(OptionalAuthMiddleware::class);

    $group->get('/itineraries/{id}/activities',    [ActivityController::class, 'index'])->add(OptionalAuthMiddleware::class);
    $group->post('/itineraries/{id}/activities',   [ActivityController::class, 'create'])->add(OptionalAuthMiddleware::class);
    $group->get('/itineraries/{id}/activities/{actId}',    [ActivityController::class, 'show'])->add(OptionalAuthMiddleware::class);
    $group->map(['PUT', 'PATCH'], '/itineraries/{id}/activities/{actId}', [ActivityController::class, 'update'])->add(OptionalAuthMiddleware::class);
    $group->delete('/itineraries/{id}/activities/{actId}', [ActivityController::class, 'destroy'])->add(OptionalAuthMiddleware::class);
    $group->post('/itineraries/{id}/activities/{actId}/move', [ActivityController::class, 'move'])->add(OptionalAuthMiddleware::class);
});
