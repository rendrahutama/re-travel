<?php
declare(strict_types=1);

use App\Middleware\CorsMiddleware;
use DI\Container;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Factory\AppFactory;

require __DIR__ . '/../vendor/autoload.php';

$dotenv = Dotenv\Dotenv::createImmutable(__DIR__ . '/..');
$dotenv->safeLoad();

$container = new Container();

$container->set(PDO::class, function (): PDO {
    $host = $_ENV['DB_HOST'] ?? 'localhost';
    $port = $_ENV['DB_PORT'] ?? '3306';
    $name = $_ENV['DB_NAME'] ?? 'retravel';
    $user = $_ENV['DB_USER'] ?? 'root';
    $pass = $_ENV['DB_PASS'] ?? '';

    $dsn = "mysql:host={$host};port={$port};dbname={$name};charset=utf8mb4";
    return new PDO($dsn, $user, $pass, [
        PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        PDO::ATTR_EMULATE_PREPARES   => false,
    ]);
});

$container->set(\App\Db\SsoPdo::class, function (): \App\Db\SsoPdo {
    $host = $_ENV['SSO_DB_HOST'] ?? $_ENV['DB_HOST'] ?? 'localhost';
    $port = $_ENV['SSO_DB_PORT'] ?? $_ENV['DB_PORT'] ?? '3306';
    $name = $_ENV['SSO_DB_NAME'] ?? 'relogin';
    $user = $_ENV['SSO_DB_USER'] ?? $_ENV['DB_USER'] ?? 'root';
    $pass = $_ENV['SSO_DB_PASS'] ?? $_ENV['DB_PASS'] ?? '';

    $dsn = "mysql:host={$host};port={$port};dbname={$name};charset=utf8mb4";
    return new \App\Db\SsoPdo($dsn, $user, $pass, [
        \PDO::ATTR_ERRMODE            => \PDO::ERRMODE_EXCEPTION,
        \PDO::ATTR_DEFAULT_FETCH_MODE => \PDO::FETCH_ASSOC,
        \PDO::ATTR_EMULATE_PREPARES   => false,
    ]);
});

AppFactory::setContainer($container);
$app = AppFactory::create();

$basePath = $_ENV['APP_BASE_PATH'] ?? '';
if ($basePath !== '') {
    $app->setBasePath($basePath);
}

$app->addBodyParsingMiddleware();
$app->addRoutingMiddleware();

$debug = filter_var($_ENV['APP_DEBUG'] ?? 'false', FILTER_VALIDATE_BOOLEAN);
$errorMiddleware = $app->addErrorMiddleware($debug, true, true);
$errorMiddleware->setDefaultErrorHandler(function (
    Request $request,
    Throwable $exception,
    bool $displayErrorDetails,
) use ($app): Response {
    $statusCode = 500;
    $message    = 'Internal server error';

    if ($exception instanceof \App\Exception\HttpException) {
        $statusCode = $exception->getStatusCode();
        $message    = $exception->getMessage();
    } elseif ($displayErrorDetails) {
        $message = $exception->getMessage();
    }

    $payload = ['error' => $message];
    if ($displayErrorDetails) {
        $payload['trace'] = $exception->getTraceAsString();
    }

    $response = $app->getResponseFactory()->createResponse();
    $response->getBody()->write(json_encode($payload));
    return $response
        ->withStatus($statusCode)
        ->withHeader('Content-Type', 'application/json');
});

$app->add(new CorsMiddleware());

require __DIR__ . '/../src/routes.php';

$app->run();
