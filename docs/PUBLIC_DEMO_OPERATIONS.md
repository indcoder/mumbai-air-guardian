# Public demo operations

This is internal deployment and operations tracking. Judge-facing materials describe the intended public demo and its availability caveat without claiming a deployment state.

## Current preparation status

- Intended hostname: `air.hackmum.in`
- Public Cloud Run endpoint, DNS, TLS and Cloudflare protection: not verified
- Google Cloud CLI: not available in the verified environment; no deployment command was run
- Competition-input redistribution rights: unconfirmed; do not upload the excluded input files to a public service
- Application readiness: not deployment-ready as-is. The Go service currently serves the API and binds to loopback; frontend API-origin restrictions and static-asset serving/fallback behavior need a deployment-compatible design.
- Arduino interface: simulated only; do not expose a device-control endpoint
- Ollama: local optional service; do not expose its endpoint publicly

## Release gates

Before deploying, resolve data redistribution rights and select a compliant way to provide the required application data. Then verify the deployment design can safely serve the API and Angular application without exposing Ollama or Arduino interfaces. Add appropriate request limits and abuse protection before opening public access.

After deployment, verify all of the following against the public endpoint:

- HTTPS and expected hostname
- Angular loading, navigation and client-side routing
- `/api/health`
- Four-station Air Pulse and activity planning
- Deterministic cloud explanation fallback
- English, Marathi and Hindi interface
- PWA installability and assets
- No public Ollama endpoint or Arduino interface
- Cloudflare proxy/protection and anti-abuse controls

Record evidence and dates for each verification here. Prepare and test a rapid takedown procedure before announcing the endpoint.

## Takedown outline

If unexpected traffic or abuse requires suspension, use the authorized hosting and DNS controls to disable or restrict public access, preserve relevant operational logs, and verify that the hostname no longer serves the application. Keep the GitHub repository and reproducibility instructions available as the durable submission artifact. Document the exact authorized control-plane steps and responsible account before launch; none are verified in this checklist.
