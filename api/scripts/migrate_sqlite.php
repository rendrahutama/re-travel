<?php
declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

$dotenv = Dotenv\Dotenv::createImmutable(__DIR__ . '/..');
$dotenv->safeLoad();

$sqlitePath = $argv[1] ?? __DIR__ . '/../../api-go/db/re_itinerary.db';

if (!file_exists($sqlitePath)) {
    echo "SQLite file not found: {$sqlitePath}\n";
    exit(1);
}

$sqlite = new PDO("sqlite:{$sqlitePath}", options: [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]);

$host = $_ENV['DB_HOST'] ?? 'localhost';
$port = $_ENV['DB_PORT'] ?? '3306';
$name = $_ENV['DB_NAME'] ?? 'reitinerary';
$user = $_ENV['DB_USER'] ?? 'root';
$pass = $_ENV['DB_PASS'] ?? '';

$mysql = new PDO(
    "mysql:host={$host};port={$port};dbname={$name};charset=utf8mb4",
    $user,
    $pass,
    [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
);

echo "Migrating users...\n";
$users = $sqlite->query("SELECT * FROM users")->fetchAll(PDO::FETCH_ASSOC);
$stmt  = $mysql->prepare("INSERT IGNORE INTO users (id, name, email, password, created_at) VALUES (?, ?, ?, ?, ?)");
foreach ($users as $row) {
    $stmt->execute([$row['id'], $row['name'], $row['email'], $row['password'], $row['created_at']]);
}
echo "  " . count($users) . " users done.\n";

echo "Migrating itineraries...\n";
$itineraries = $sqlite->query("SELECT * FROM itineraries")->fetchAll(PDO::FETCH_ASSOC);
$stmt = $mysql->prepare("INSERT IGNORE INTO itineraries
    (id, user_id, slug, name, description, start_date, end_date, currency, cover_image_url, estimated_cost, is_public, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)");
foreach ($itineraries as $row) {
    $stmt->execute([
        $row['id'], $row['user_id'], $row['slug'], $row['name'], $row['description'],
        $row['start_date'], $row['end_date'], $row['currency'], $row['cover_image_url'],
        $row['estimated_cost'], $row['is_public'], $row['created_at'], $row['updated_at'],
    ]);
}
echo "  " . count($itineraries) . " itineraries done.\n";

echo "Migrating activities...\n";
$activities = $sqlite->query("SELECT * FROM activities")->fetchAll(PDO::FETCH_ASSOC);
$stmt = $mysql->prepare("INSERT IGNORE INTO activities
    (id, itinerary_id, activity_type, identifier, name, location_name, location_address,
     latitude, longitude, activity_date, start_time, cost, ticket_status, details, sort_order, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)");
foreach ($activities as $row) {
    $stmt->execute([
        $row['id'], $row['itinerary_id'], $row['activity_type'], $row['identifier'], $row['name'],
        $row['location_name'], $row['location_address'], $row['latitude'], $row['longitude'],
        $row['activity_date'], $row['start_time'], $row['cost'], $row['ticket_status'],
        $row['details'], $row['sort_order'], $row['created_at'], $row['updated_at'],
    ]);
}
echo "  " . count($activities) . " activities done.\n";

echo "Migration complete.\n";
