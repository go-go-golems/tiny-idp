# tiny-idp Strict Engine Key Rotation

The strict engine signs ID Tokens with the active key in `storage.KeyStore` and publishes `VerificationKeys` at `/jwks`.

## Rotation invariant

A safe rotation has three states:

1. A new RSA key is generated and stored.
2. The new key becomes active for new ID Tokens.
3. The previously active key is retired but remains published in JWKS so relying parties can validate old ID Tokens until expiry.

`internal/keys.RotateRSA` implements this invariant for repository stores.

## Operational sequence

```text
1. Confirm the store is persistent in production.
2. Generate a new key ID that has never been used before.
3. Rotate to the new key.
4. Confirm `/jwks` contains both old and new kids.
5. Confirm a new ID Token has the new `kid` and verifies against JWKS.
6. Keep the retired key published for at least the maximum ID Token lifetime plus clock skew.
7. Only remove old verification material after that retention period and after audit review.
```

## Validation

```bash
go test ./internal/keys ./internal/store/sqlite ./internal/fositeadapter
```

Important tests:

- `TestRotateRSAActivatesNewKeyAndKeepsOldVerifiable`
- `TestSigningKeyRotationPersistsRetiredVerificationKey`
- strict ID Token JWKS validation in `TestStrictAuthorizationCodeFlow`
