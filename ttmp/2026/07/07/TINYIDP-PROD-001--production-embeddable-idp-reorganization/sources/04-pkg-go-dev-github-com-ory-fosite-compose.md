## Documentation

### Index

### Constants

This section is empty.

### Variables

This section is empty.

### Functions

#### func Compose ¶

```
func Compose(config *fosite.Config, storage interface{}, strategy interface{}, factories 
...Factory) fosite.OAuth2Provider
```

Compose takes a config, a storage, a strategy and handlers to instantiate an OAuth2Provider:

```
import "github.com/ory/fosite/compose"

// var storage = new(MyFositeStorage)
var config = Config {
    AccessTokenLifespan: time.Minute * 30,
       // check Config for further configuration options
}

var strategy = NewOAuth2HMACStrategy(config)

var oauth2Provider = Compose(
    config,
       storage,
       strategy,
       NewOAuth2AuthorizeExplicitHandler,
       OAuth2ClientCredentialsGrantFactory,
       // for a complete list refer to the docs of this package
)
```

Compose makes use of interface{} types in order to be able to handle a all types of stores, 
strategies and handlers.

#### func ComposeAllEnabled ¶

```
func ComposeAllEnabled(config *fosite.Config, storage interface{}, key interface{}) 
fosite.OAuth2Provider
```

ComposeAllEnabled returns a fosite instance with all OAuth2 and OpenID Connect handlers enabled.

#### func NewOAuth2HMACStrategy ¶

```
func NewOAuth2HMACStrategy(config HMACSHAStrategyConfigurator) *oauth2.HMACSHAStrategy
```

#### func NewOAuth2JWTStrategy ¶

```
func NewOAuth2JWTStrategy(keyGetter func(context.Context) (interface{}, error), strategy 
oauth2.CoreStrategy, config fosite.Configurator) *oauth2.DefaultJWTStrategy
```

#### func NewOpenIDConnectStrategy ¶

```
func NewOpenIDConnectStrategy(keyGetter func(context.Context) (interface{}, error), config 
fosite.Configurator) *openid.DefaultStrategy
```

#### func OAuth2AuthorizeExplicitFactory ¶

```
func OAuth2AuthorizeExplicitFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OAuth2AuthorizeExplicitFactory creates an OAuth2 authorize code grant ("authorize explicit flow") 
handler and registers an access token, refresh token and authorize code validator.

#### func OAuth2AuthorizeImplicitFactory ¶

```
func OAuth2AuthorizeImplicitFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OAuth2AuthorizeImplicitFactory creates an OAuth2 implicit grant ("authorize implicit flow") handler 
and registers an access token, refresh token and authorize code validator.

#### func OAuth2ClientCredentialsGrantFactory ¶

```
func OAuth2ClientCredentialsGrantFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OAuth2ClientCredentialsGrantFactory creates an OAuth2 client credentials grant handler and 
registers an access token, refresh token and authorize code validator.

#### added in v0.16.4

```
func OAuth2PKCEFactory(config fosite.Configurator, storage interface{}, strategy interface{}) 
interface{}
```

OAuth2PKCEFactory creates a PKCE handler.

#### func OAuth2RefreshTokenGrantFactory ¶

```
func OAuth2RefreshTokenGrantFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OAuth2RefreshTokenGrantFactory creates an OAuth2 refresh grant handler and registers an access 
token, refresh token and authorize code validator.nmj

#### func OAuth2ResourceOwnerPasswordCredentialsFactory deprecated

```
func OAuth2ResourceOwnerPasswordCredentialsFactory(config fosite.Configurator, storage interface{}, 
strategy interface{}) interface{}
```

OAuth2ResourceOwnerPasswordCredentialsFactory creates an OAuth2 resource owner password credentials 
grant handler and registers an access token, refresh token and authorize code validator.

