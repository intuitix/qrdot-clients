<?php

declare(strict_types=1);

namespace Qrdot;

function verify_webhook_signature(
    string $secret,
    string $rawBody,
    string $header,
    int $toleranceSec = 300,
): bool {
    $parts = [];
    foreach (explode(',', $header) as $piece) {
        $kv = explode('=', trim($piece), 2);
        if (count($kv) === 2) {
            $parts[$kv[0]] = $kv[1];
        }
    }
    if (!isset($parts['t'], $parts['v1']) || !ctype_digit($parts['t'])) {
        return false;
    }
    $t = (int) $parts['t'];
    if (abs(time() - $t) > $toleranceSec) {
        return false;
    }
    $expected = hash_hmac('sha256', $t . '.' . $rawBody, $secret);
    return hash_equals($expected, $parts['v1']);
}
