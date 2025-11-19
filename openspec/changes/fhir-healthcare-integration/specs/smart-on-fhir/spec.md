# Specification: SMART on FHIR Authorization

## Overview

OAuth2-based authorization framework for FHIR applications following the SMART App Launch 2.0 specification. Supports
EHR launch, standalone launch, and backend services (system-to-system) authorization flows with comprehensive token
management.

## ADDED Requirements

### Requirement: Discover SMART authorization endpoints

Automatically discover OAuth2 authorization and token URLs from FHIR server CapabilityStatement.

#### Scenario: Discover endpoints from CapabilityStatement

**Given** a FHIR server at `https://fhir.example.com` with CapabilityStatement:
```json
{
  "rest": [{
    "security": {
      "extension": [{
        "url": "http://fhir-registry.smarthealthit.org/StructureDefinition/oauth-uris",
        "extension": [
          {"url": "authorize", "valueUri": "https://auth.example.com/authorize"},
          {"url": "token", "valueUri": "https://auth.example.com/token"}
        ]
      }]
    }
  }]
}
```

**When** creating a SMART client:
```go
client, err := smart.NewClient(smart.Config{
    FHIRBaseURL: "https://fhir.example.com",
    ClientID:    "my-app",
})
```

**Then** the client:
- Fetches CapabilityStatement from `https://fhir.example.com/metadata`
- Extracts authorization URLs from extensions
- Sets `client.AuthorizeURL` = `"https://auth.example.com/authorize"`
- Sets `client.TokenURL` = `"https://auth.example.com/token"`

#### Scenario: Skip discovery with explicit configuration

**Given** explicit authorization URLs in config

**When** creating client:
```go
client, err := smart.NewClient(smart.Config{
    FHIRBaseURL:   "https://fhir.example.com",
    AuthorizeURL:  "https://auth.example.com/authorize",
    TokenURL:      "https://auth.example.com/token",
    ClientID:      "my-app",
    SkipDiscovery: true,
})
```

**Then** the client:
- Skips CapabilityStatement fetch
- Uses provided URLs directly
- No network call during initialization

#### Scenario: Handle missing SMART extensions in CapabilityStatement

**Given** a FHIR server with CapabilityStatement lacking SMART extensions

**When** creating client without explicit URLs

**Then** the client returns error:
- Error = `ErrSMARTNotSupported`
- Error message: `"FHIR server does not support SMART authorization (missing oauth-uris extension)"`

---

### Requirement: Perform EHR Launch flow

Launch app from within EHR system with patient/encounter context.

#### Scenario: Handle EHR launch with patient context

**Given** an EHR launches app with parameters:
- `iss` = `"https://fhir.example.com"`
- `launch` = `"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`

**When** app calls:
```go
ctx := smart.NewLaunchContext("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
authURL, err := client.GetAuthorizationURL(ctx, []string{
    "launch",
    "patient/*.read",
    "openid", "fhirUser",
})
```

**Then** the function returns URL with:
- Base: `"https://auth.example.com/authorize"`
- Query parameters:
  - `response_type=code`
  - `client_id=my-app`
  - `redirect_uri=http://localhost:8080/callback`
  - `scope=launch+patient/*.read+openid+fhirUser`
  - `state=<random-state>`
  - `launch=eyJhbGci...` (launch token passed through)
  - `aud=https://fhir.example.com`

#### Scenario: Exchange authorization code for access token

**Given** authorization server redirects back with:
- `code` = `"auth-code-12345"`
- `state` = `"<matching-state>"`

**When** app calls:
```go
token, err := client.ExchangeCode(ctx, "auth-code-12345", "stored-state")
```

**Then** the client:
- POSTs to token endpoint with `grant_type=authorization_code`
- Validates state matches stored value
- Returns token with:
  - `AccessToken` (JWT)
  - `RefreshToken` (if provided)
  - `ExpiresIn` (seconds)
  - `Scope` (granted scopes)
  - `PatientID` (from `patient` claim if present)

#### Scenario: Resolve launch context from token

**Given** an access token with patient context

**When** calling:
```go
context, err := client.ResolveContext(token)
```

**Then** the context contains:
- `PatientID` = `"Patient/123"`
- `EncounterID` = `nil` (if not in token)
- `UserID` = `"Practitioner/456"` (from `fhirUser` claim)
- `Scope` = `["patient/*.read", "openid", "fhirUser"]`

---

### Requirement: Perform standalone launch flow

Launch app independently with user login and patient selection.

#### Scenario: Standalone launch without launch token

**Given** app starts independently (not from EHR)

**When** building authorization URL:
```go
authURL, err := client.GetAuthorizationURL(nil, []string{
    "patient/*.read",
    "user/*.write",
    "openid", "fhirUser",
})
```

**Then** the URL includes:
- `response_type=code`
- `client_id=my-app`
- `redirect_uri=http://localhost:8080/callback`
- `scope=patient/*.read+user/*.write+openid+fhirUser`
- `state=<random-state>`
- `aud=https://fhir.example.com`
- NO `launch` parameter (standalone mode)

