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

$pdo = new PDO(
    "mysql:host={$host};port={$port};dbname={$name};charset=utf8mb4",
    $user,
    $pass,
    [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
);

$uploadDir = __DIR__ . '/../public/uploads';
$basePath  = $_ENV['APP_BASE_PATH'] ?? '';
$baseUrl   = 'http://localhost' . $basePath . '/uploads';

$rows = $pdo->query("SELECT id, cover_image_url FROM itineraries WHERE cover_image_url LIKE 'data:image/%'")->fetchAll(PDO::FETCH_ASSOC);

if (empty($rows)) {
    echo "No base64 images found.\n";
    exit(0);
}

foreach ($rows as $row) {
    if (!preg_match('/^data:image\/(\w+);base64,(.+)$/', $row['cover_image_url'], $m)) {
        echo "Skipping id={$row['id']}: unrecognized format\n";
        continue;
    }

    $ext      = $m[1] === 'jpeg' ? 'jpg' : $m[1];
    $data     = base64_decode($m[2]);
    $filename = bin2hex(random_bytes(16)) . '.' . $ext;

    file_put_contents($uploadDir . '/' . $filename, $data);

    $url = $baseUrl . '/' . $filename;
    $pdo->prepare('UPDATE itineraries SET cover_image_url = ? WHERE id = ?')->execute([$url, $row['id']]);

    echo "Migrated id={$row['id']} → {$filename}\n";
}

echo "Done.\n";
