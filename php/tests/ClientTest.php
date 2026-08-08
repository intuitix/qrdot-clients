<?php

declare(strict_types=1);

namespace Qrdot\Tests;

use PHPUnit\Framework\TestCase;
use Qrdot\ApiError;
use Qrdot\Client;
use function Qrdot\verify_webhook_signature;

final class ClientTest extends TestCase
{
    public function testRejectsBadKey(): void
    {
        $this->expectException(\InvalidArgumentException::class);
        new Client('bad');
    }

    public function testVerifyWebhookSignature(): void
    {
        $secret = 'whsec_test';
        $body = '{"ok":true}';
        $t = time();
        $sig = hash_hmac('sha256', $t . '.' . $body, $secret);
        $header = "t={$t},v1={$sig}";
        $this->assertTrue(verify_webhook_signature($secret, $body, $header));
        $this->assertFalse(verify_webhook_signature($secret, $body . 'x', $header));
    }

    public function testApiErrorType(): void
    {
        $e = new ApiError('validation', 'bad', 400);
        $this->assertSame('validation', $e->code);
        $this->assertSame(400, $e->status);
    }

    public function testClientExposesNodeParityResources(): void
    {
        $c = new Client('sk_test_abc');
        $this->assertTrue(isset($c->domains));
        $this->assertTrue(method_exists($c->domains, 'domainConnectStart'));
        $this->assertTrue(method_exists($c->domains, 'forwardDns'));
        $this->assertTrue(method_exists($c->domains, 'setDefault'));
        $this->assertTrue(method_exists($c->webhooks, 'listDeliveries'));
        $this->assertTrue(method_exists($c->webhooks, 'replay'));
        $this->assertTrue(method_exists($c->assets, 'presignLogo'));
        $this->assertTrue(method_exists($c->assets, 'completeLogo'));
        $this->assertTrue(method_exists($c->assets, 'getLogo'));
        $this->assertTrue(method_exists($c, 'createQr'));
    }
}
