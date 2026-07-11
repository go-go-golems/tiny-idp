## Documentation

### Index

### Constants

This section is empty.

### Variables

This section is empty.

### Functions

#### added in v0.29.0

```
func MaybeBeginTx(ctx context.Context, storage interface{}) (context.Context, error)
```

MaybeBeginTx is a helper function that can be used to initiate a transaction if the supplied storage implements the \`Transactional\` interface.

#### added in v0.29.0

```
func MaybeCommitTx(ctx context.Context, storage interface{}) error
```

MaybeCommitTx is a helper function that can be used to commit a transaction if the supplied storage implements the \`Transactional\` interface.

#### added in v0.29.0

```
func MaybeRollbackTx(ctx context.Context, storage interface{}) error
```

MaybeRollbackTx is a helper function that can be used to rollback a transaction if the supplied storage implements the \`Transactional\` interface.

### Types

#### added in v0.37.0

```
type IssuerPublicKeys struct {
    Issuer    string
    KeysBySub map[string]SubjectPublicKeys
}
```

#### type MemoryStore ¶

```
type MemoryStore struct {
    Clients         map[string]fosite.Client
    AuthorizeCodes  map[string]StoreAuthorizeCode
    IDSessions      map[string]fosite.Requester
    AccessTokens    map[string]fosite.Requester
    RefreshTokens   map[string]StoreRefreshToken
    PKCES           map[string]fosite.Requester
    Users           map[string]MemoryUserRelation
    BlacklistedJTIs map[string]time.Time
    // In-memory request ID to token signatures
    AccessTokenRequestIDs  map[string]string
    RefreshTokenRequestIDs map[string]string
    // Public keys to check signature in auth grant jwt assertion.
    IssuerPublicKeys map[string]IssuerPublicKeys
    PARSessions      map[string]fosite.AuthorizeRequester
    // contains filtered or unexported fields
}
```

#### func NewExampleStore ¶

```
func NewExampleStore() *MemoryStore
```

#### func NewMemoryStore ¶

```
func NewMemoryStore() *MemoryStore
```

#### func (\*MemoryStore) Authenticate ¶

```
func (s *MemoryStore) Authenticate(_ context.Context, name string, secret string) (subject string, err error)
```

#### added in v0.31.0

```
func (s *MemoryStore) ClientAssertionJWTValid(_ context.Context, jti string) error
```

#### func (\*MemoryStore) CreateAccessTokenSession ¶

```
func (s *MemoryStore) CreateAccessTokenSession(_ context.Context, signature string, req fosite.Requester) error
```

#### func (\*MemoryStore) CreateAuthorizeCodeSession ¶

```
func (s *MemoryStore) CreateAuthorizeCodeSession(_ context.Context, code string, req fosite.Requester) error
```

#### func (\*MemoryStore) CreateOpenIDConnectSession ¶

```
func (s *MemoryStore) CreateOpenIDConnectSession(_ context.Context, authorizeCode string, requester fosite.Requester) error
```

#### added in v0.43.0

```
func (s *MemoryStore) CreatePARSession(ctx context.Context, requestURI string, request fosite.AuthorizeRequester) error
```

CreatePARSession stores the pushed authorization request context. The requestURI is used to derive the key.

#### added in v0.17.0

```
func (s *MemoryStore) CreatePKCERequestSession(_ context.Context, code string, req fosite.Requester) error
```

#### func (\*MemoryStore) CreateRefreshTokenSession ¶

```
func (s *MemoryStore) CreateRefreshTokenSession(_ context.Context, signature, accessTokenSignature string, req fosite.Requester) error
```

#### func (\*MemoryStore) DeleteAccessTokenSession ¶

```
func (s *MemoryStore) DeleteAccessTokenSession(_ context.Context, signature string) error
```

#### func (\*MemoryStore) DeleteOpenIDConnectSession ¶

```
func (s *MemoryStore) DeleteOpenIDConnectSession(_ context.Context, authorizeCode string) error
```

#### added in v0.43.0

```
func (s *MemoryStore) DeletePARSession(ctx context.Context, requestURI string) (err error)
```

DeletePARSession deletes the context.

#### added in v0.17.0

```
func (s *MemoryStore) DeletePKCERequestSession(_ context.Context, code string) error
```

#### func (\*MemoryStore) DeleteRefreshTokenSession ¶

```
func (s *MemoryStore) DeleteRefreshTokenSession(_ context.Context, signature string) error
```

#### func (\*MemoryStore) GetAccessTokenSession ¶

```
func (s *MemoryStore) GetAccessTokenSession(_ context.Context, signature string, _ fosite.Session) (fosite.Requester, error)
```

#### func (\*MemoryStore) GetAuthorizeCodeSession ¶

```
func (s *MemoryStore) GetAuthorizeCodeSession(_ context.Context, code string, _ fosite.Session) (fosite.Requester, error)
```

#### func (\*MemoryStore) GetClient ¶

```
func (s *MemoryStore) GetClient(_ context.Context, id string) (fosite.Client, error)
```

#### func (\*MemoryStore) GetOpenIDConnectSession ¶

```
func (s *MemoryStore) GetOpenIDConnectSession(_ context.Context, authorizeCode string, requester fosite.Requester) (fosite.Requester, error)
```

#### added in v0.43.0

```
func (s *MemoryStore) GetPARSession(ctx context.Context, requestURI string) (fosite.AuthorizeRequester, error)
```

GetPARSession gets the push authorization request context. If the request is nil, a new request object is created. Otherwise, the same object is updated.

#### added in v0.17.0

```
func (s *MemoryStore) GetPKCERequestSession(_ context.Context, code string, _ fosite.Session) (fosite.Requester, error)
```

#### added in v0.37.0

```
func (s *MemoryStore) GetPublicKey(ctx context.Context, issuer string, subject string, keyId string) (*jose.JSONWebKey, error)
```

#### added in v0.37.0

```
func (s *MemoryStore) GetPublicKeyScopes(ctx context.Context, issuer string, subject string, keyId string) ([]string, error)
```

#### added in v0.37.0

```
func (s *MemoryStore) GetPublicKeys(ctx context.Context, issuer string, subject string) (*jose.JSONWebKeySet, error)
```

#### func (\*MemoryStore) GetRefreshTokenSession ¶

```
func (s *MemoryStore) GetRefreshTokenSession(_ context.Context, signature string, _ fosite.Session) (fosite.Requester, error)
```

#### added in v0.20.0

```
func (s *MemoryStore) InvalidateAuthorizeCodeSession(ctx context.Context, code string) error
```

#### added in v0.37.0

```
func (s *MemoryStore) IsJWTUsed(ctx context.Context, jti string) (bool, error)
```

#### added in v0.37.0

```
func (s *MemoryStore) MarkJWTUsedForTime(ctx context.Context, jti string, exp time.Time) error
```

#### func (\*MemoryStore) RevokeAccessToken ¶

```
func (s *MemoryStore) RevokeAccessToken(ctx context.Context, requestID string) error
```

#### func (\*MemoryStore) RevokeRefreshToken ¶

```
func (s *MemoryStore) RevokeRefreshToken(ctx context.Context, requestID string) error
```

#### added in v0.49.0

```
func (s *MemoryStore) RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) (err error)
```

#### added in v0.31.0

```
func (s *MemoryStore) SetClientAssertionJWT(_ context.Context, jti string, exp time.Time) error
```

#### added in v0.43.0

```
func (s *MemoryStore) SetTokenLifespans(clientID string, lifespans *fosite.ClientLifespanConfig) error
```

#### type MemoryUserRelation ¶

```
type MemoryUserRelation struct {
    Username string
    Password string
}
```

#### added in v0.37.0

```
type PublicKeyScopes struct {
    Key    *jose.JSONWebKey
    Scopes []string
}
```

#### added in v0.20.0

```
type StoreAuthorizeCode struct {
    fosite.Requester
    // contains filtered or unexported fields
}
```

#### added in v0.39.0

```
type StoreRefreshToken struct {
    fosite.Requester
    // contains filtered or unexported fields
}
```

#### added in v0.37.0

```
type SubjectPublicKeys struct {
    Subject string
    Keys    map[string]PublicKeyScopes
}
```

#### added in v0.29.0

```
type Transactional interface {
    BeginTX(ctx context.Context) (context.Context, error)
    Commit(ctx context.Context) error
    Rollback(ctx context.Context) error
}
```

A storage provider that has support for transactions should implement this interface to ensure atomicity for certain flows that require transactional semantics. Fosite will call these methods (when atomicity is required) if and only if the storage provider has implemented \`Transactional\`. It is expected that the storage provider will examine context for an existing transaction each time a database operation is to be performed.

An implementation of \`BeginTX\` should attempt to initiate a new transaction and store that under a unique key in the context that can be accessible by \`Commit\` and \`Rollback\`. The "transactional aware" context will then be returned for further propagation, eventually to be consumed by \`Commit\` or \`Rollback\` to finish the transaction.

Implementations for \`Commit\` & \`Rollback\` should look for the transaction object inside the supplied context using the same key used by \`BeginTX\`. If these methods have been called, it is expected that a txn object should be available in the provided context.