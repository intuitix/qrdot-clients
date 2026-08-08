# frozen_string_literal: true

require "json"
require "net/http"
require "openssl"
require "uri"
require_relative "qrdot/version"
require_relative "qrdot/client"

module Qrdot
  class Error < StandardError; end

  class ApiError < Error
    attr_reader :code, :status

    def initialize(code, message, status)
      super(message)
      @code = code
      @status = status
    end
  end

  # Verify X-Qrdot-Signature: t=…,v1=… over "#{t}.#{raw_body}"
  def self.verify_webhook_signature(secret, raw_body, header, tolerance_sec: 300)
    parts = {}
    header.to_s.split(",").each do |piece|
      k, v = piece.strip.split("=", 2)
      parts[k] = v if k && v
    end
    t = Integer(parts["t"], exception: false)
    v1 = parts["v1"]
    return false if t.nil? || t.zero? || v1.nil? || v1.empty?
    return false if (Time.now.to_i - t).abs > tolerance_sec

    expected = OpenSSL::HMAC.hexdigest("SHA256", secret, "#{t}.#{raw_body}")
    return false if expected.bytesize != v1.bytesize

    res = 0
    expected.bytes.zip(v1.bytes) { |a, b| res |= a ^ b }
    res.zero?
  end
end
