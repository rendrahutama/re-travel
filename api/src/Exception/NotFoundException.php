<?php
declare(strict_types=1);

namespace App\Exception;

class NotFoundException extends HttpException
{
    public function __construct(string $resource)
    {
        parent::__construct("{$resource} not found", 404);
    }
}