Deprecated: This factory is deprecated as a means to communicate that the ROPC grant type is widely 
discouraged and is at the time of this writing going to be omitted in the OAuth 2.1 spec. For more 
information on why this grant type is discouraged see: 
[https://www.scottbrady91.com/oauth/why-the-resource-owner-password-credentials-grant-type-is-not-au
thentication-nor-suitable-for-modern-applications](https://www.scottbrady91.com/oauth/why-the-resour
ce-owner-password-credentials-grant-type-is-not-authentication-nor-suitable-for-modern-applications)

#### added in v0.6.17

```
func OAuth2StatelessJWTIntrospectionFactory(config fosite.Configurator, storage interface{}, 
strategy interface{}) interface{}
```

OAuth2StatelessJWTIntrospectionFactory creates an OAuth2 token introspection handler and registers 
an access token validator. This can only be used to validate JWTs and does so statelessly, meaning 
it uses only the data available in the JWT itself, and does not access the storage implementation 
at all.

Due to the stateless nature of this factory, THE BUILT-IN REVOCATION MECHANISMS WILL NOT WORK. If 
you need revocation, you can validate JWTs statefully, using the other factories.

#### added in v0.5.0

```
func OAuth2TokenIntrospectionFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OAuth2TokenIntrospectionFactory creates an OAuth2 token introspection handler and registers an 
access token and refresh token validator.

#### added in v0.4.0

```
func OAuth2TokenRevocationFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OAuth2TokenRevocationFactory creates an OAuth2 token revocation handler.

#### added in v0.45.0

```
func OIDCUserinfoVerifiableCredentialFactory(config fosite.Configurator, storage, strategy any) any
```

OIDCUserinfoVerifiableCredentialFactory creates a verifiable credentials handler.

#### added in v0.5.0

```
func OpenIDConnectExplicitFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OpenIDConnectExplicitFactory creates an OpenID Connect explicit ("authorize code flow") grant 
handler.

\*\*Important note:\*\* You must add this handler \*after\* you have added an OAuth2 authorize code 
handler!

#### added in v0.5.0

```
func OpenIDConnectHybridFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OpenIDConnectHybridFactory creates an OpenID Connect hybrid grant handler.

\*\*Important note:\*\* You must add this handler \*after\* you have added an OAuth2 authorize code 
handler!

#### added in v0.5.0

```
func OpenIDConnectImplicitFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

OpenIDConnectImplicitFactory creates an OpenID Connect implicit ("implicit flow") grant handler.

\*\*Important note:\*\* You must add this handler \*after\* you have added an OAuth2 authorize code 
handler!

#### added in v0.11.0

```
func OpenIDConnectRefreshFactory(config fosite.Configurator, _ interface{}, strategy interface{}) 
interface{}
```

OpenIDConnectRefreshFactory creates a handler for refreshing openid connect tokens.

\*\*Important note:\*\* You must add this handler \*after\* you have added an OAuth2 authorize code 
handler!

#### added in v0.43.0

```
func PushedAuthorizeHandlerFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

PushedAuthorizeHandlerFactory creates the basic PAR handler

#### added in v0.37.0

```
func RFC7523AssertionGrantFactory(config fosite.Configurator, storage interface{}, strategy 
interface{}) interface{}
```

RFC7523AssertionGrantFactory creates an OAuth2 Authorize JWT Grant (using JWTs as Authorization 
Grants) handler and registers an access token, refresh token and authorize code validator.

### Types

#### type CommonStrategy ¶

```
type CommonStrategy struct {
    oauth2.CoreStrategy
    openid.OpenIDConnectTokenStrategy
    jwt.Signer
}
```

#### added in v0.6.17

```
type Factory func(config fosite.Configurator, storage interface{}, strategy interface{}) interface{}
```

#### added in v0.43.0

```
type HMACSHAStrategyConfigurator interface {
    fosite.AccessTokenLifespanProvider
    fosite.RefreshTokenLifespanProvider
    fosite.AuthorizeCodeLifespanProvider
    fosite.TokenEntropyProvider
    fosite.GlobalSecretProvider
    fosite.RotatedGlobalSecretsProvider
    fosite.HMACHashingProvider
}
```