#### Scenario: Select patient during standalone authorization

**Given** user authorizes app

**When** authorization server redirects with code

**Then** token exchange returns access token where:
- `patient` claim contains selected patient ID
- `scope` reflects granted permissions (may differ from requested)
- App can now access selected patient's data

---

### Requirement: Implement PKCE for public clients

Use Proof Key for Code Exchange to prevent authorization code interception attacks.

#### Scenario: Generate PKCE parameters for authorization

**Given** a public client (no client secret)

**When** building authorization URL with PKCE:
```go
verifier, challenge, err := smart.GeneratePKCE()
authURL, err := client.GetAuthorizationURLWithPKCE(nil, scopes, challenge)
```

**Then** the authorization URL includes:
- `code_challenge=<base64url-encoded-sha256(verifier)>`
- `code_challenge_method=S256`
- Client stores `verifier` securely for token exchange

#### Scenario: Exchange code with PKCE verifier

**Given** authorization code and stored verifier

**When** exchanging code:
```go
token, err := client.ExchangeCodeWithPKCE(ctx, code, state, verifier)
```

**Then** the token request includes:
- `code_verifier=<original-verifier>`
- Authorization server validates verifier matches challenge
- Returns access token if valid

---

### Requirement: Implement Backend Services authorization

System-to-system authorization using JWT assertions for bulk data access and automated workflows.

#### Scenario: Authenticate with RS384 JWT assertion

**Given** a backend service with:
- RSA private key for signing
- Client ID registered with authorization server

**When** authenticating:
```go
jwt := smart.CreateJWTAssertion(smart.JWTClaims{
    Issuer:   "my-backend-service",
    Subject:  "my-backend-service",
    Audience: "https://auth.example.com/token",
    Expiry:   time.Now().Add(5 * time.Minute),
    JTI:      uuid.New().String(),
})

token, err := client.BackendServicesAuth(ctx, jwt)
```

**Then** the client:
- POSTs to token endpoint with:
  - `grant_type=client_credentials`
  - `client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer`
  - `client_assertion=<signed-jwt>`
  - `scope=system/*.read`
- Returns access token valid for system-level access

#### Scenario: Request system-level scopes

**Given** backend service with system access

**When** requesting token with:
```go
token, err := client.BackendServicesAuth(ctx, jwt, []string{
    "system/Patient.read",
    "system/Observation.read",
    "system/ImagingStudy.read",
})
```

**Then** the access token:
- Grants system-level access (not patient-specific)
- Can read all patients' data
- Scope reflects granted permissions

---

### Requirement: Manage token lifecycle

Automatically refresh access tokens before expiry and cache tokens in memory.

#### Scenario: Automatically refresh expired token

**Given** an access token with:
- `ExpiresIn` = `3600` (1 hour)
- `RefreshToken` = `"refresh-token-xyz"`
- Token issued 55 minutes ago

**When** app makes FHIR request:
```go
patient, err := client.GetResource(ctx, "Patient/123")
```

**Then** the client:
- Detects token expires in <5 minutes
- Refreshes token using refresh token
- POSTs to token endpoint with `grant_type=refresh_token`
- Receives new access token
- Caches new token
- Retries original request with new token

#### Scenario: Cache access token to avoid repeated exchanges

**Given** a valid access token stored in memory

**When** multiple requests use the same client:
```go
patient1, _ := client.GetResource(ctx, "Patient/123")
patient2, _ := client.GetResource(ctx, "Patient/456")
```

**Then** the client:
- Uses cached access token for both requests
- Does not exchange authorization code twice
- No unnecessary token endpoint calls

#### Scenario: Handle refresh token expiry gracefully

**Given** a refresh token that has expired

**When** attempting to refresh access token

**Then** the client:
- Returns error: `ErrRefreshTokenExpired`
- Clears cached tokens
- App must re-authorize user (new authorization flow)

---

### Requirement: Validate and parse access token scopes

Enforce scope-based access control for fine-grained permissions.

#### Scenario: Parse granted scopes from token

**Given** an access token with scopes: `"patient/Patient.read patient/Observation.read launch"`

**When** parsing token:
```go
scopes := smart.ParseScopes(token.Scope)
```

**Then** scopes contains:
- `["patient/Patient.read", "patient/Observation.read", "launch"]`

#### Scenario: Check if token has required scope

**Given** a token with `patient/Patient.read`

**When** checking permission:
```go
hasPermission := smart.HasScope(token, "patient/Patient.read")
```

**Then** `hasPermission` = `true`

#### Scenario: Reject insufficient scope

**Given** a token with only `patient/Patient.read`

**When** checking for write permission:
```go
hasPermission := smart.HasScope(token, "patient/Patient.write")
```

**Then** `hasPermission` = `false`

#### Scenario: Validate wildcard scopes

**Given** a token with `patient/*.read`

**When** checking:
```go
hasPermission := smart.HasScope(token, "patient/Observation.read")
```

**Then** `hasPermission` = `true` (wildcard matches)

---

### Requirement: Support SMART capabilities discovery

Detect which SMART features the authorization server supports.

