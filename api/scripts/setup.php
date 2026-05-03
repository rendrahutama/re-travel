<?php
declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

$dotenv = Dotenv\Dotenv::createImmutable(__DIR__ . '/..');
$dotenv->safeLoad();

$host = $_ENV['DB_HOST'] ?? 'localhost';
$port = $_ENV['DB_PORT'] ?? '3306';
$name = $_ENV['DB_NAME'] ?? 'reitinerary';
$user = $_ENV['DB_USER'] ?? 'root';
$pass = $_ENV['DB_PASS'] ?? '';

echo "Connecting to MySQL ({$host}:{$port}/{$name})...\n";

$pdo = new PDO(
    "mysql:host={$host};port={$port};dbname={$name};charset=utf8mb4",
    $user,
    $pass,
    [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
);

echo "Running migrations...\n";

$sql = file_get_contents(__DIR__ . '/../migrations/001_schema.sql');
foreach (explode(';', $sql) as $statement) {
    $statement = trim($statement);
    if ($statement === '') {
        continue;
    }
    try {
        $pdo->exec($statement);
    } catch (\PDOException $e) {
        // Ignore "duplicate key name" — index already exists
        if (!str_contains($e->getMessage(), '1061')) {
            throw $e;
        }
    }
}

echo "Schema applied.\n";

$defaultName  = $_ENV['DEFAULT_USER_NAME']     ?? 'Demo User';
$defaultEmail = $_ENV['DEFAULT_USER_EMAIL']    ?? 'demo@reitinerary.local';
$defaultPass  = $_ENV['DEFAULT_USER_PASSWORD'] ?? 'demo-password';

$hash = password_hash($defaultPass, PASSWORD_BCRYPT);
$pdo->prepare('INSERT IGNORE INTO users (name, email, password) VALUES (?, ?, ?)')
    ->execute([$defaultName, $defaultEmail, $hash]);

echo "Default user ensured ({$defaultEmail}).\n";
echo "Setup complete.\n";
