# intuitix/qrdot-php

Official PHP client for the [QR.](https://qrdot.dev) API.

```bash
composer require intuitix/qrdot-php
```

Source: public mirror [`intuitix/qrdot-clients`](https://github.com/intuitix/qrdot-clients) (`php/`). Packagist package path = `php`.

## Quick start

```php
<?php
use Qrdot\Client;
use function Qrdot\verify_webhook_signature;

$client = new Client(getenv('QRDOT_API_KEY'));

$qr = $client->qr->create([
    'targetUrl' => 'https://example.com',
    'name' => 'Launch',
    'style' => [
        'dark_color' => '#0F766E',
        'light_color' => '#FFFFFF',
    ],
]);

[$png, $mime] = $client->qr->image($qr['id'], 'png');
[$zip, ] = $client->qr->exportImages([$qr['id']], 'png');

$ok = verify_webhook_signature($secret, $rawBody, $_SERVER['HTTP_X_QRDOT_SIGNATURE'] ?? '');
```

## Docs

- Guides: https://qrdot.dev/docs/
- OpenAPI: https://api.qrdot.dev/v1/openapi.json