#### Scenario: Detect supported SMART capabilities

**Given** a CapabilityStatement with:
```json
{
  "rest": [{
    "security": {
      "service": [{
        "coding": [{
          "system": "http://terminology.hl7.org/CodeSystem/restful-security-service",
          "code": "SMART-on-FHIR"
        }]
      }],
      "extension": [{
        "url": "http://fhir-registry.smarthealthit.org/StructureDefinition/capabilities",
        "valueCode": "launch-ehr"
      }, {
        "url": "http://fhir-registry.smarthealthit.org/StructureDefinition/capabilities",
        "valueCode": "client-confidential-symmetric"
      }]
    }
  }]
}
```

**When** querying capabilities:
```go
caps, err := client.GetCapabilities()
```

**Then** capabilities include:
- `"launch-ehr"` (supports EHR launch)
- `"client-confidential-symmetric"` (supports client secrets)
- NOT `"launch-standalone"` (not in list)

---

### Requirement: Handle authorization errors

Gracefully handle OAuth2 errors from authorization server.

#### Scenario: Handle user denies authorization

**Given** user clicks "Deny" during authorization

**When** authorization server redirects with:
- `error=access_denied`
- `error_description=User denied authorization`

**Then** client returns:
- Error = `ErrAuthorizationDenied`
- Error message includes description

#### Scenario: Handle invalid client credentials

**Given** client with incorrect client secret

**When** exchanging authorization code

**Then** authorization server returns:
- HTTP 401 Unauthorized
- `error=invalid_client`

**And** client returns:
- Error = `ErrInvalidClient`
- Error message: `"Invalid client credentials"`

#### Scenario: Handle invalid scope request

**Given** client requests unsupported scope: `"invalid-scope"`

**When** authorization request is made

**Then** authorization server returns:
- `error=invalid_scope`
- `error_description=Scope 'invalid-scope' not supported`

**And** client returns:
- Error = `ErrInvalidScope`
- Error message includes description

---

## MODIFIED Requirements

None (new capability)

---

## REMOVED Requirements

None (new capability)

---

## Cross-References

- **Required by**: `fhir-subscriptions` spec - Subscriptions may require authenticated access
- **Enables**: Bulk FHIR export (requires Backend Services authorization)
- **Related to**: `us-core-ig` spec - US Core profiles accessed via SMART-authorized requests

---

## Implementation Notes

### Package Structure
```
fhir/smart/
├── client.go             # Main SMART client
├── config.go             # Client configuration
├── capabilities.go       # SMART capability discovery
├── errors.go             # SMART-specific errors
├── auth/
│   ├── oauth2.go         # OAuth2 authorization code flow
│   ├── backend.go        # Backend Services (JWT assertions)
│   ├── token_manager.go  # Token refresh and caching
│   └── pkce.go           # PKCE implementation
├── launch/
│   ├── ehr.go            # EHR launch flow
│   ├── standalone.go     # Standalone launch flow
│   └── context.go        # Launch context resolution
└── scopes.go             # Scope parsing and validation
```

### Key Types
```go
type Config struct {
    FHIRBaseURL   string
    AuthorizeURL  string  // Optional, auto-discovered if not provided
    TokenURL      string  // Optional, auto-discovered if not provided
    ClientID      string
    ClientSecret  string  // Optional for public clients
    RedirectURI   string
    SkipDiscovery bool
}

type Client struct {
    config       *Config
    tokenManager *TokenManager
    httpClient   *http.Client
}

type Token struct {
    AccessToken  string
    TokenType    string
    ExpiresIn    int
    RefreshToken string
    Scope        string
    PatientID    string
    UserID       string
}

type LaunchContext struct {
    LaunchToken  string
    PatientID    string
    EncounterID  string
    UserID       string
    Scope        []string
}

var (
    ErrSMARTNotSupported     = errors.New("FHIR server does not support SMART authorization")
    ErrAuthorizationDenied   = errors.New("user denied authorization")
    ErrInvalidClient         = errors.New("invalid client credentials")
    ErrInvalidScope          = errors.New("invalid scope")
    ErrRefreshTokenExpired   = errors.New("refresh token expired")
    ErrInvalidState          = errors.New("state parameter mismatch")
)
```

### Testing Approach
- Mock authorization server for unit tests
- [SMART Launcher](https://launch.smarthealthit.org/) conformance tests
- Token refresh simulation (expired tokens)
- PKCE code verifier/challenge validation
- Backend Services JWT signature verification
- Concurrent token access (race conditions)

### Dependencies
- `golang.org/x/oauth2` - OAuth2 client library (Apache-2.0)
- `github.com/google/uuid` - UUID generation for state/JTI (BSD-3-Clause)
- `github.com/golang-jwt/jwt/v5` - JWT parsing for Backend Services (MIT, optional)

### Security Considerations
- Always use HTTPS for authorization endpoints
- Mandate PKCE for public clients (mobile, SPA)
- Validate state parameter to prevent CSRF
- Secure token storage (never log access tokens)
- Rotate refresh tokens on each use
- Implement token expiry checks
- Use constant-time string comparison for secrets