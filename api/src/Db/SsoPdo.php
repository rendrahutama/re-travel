<?php
declare(strict_types=1);

namespace App\Db;

// Thin subclass used as a distinct type so PHP-DI can inject the SSO DB
// connection separately from the app's main PDO connection.
class SsoPdo extends \PDO {}
