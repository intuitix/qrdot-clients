<?php

declare(strict_types=1);

namespace Qrdot;

final class ApiError extends \RuntimeException
{
    /**
     * @param string $errorCode API error code (e.g. validation_error). Not Exception::$code.
     */
    public function __construct(
        public readonly string $errorCode,
        string $message,
        public readonly int $status,
    ) {
        parent::__construct($message, $status);
    }

    /** @deprecated Use $errorCode — kept for Node-SDK mental model */
    public function __get(string $name): mixed
    {
        if ($name === 'code') {
            return $this->errorCode;
        }
        throw new \Error('Undefined property ' . self::class . '::$' . $name);
    }
}

final class Client
{
    public readonly QrResource $qr;
    public readonly AssetsResource $assets;
    public readonly AnalyticsResource $analytics;
    public readonly WebhooksResource $webhooks;
    public readonly DomainsResource $domains;

    public function __construct(
        private readonly string $apiKey,
        private readonly string $baseUrl = 'https://api.qrdot.dev',
    ) {
        if (!str_starts_with($apiKey, 'sk_test_') && !str_starts_with($apiKey, 'sk_live_')) {
            throw new \InvalidArgumentException('apiKey must start with sk_test_ or sk_live_');
        }
        $this->qr = new QrResource($this);
        $this->assets = new AssetsResource($this);
        $this->analytics = new AnalyticsResource($this);
        $this->webhooks = new WebhooksResource($this);
        $this->domains = new DomainsResource($this);
    }

    /** Convenience — same as `$client->qr->create(...)`. */
    /** @param array<string, mixed> $payload */
    public function createQr(array $payload, ?string $idempotencyKey = null): mixed
    {
        return $this->qr->create($payload, $idempotencyKey);
    }

    /** @param array<string, mixed>|null $body */
    /** @param array<string, string>|null $headers */
    public function request(string $method, string $path, ?array $body = null, ?array $headers = null): mixed
    {
        [$data, $status] = $this->raw($method, $path, $body, $headers);
        if ($status === 204 || $data === '') {
            return null;
        }
        return json_decode($data, true, 512, JSON_THROW_ON_ERROR);
    }

    /** @param array<string, mixed>|null $body */
    /** @return array{0: string, 1: string} */
    public function requestBytes(string $method, string $path, ?array $body = null): array
    {
        [$data, , $ct] = $this->raw($method, $path, $body, null);
        $mime = $ct !== null ? trim(explode(';', $ct)[0]) : 'application/octet-stream';
        return [$data, $mime];
    }

    /** @param array<string, string> $headers */
    public function putExternal(string $url, string $body, array $headers): int
    {
        $ch = curl_init($url);
        if ($ch === false) {
            throw new \RuntimeException('curl_init failed');
        }
        curl_setopt_array($ch, [
            CURLOPT_CUSTOMREQUEST => 'PUT',
            CURLOPT_POSTFIELDS => $body,
            CURLOPT_HTTPHEADER => $this->formatHeaders($headers),
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => 60,
        ]);
        curl_exec($ch);
        $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        return $status;
    }

    /**
     * @param array<string, mixed>|null $body
     * @param array<string, string>|null $headers
     * @return array{0: string, 1: int, 2: ?string}
     */
    private function raw(
        string $method,
        string $path,
        ?array $body,
        ?array $headers,
    ): array {
        $hdrs = array_merge(
            [
                'Authorization' => 'Bearer ' . $this->apiKey,
                'Accept' => 'application/json',
            ],
            $headers ?? [],
        );
        $payload = null;
        if ($body !== null) {
            $payload = json_encode($body, JSON_THROW_ON_ERROR);
            $hdrs['Content-Type'] = 'application/json';
        }
        $ch = curl_init(rtrim($this->baseUrl, '/') . $path);
        if ($ch === false) {
            throw new \RuntimeException('curl_init failed');
        }
        curl_setopt_array($ch, [
            CURLOPT_CUSTOMREQUEST => $method,
            CURLOPT_HTTPHEADER => $this->formatHeaders($hdrs),
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => 60,
            CURLOPT_HEADER => true,
        ]);
        if ($payload !== null) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);
        }
        $raw = curl_exec($ch);
        if ($raw === false) {
            $err = curl_error($ch);
            throw new \RuntimeException($err);
        }
        $status = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $headerSize = (int) curl_getinfo($ch, CURLINFO_HEADER_SIZE);
        $headerBlob = substr((string) $raw, 0, $headerSize);
        $data = substr((string) $raw, $headerSize);
        $contentType = null;
        foreach (explode("\r\n", $headerBlob) as $line) {
            if (stripos($line, 'Content-Type:') === 0) {
                $contentType = trim(substr($line, strlen('Content-Type:')));
            }
        }
        if ($status < 200 || $status >= 300) {
            $code = 'internal';
            $message = 'Request failed';
            try {
                $parsed = json_decode($data, true, 512, JSON_THROW_ON_ERROR);
                $code = $parsed['error']['code'] ?? $code;
                $message = $parsed['error']['message'] ?? $message;
            } catch (\Throwable) {
            }
            throw new ApiError($code, $message, $status);
        }
        return [$data, $status, $contentType];
    }

    /** @param array<string, string> $headers */
    /** @return list<string> */
    private function formatHeaders(array $headers): array
    {
        $out = [];
        foreach ($headers as $k => $v) {
            $out[] = $k . ': ' . $v;
        }
        return $out;
    }
}

