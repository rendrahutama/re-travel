<?php
/**
 * OG meta tag handler for social crawlers.
 * Apache routes bot User-Agents on /itinerary/* here via .htaccess.
 * Fetches itinerary data from the API and returns minimal HTML with OG tags.
 */

function loadEnv(string $path): array
{
    if (!file_exists($path)) return [];
    $vars = [];
    foreach (file($path, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) as $line) {
        $line = trim($line);
        if ($line === '' || str_starts_with($line, '#')) continue;
        [$key, $val] = array_pad(explode('=', $line, 2), 2, '');
        $vars[trim($key)] = trim($val, " \t\"'");
    }
    return $vars;
}

$env = loadEnv(__DIR__ . '/.env');

define('API_BASE_URL', rtrim($env['VITE_API_BASE_URL'] ?? 'http://localhost:8080', '/'));
define('SITE_URL',     rtrim($env['VITE_SITE_URL']     ?? 'http://localhost:5173', '/'));
define('APP_NAME',     'Re-Itinerary');

function fetchItinerary(string $slug): ?array
{
    $url = API_BASE_URL . '/api/itineraries/' . rawurlencode($slug);
    $ctx = stream_context_create(['http' => ['timeout' => 5, 'ignore_errors' => true]]);
    $body = @file_get_contents($url, false, $ctx);
    if ($body === false) return null;
    $data = json_decode($body, true);
    return is_array($data) && isset($data['name']) ? $data : null;
}

function extractSlug(): ?string
{
    $uri  = $_SERVER['REQUEST_URI'] ?? '';
    $path = parse_url($uri, PHP_URL_PATH);
    if (preg_match('~^/itinerary/([^/?#]+)~', $path, $m)) {
        return $m[1];
    }
    return null;
}

function renderOg(string $title, string $desc, ?string $image, string $url): void
{
    $title = htmlspecialchars($title, ENT_QUOTES);
    $desc  = htmlspecialchars($desc,  ENT_QUOTES);
    $url   = htmlspecialchars($url,   ENT_QUOTES);
    $image = $image ? htmlspecialchars($image, ENT_QUOTES) : '';

    echo <<<HTML
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{$title}</title>
  <meta name="description" content="{$desc}">
  <meta property="og:title"       content="{$title}">
  <meta property="og:description" content="{$desc}">
  <meta property="og:type"        content="website">
  <meta property="og:url"         content="{$url}">
  <meta name="twitter:card"        content="summary_large_image">
  <meta name="twitter:title"       content="{$title}">
  <meta name="twitter:description" content="{$desc}">
HTML;

    if ($image !== '') {
        echo "  <meta property=\"og:image\"    content=\"{$image}\">\n";
        echo "  <meta name=\"twitter:image\" content=\"{$image}\">\n";
    }

    echo <<<HTML
  <meta http-equiv="refresh" content="0;url={$url}">
</head>
<body></body>
</html>
HTML;
}

function renderFallback(): void
{
    renderOg(
        APP_NAME . ' | Travel Itinerary Planner',
        'Plan trips, organize activities, track locations, and manage travel schedules with RE-ITINERARY.',
        null,
        SITE_URL
    );
}

$slug = extractSlug();

if ($slug === null) {
    renderFallback();
    exit;
}

$itinerary = fetchItinerary($slug);

if ($itinerary === null) {
    renderFallback();
    exit;
}

$name  = $itinerary['name'] ?? 'Itinerary';
$desc  = trim($itinerary['description'] ?? '');
$image = $itinerary['image'] ?? null;

$dates = '';
if (!empty($itinerary['startDate'])) {
    $dates = date('d M Y', strtotime($itinerary['startDate']));
    if (!empty($itinerary['endDate']) && $itinerary['endDate'] !== $itinerary['startDate']) {
        $dates .= ' – ' . date('d M Y', strtotime($itinerary['endDate']));
    }
}

if ($desc === '' && $dates !== '') {
    $desc = $dates;
} elseif ($dates !== '') {
    $desc = $dates . ' · ' . $desc;
}

if ($desc === '') {
    $desc = 'View this itinerary on ' . APP_NAME . '.';
}

renderOg(
    $name . ' | ' . APP_NAME,
    $desc,
    $image,
    SITE_URL . '/itinerary/' . rawurlencode($slug)
);
