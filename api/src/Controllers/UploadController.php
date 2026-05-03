<?php
declare(strict_types=1);

namespace App\Controllers;

use App\Exception\HttpException;
use App\Exception\ValidationException;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;

class UploadController extends BaseController
{
    private const MAX_SIZE     = 5 * 1024 * 1024;
    private const ALLOWED_MIME = ['image/jpeg', 'image/png', 'image/webp', 'image/gif'];
    private const EXT_MAP      = [
        'image/jpeg' => 'jpg',
        'image/png'  => 'png',
        'image/webp' => 'webp',
        'image/gif'  => 'gif',
    ];

    public function image(Request $request, Response $response): Response
    {
        $userId = $request->getAttribute('userId');
        if ($userId === null) {
            throw new HttpException('unauthorized', 401);
        }

        $files = $request->getUploadedFiles();
        $file  = $files['image'] ?? null;

        if ($file === null || $file->getError() !== UPLOAD_ERR_OK) {
            throw new ValidationException('image file is required');
        }

        if ($file->getSize() > self::MAX_SIZE) {
            throw new ValidationException('image must be under 5MB');
        }

        $mime = $file->getClientMediaType();
        if (!in_array($mime, self::ALLOWED_MIME, true)) {
            throw new ValidationException('only JPEG, PNG, WebP, and GIF are allowed');
        }

        $ext       = self::EXT_MAP[$mime];
        $filename  = bin2hex(random_bytes(16)) . '.' . $ext;
        $uploadDir = __DIR__ . '/../../public/uploads';

        $file->moveTo($uploadDir . '/' . $filename);

        $uri      = $request->getUri();
        $host     = $uri->getHost();
        $port     = $uri->getPort();
        if ($port && $port !== 80 && $port !== 443) {
            $host .= ':' . $port;
        }
        $basePath = $_ENV['APP_BASE_PATH'] ?? '';
        $url      = $uri->getScheme() . '://' . $host . $basePath . '/uploads/' . $filename;

        return $this->json($response, ['url' => $url], 201);
    }
}