final class QrResource
{
    public function __construct(private readonly Client $client)
    {
    }

    /** @param array<string, mixed> $payload */
    public function create(array $payload, ?string $idempotencyKey = null): mixed
    {
        $headers = $idempotencyKey ? ['Idempotency-Key' => $idempotencyKey] : null;
        return $this->client->request('POST', '/v1/qr', $payload, $headers);
    }

    /** @param array<string, string> $query */
    public function list(array $query = []): mixed
    {
        $qs = $query ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', '/v1/qr' . $qs);
    }

    /** @param list<array<string, mixed>> $items */
    public function batch(array $items, ?string $idempotencyKey = null): mixed
    {
        $headers = $idempotencyKey ? ['Idempotency-Key' => $idempotencyKey] : null;
        return $this->client->request('POST', '/v1/qr/batch', ['items' => $items], $headers);
    }

    public function get(string $id): mixed
    {
        return $this->client->request('GET', '/v1/qr/' . rawurlencode($id));
    }

    /** @param array<string, mixed> $payload */
    public function update(string $id, array $payload): mixed
    {
        return $this->client->request('PATCH', '/v1/qr/' . rawurlencode($id), $payload);
    }

    public function delete(string $id): void
    {
        $this->client->request('DELETE', '/v1/qr/' . rawurlencode($id));
    }

    public function duplicate(string $id): mixed
    {
        return $this->client->request('POST', '/v1/qr/' . rawurlencode($id) . '/duplicate');
    }

    /** @return array{0: string, 1: string} */
    public function image(string $id, string $format = 'png'): array
    {
        return $this->client->requestBytes('GET', '/v1/qr/' . rawurlencode($id) . '/image.' . $format);
    }

    /** @param list<string> $ids */
    /** @return array{0: string, 1: string} */
    public function exportImages(array $ids, string $format = 'png'): array
    {
        return $this->client->requestBytes('POST', '/v1/qr/export/images', [
            'ids' => $ids,
            'format' => $format,
        ]);
    }
}

final class AssetsResource
{
    public function __construct(private readonly Client $client)
    {
    }

    public function presignLogo(string $contentType, ?string $filename = null): mixed
    {
        $body = ['content_type' => $contentType];
        if ($filename !== null) {
            $body['filename'] = $filename;
        }
        return $this->client->request('POST', '/v1/assets/logo/presign', $body);
    }

    public function completeLogo(string $id, ?string $filename = null): mixed
    {
        $complete = [];
        if ($filename !== null) {
            $complete['filename'] = $filename;
        }
        return $this->client->request(
            'POST',
            '/v1/assets/logo/' . rawurlencode($id) . '/complete',
            $complete,
        );
    }

    public function listLogos(): mixed
    {
        return $this->client->request('GET', '/v1/assets/logo');
    }

    public function getLogo(string $id): mixed
    {
        return $this->client->request('GET', '/v1/assets/logo/' . rawurlencode($id));
    }

    public function deleteLogo(string $id): void
    {
        $this->client->request('DELETE', '/v1/assets/logo/' . rawurlencode($id));
    }

