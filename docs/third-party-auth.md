# Google and Apple sign-in

The API validates provider-signed ID tokens with cached JWKS public keys. It
never stores an ID token, access token, refresh token, or Apple authorization
code. Local accounts are linked by the provider's stable `sub` claim, not by
email address.

## Configuration

Set every OAuth client ID that may be used by the app. Google usually needs the
Android, iOS and Web OAuth client IDs. Apple needs the relevant bundle ID and/or
Services ID.

```yaml
third_party_auth:
  google:
    client_ids: [google-client-id.apps.googleusercontent.com]
  apple:
    client_ids: [com.example.app, com.example.web]
```

Environment variables use Viper's nested-key convention. For deployment,
prefer a mounted configuration file for the client ID arrays.

## Public login APIs

- `POST /api/auth/google`
- `POST /api/auth/apple`

Google request example:

```json
{
  "id_token": "provider-signed-jwt",
  "phone_code": "stable-device-id",
  "nonce": "optional-request-nonce",
  "app_name": "AI_VIDEO",
  "app_version": "1.0.0",
  "device_country": "CN"
}
```

Apple accepts either `id_token` or `identity_token`. Because Apple only returns
the user's name on the first authorization, the client may also send
`display_name`, `given_name`, and `family_name` on that first request.

The response is the same local JWT/profile payload returned by device login.
An unregistered guest on the same device is upgraded in place; future logins on
another device resolve the same local user through `video_user_identity`.

## Authenticated account-link APIs

- `GET /api/users/me/identities`
- `POST /api/users/me/identities/google`
- `POST /api/users/me/identities/apple`
- `DELETE /api/users/me/identities/{google|apple}`

Bind requests use the same provider token/name fields as public login. A
provider subject cannot belong to two local users, and a local user can bind at
most one identity per provider. The final third-party login method cannot be
removed, preventing an account from becoming inaccessible.

## Token checks

The backend requires RS256 signatures and validates `kid`, issuer, audience,
authorized party (`azp`, when supplied), expiration, issued-at time, subject,
and the nonce when the client supplies one. Verified provider email is stored as
profile data when it does not conflict with another local user; matching email
alone never merges accounts.
