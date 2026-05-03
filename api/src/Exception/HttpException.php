<?php
declare(strict_types=1);

namespace App\Exception;

use RuntimeException;

class HttpException extends RuntimeException
{
    public function __construct(
        string $message,
        private readonly int $statusCode = 500,
    ) {
        parent::__construct($message);
    }

    public function getStatusCode(): int
    {
        return $this->statusCode;
    }
}
