# Licensing keys (local / container)

Generate (from `backend/`):

```bash
go run ./scripts/generate_licensing_keys/main.go ./keys/licensing
# or: openssl genrsa -out keys/licensing/dev-private.pem 2048
```

Files (gitignored `*.pem`):
- `dev-private.pem` / `dev-public.pem` — RS256 for activation JWT + manifest
- `addon/paid-addon.js` — plaintext bundle encrypted per device on `/addon/paid-addon.js.enc`

For production set `LICENSING_PRIVATE_KEY_PATH` and `licensing.jwtKid=rs3-prod-1` (public key must match desktop `prodPublicKeyPem.js`).
