<?php
declare(strict_types=1);

namespace App\Exception;

class ForbiddenException extends HttpException
{
    public function __construct()
    {
        parent::__construct('access denied', 403);
    }
}
