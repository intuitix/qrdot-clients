# frozen_string_literal: true

require "minitest/autorun"
require "openssl"
require_relative "../lib/qrdot"

class TestQrdot < Minitest::Test
  def test_rejects_bad_key
    assert_raises(ArgumentError) { Qrdot::Client.new("bad") }
  end

  def test_verify_webhook_signature
    secret = "whsec_test"
    body = '{"ok":true}'
    t = Time.now.to_i
    sig = OpenSSL::HMAC.hexdigest("SHA256", secret, "#{t}.#{body}")
    header = "t=#{t},v1=#{sig}"
    assert Qrdot.verify_webhook_signature(secret, body, header)
    refute Qrdot.verify_webhook_signature(secret, "#{body}x", header)
  end

  def test_client_exposes_node_parity_resources
    c = Qrdot::Client.new("sk_test_abc")
    assert_respond_to c, :domains
    assert_respond_to c.domains, :domain_connect_start
    assert_respond_to c.domains, :forward_dns
    assert_respond_to c.domains, :set_default
    assert_respond_to c.webhooks, :list_deliveries
    assert_respond_to c.webhooks, :replay
    assert_respond_to c.assets, :presign_logo
    assert_respond_to c.assets, :complete_logo
    assert_respond_to c.assets, :get_logo
    assert_respond_to c, :create_qr
  end
end
