<?php
declare(strict_types=1);

namespace App\Controllers;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

class SitemapController extends BaseController
{
    public function index(Request $request, Response $response): Response
    {
        $siteUrl = rtrim($_ENV['SITE_URL'] ?? 'http://localhost:5173', '/');

        $stmt = $this->pdo->query('
            SELECT slug, created_at FROM itineraries
            WHERE is_public = 1
            ORDER BY start_date ASC, id ASC
        ');

        $urls   = [];
        $urls[] = ['loc' => $siteUrl . '/', 'changefreq' => 'daily', 'priority' => '1.0'];

        foreach ($stmt->fetchAll() as $row) {
            $lastMod = isset($row['created_at']) ? substr($row['created_at'], 0, 10) : '';
            $urls[]  = [
                'loc'        => "{$siteUrl}/itinerary/{$row['slug']}",
                'lastmod'    => $lastMod,
                'changefreq' => 'weekly',
                'priority'   => '0.8',
            ];
        }

        $xml  = '<?xml version="1.0" encoding="UTF-8"?>' . "\n";
        $xml .= '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">' . "\n";
        foreach ($urls as $url) {
            $xml .= "  <url>\n";
            $xml .= '    <loc>' . htmlspecialchars($url['loc']) . "</loc>\n";
            if (!empty($url['lastmod'])) {
                $xml .= "    <lastmod>{$url['lastmod']}</lastmod>\n";
            }
            $xml .= "    <changefreq>{$url['changefreq']}</changefreq>\n";
            $xml .= "    <priority>{$url['priority']}</priority>\n";
            $xml .= "  </url>\n";
        }
        $xml .= '</urlset>';

        $response->getBody()->write($xml);
        return $response->withHeader('Content-Type', 'application/xml; charset=utf-8');
    }
}
