# frozen_string_literal: true

module Qrdot
  class Client
    attr_reader :qr, :assets, :analytics, :webhooks, :domains

    def initialize(api_key, base_url: "https://api.qrdot.dev")
      unless api_key.start_with?("sk_test_", "sk_live_")
        raise ArgumentError, "api_key must start with sk_live_ (legacy sk_test_ accepted)"
      end

      @api_key = api_key
      @base_url = base_url.sub(%r{/\z}, "")
      @qr = QrResource.new(self)
      @assets = AssetsResource.new(self)
      @analytics = AnalyticsResource.new(self)
      @webhooks = WebhooksResource.new(self)
      @domains = DomainsResource.new(self)
    end

    # Convenience — same as qr.create
    def create_qr(payload, idempotency_key: nil)
      qr.create(payload, idempotency_key: idempotency_key)
    end

    def request(method, path, body: nil, headers: {})
      data, status, = raw(method, path, body: body, headers: headers)
      return nil if status == 204 || data.nil? || data.empty?

      JSON.parse(data)
    end

    def request_bytes(method, path, body: nil)
      data, _status, ct = raw(method, path, body: body, headers: {})
      mime = (ct || "application/octet-stream").split(";").first.strip
      [data, mime]
    end

    def put_external(url, body, headers)
      uri = URI(url)
      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = uri.scheme == "https"
      req = Net::HTTP::Put.new(uri)
      headers.each { |k, v| req[k] = v }
      req.body = body
      res = http.request(req)
      res.code.to_i
    end

    def raw(method, path, body:, headers:)
      uri = URI("#{@base_url}#{path}")
      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = uri.scheme == "https"
      http.open_timeout = 30
      http.read_timeout = 60

      req = Net::HTTPGenericRequest.new(
        method,
        !body.nil?,
        true,
        uri.request_uri
      )
      req["Authorization"] = "Bearer #{@api_key}"
      req["Accept"] = "application/json"
      headers.each { |k, v| req[k] = v }
      if body
        req["Content-Type"] = "application/json"
        req.body = JSON.generate(body)
      end

      res = http.request(req)
      status = res.code.to_i
      if status < 200 || status >= 300
        code = "internal"
        message = res.message
        begin
          parsed = JSON.parse(res.body.to_s)
          code = parsed.dig("error", "code") || code
          message = parsed.dig("error", "message") || message
        rescue JSON::ParserError
          # ignore
        end
        raise ApiError.new(code, message, status)
      end
      [res.body.to_s, status, res["content-type"]]
    end
  end

  class QrResource
    def initialize(client)
      @c = client
    end

    def create(payload, idempotency_key: nil)
      headers = {}
      headers["Idempotency-Key"] = idempotency_key if idempotency_key
      @c.request("POST", "/v1/qr", body: payload, headers: headers)
    end

    def list(query = {})
      qs = query.empty? ? "" : "?#{URI.encode_www_form(query)}"
      @c.request("GET", "/v1/qr#{qs}")
    end

    def batch(items, idempotency_key: nil)
      headers = {}
      headers["Idempotency-Key"] = idempotency_key if idempotency_key
      @c.request("POST", "/v1/qr/batch", body: { items: items }, headers: headers)
    end

    def get(id)
      @c.request("GET", "/v1/qr/#{URI.encode_www_form_component(id)}")
    end

    def update(id, payload)
      @c.request("PATCH", "/v1/qr/#{URI.encode_www_form_component(id)}", body: payload)
    end

    def delete(id)
      @c.request("DELETE", "/v1/qr/#{URI.encode_www_form_component(id)}")
    end

    def duplicate(id)
      @c.request("POST", "/v1/qr/#{URI.encode_www_form_component(id)}/duplicate")
    end

    def image(id, format = "png")
      @c.request_bytes("GET", "/v1/qr/#{URI.encode_www_form_component(id)}/image.#{format}")
    end

    def export_images(ids, format = "png")
      @c.request_bytes("POST", "/v1/qr/export/images", body: { ids: ids, format: format })
    end
  end

  class AssetsResource
    def initialize(client)
      @c = client
    end

    def presign_logo(content_type, filename: nil)
      body = { content_type: content_type }
      body[:filename] = filename if filename
      @c.request("POST", "/v1/assets/logo/presign", body: body)
    end

    def complete_logo(id, filename: nil)
      complete = {}
      complete[:filename] = filename if filename
      @c.request(
        "POST",
        "/v1/assets/logo/#{URI.encode_www_form_component(id)}/complete",
        body: complete
      )
    end

    def list_logos
      @c.request("GET", "/v1/assets/logo")
    end

    def get_logo(id)
      @c.request("GET", "/v1/assets/logo/#{URI.encode_www_form_component(id)}")
    end

    def delete_logo(id)
      @c.request("DELETE", "/v1/assets/logo/#{URI.encode_www_form_component(id)}")
    end

    def upload_logo(bytes, content_type, filename: nil)
      presign = presign_logo(content_type, filename: filename)
      status = @c.put_external(presign["upload_url"], bytes, presign["headers"] || {})
      if status < 200 || status >= 300
        raise ApiError.new("internal", "Logo storage upload failed (#{status})", status)
      end
      complete_logo(presign["asset_id"], filename: filename)
    end
  end

  class AnalyticsResource
    def initialize(client)
      @c = client
    end

    def summary(query = {})
      qs = query.empty? ? "" : "?#{URI.encode_www_form(query)}"
      @c.request("GET", "/v1/analytics/summary#{qs}")
    end

    def qr(id, query = {})
      qs = query.empty? ? "" : "?#{URI.encode_www_form(query)}"
      @c.request("GET", "/v1/analytics/qr/#{URI.encode_www_form_component(id)}#{qs}")
    end

    def scans(id, query = {})
      qs = query.empty? ? "" : "?#{URI.encode_www_form(query)}"
      @c.request("GET", "/v1/analytics/qr/#{URI.encode_www_form_component(id)}/scans#{qs}")
    end
  end

  class WebhooksResource
    def initialize(client)
      @c = client
    end

    def create(payload)
      payload = payload.dup
      payload[:events] ||= payload["events"] || ["qr.scanned"]
      @c.request("POST", "/v1/webhooks", body: payload)
    end

    def list
      @c.request("GET", "/v1/webhooks")
    end

    def update(id, patch)
      @c.request("PATCH", "/v1/webhooks/#{URI.encode_www_form_component(id)}", body: patch)
    end

    def delete(id)
      @c.request("DELETE", "/v1/webhooks/#{URI.encode_www_form_component(id)}")
    end

    def test(id, payload = {})
      @c.request("POST", "/v1/webhooks/#{URI.encode_www_form_component(id)}/test", body: payload)
    end

    def list_deliveries(id, limit: nil)
      qs = limit.nil? ? "" : "?limit=#{URI.encode_www_form_component(limit.to_s)}"
      @c.request("GET", "/v1/webhooks/#{URI.encode_www_form_component(id)}/deliveries#{qs}")
    end

    def replay(id, payload)
      @c.request("POST", "/v1/webhooks/#{URI.encode_www_form_component(id)}/replay", body: payload)
    end

    def verify_signature(secret, raw_body, header, tolerance_sec: 300)
      Qrdot.verify_webhook_signature(secret, raw_body, header, tolerance_sec: tolerance_sec)
    end
  end

  class DomainsResource
    def initialize(client)
      @c = client
    end

    def create(payload)
      @c.request("POST", "/v1/domains", body: payload)
    end

    def list
      @c.request("GET", "/v1/domains")
    end

    def get(id)
      @c.request("GET", "/v1/domains/#{URI.encode_www_form_component(id)}")
    end

    def dns(id)
      @c.request("GET", "/v1/domains/#{URI.encode_www_form_component(id)}/dns")
    end

    def dns_provider(id)
      @c.request("GET", "/v1/domains/#{URI.encode_www_form_component(id)}/dns-provider")
    end

    def domain_connect_start(id)
      @c.request("POST", "/v1/domains/#{URI.encode_www_form_component(id)}/domain-connect/start")
    end

    def forward_dns(id, payload)
      @c.request("POST", "/v1/domains/#{URI.encode_www_form_component(id)}/dns/forward", body: payload)
    end

    def refresh(id)
      @c.request("POST", "/v1/domains/#{URI.encode_www_form_component(id)}/refresh")
    end

    def set_default(id)
      @c.request("POST", "/v1/domains/#{URI.encode_www_form_component(id)}/default")
    end

    def delete(id)
      @c.request("DELETE", "/v1/domains/#{URI.encode_www_form_component(id)}")
    end
  end
end