    public function uploadLogo(string $bytes, string $contentType, ?string $filename = null): mixed
    {
        $presign = $this->presignLogo($contentType, $filename);
        $status = $this->client->putExternal(
            $presign['upload_url'],
            $bytes,
            $presign['headers'] ?? [],
        );
        if ($status < 200 || $status >= 300) {
            throw new ApiError('internal', "Logo storage upload failed ({$status})", $status);
        }
        return $this->completeLogo($presign['asset_id'], $filename);
    }
}

final class AnalyticsResource
{
    public function __construct(private readonly Client $client)
    {
    }

    /** @param array<string, string> $query */
    public function summary(array $query = []): mixed
    {
        $qs = $query ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', '/v1/analytics/summary' . $qs);
    }

    /** @param array<string, string> $query */
    public function qr(string $id, array $query = []): mixed
    {
        $qs = $query ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', '/v1/analytics/qr/' . rawurlencode($id) . $qs);
    }

    /** @param array<string, string> $query */
    public function scans(string $id, array $query = []): mixed
    {
        $qs = $query ? '?' . http_build_query($query) : '';
        return $this->client->request('GET', '/v1/analytics/qr/' . rawurlencode($id) . '/scans' . $qs);
    }
}

final class WebhooksResource
{
    public function __construct(private readonly Client $client)
    {
    }

    /** @param array<string, mixed> $payload */
    public function create(array $payload): mixed
    {
        $payload['events'] ??= ['qr.scanned'];
        return $this->client->request('POST', '/v1/webhooks', $payload);
    }

    public function list(): mixed
    {
        return $this->client->request('GET', '/v1/webhooks');
    }

    /** @param array<string, mixed> $patch */
    public function update(string $id, array $patch): mixed
    {
        return $this->client->request('PATCH', '/v1/webhooks/' . rawurlencode($id), $patch);
    }

    public function delete(string $id): void
    {
        $this->client->request('DELETE', '/v1/webhooks/' . rawurlencode($id));
    }

    /** @param array<string, mixed>|null $payload */
    public function test(string $id, ?array $payload = null): mixed
    {
        return $this->client->request('POST', '/v1/webhooks/' . rawurlencode($id) . '/test', $payload ?? []);
    }

    public function listDeliveries(string $id, ?int $limit = null): mixed
    {
        $qs = $limit !== null ? '?limit=' . rawurlencode((string) $limit) : '';
        return $this->client->request('GET', '/v1/webhooks/' . rawurlencode($id) . '/deliveries' . $qs);
    }

    /** @param array{qr_id: string, scan_id: string, ts: string} $payload */
    public function replay(string $id, array $payload): mixed
    {
        return $this->client->request('POST', '/v1/webhooks/' . rawurlencode($id) . '/replay', $payload);
    }

    public static function verifySignature(
        string $secret,
        string $rawBody,
        string $header,
        int $toleranceSec = 300,
    ): bool {
        return verify_webhook_signature($secret, $rawBody, $header, $toleranceSec);
    }
}

final class DomainsResource
{
    public function __construct(private readonly Client $client)
    {
    }

    /** @param array{hostname: string} $payload */
    public function create(array $payload): mixed
    {
        return $this->client->request('POST', '/v1/domains', $payload);
    }

    public function list(): mixed
    {
        return $this->client->request('GET', '/v1/domains');
    }

    public function get(string $id): mixed
    {
        return $this->client->request('GET', '/v1/domains/' . rawurlencode($id));
    }

    public function dns(string $id): mixed
    {
        return $this->client->request('GET', '/v1/domains/' . rawurlencode($id) . '/dns');
    }

    public function dnsProvider(string $id): mixed
    {
        return $this->client->request('GET', '/v1/domains/' . rawurlencode($id) . '/dns-provider');
    }

    public function domainConnectStart(string $id): mixed
    {
        return $this->client->request('POST', '/v1/domains/' . rawurlencode($id) . '/domain-connect/start');
    }

    /** @param array{email: string} $payload */
    public function forwardDns(string $id, array $payload): mixed
    {
        return $this->client->request('POST', '/v1/domains/' . rawurlencode($id) . '/dns/forward', $payload);
    }

    public function refresh(string $id): mixed
    {
        return $this->client->request('POST', '/v1/domains/' . rawurlencode($id) . '/refresh');
    }

    public function setDefault(string $id): mixed
    {
        return $this->client->request('POST', '/v1/domains/' . rawurlencode($id) . '/default');
    }

    public function delete(string $id): void
    {
        $this->client->request('DELETE', '/v1/domains/' . rawurlencode($id));
    }
}
