# frozen_string_literal: true

require_relative "lib/qrdot/version"

Gem::Specification.new do |spec|
  spec.name          = "qrdot"
  spec.version       = Qrdot::VERSION
  spec.authors       = ["QR."]
  spec.email         = ["hello@qrdot.dev"]
  spec.summary       = "Official Ruby client for the QR. API"
  spec.description   = "Thin REST client for qrdot.dev — create dynamic QR codes, images, logos, webhooks."
  spec.homepage      = "https://qrdot.dev/libraries/"
  spec.license       = "MIT"
  spec.required_ruby_version = ">= 3.1.0"
  spec.metadata["source_code_uri"] = "https://github.com/intuitix/qrdot/tree/main/clients/ruby"
  spec.metadata["documentation_uri"] = "https://qrdot.dev/docs/"
  spec.files = Dir["lib/**/*", "README.md", "LICENSE"]
  spec.require_paths = ["lib"]
  spec.add_development_dependency "minitest", "~> 5.0"
end
