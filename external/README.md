# External Service Adapters Layer

This directory acts as the dedicated location for wrapping integrations with third-party networks, services, or APIs that operate outside the internal OpenWiFi Service Discovery network mesh.

## Examples of what belongs here:
* Payment Gateway Clients (e.g. Stripe, PayPal, Braintree).
* Email/SMS Notifications Clients (e.g. SendGrid, Twilio).
* Cloud Storage APIs (e.g. AWS S3 SDK wrappers, Google Cloud Storage).
* Third-Party Authorization Providers (e.g. Auth0, Okta, Firebase Auth).

## Architecture Guideline:
1. Define a clear Go interface inside the `/internal/services` or `/internal/models` layer representing the contract your application requires.
2. Implement that interface inside `/external/<integration-name>/client.go` to isolate third-party library dependencies (such as custom SDK imports) from your core business logic.
3. Inject the client implementation into the service constructor at boot time in `cmd/main.go`.
