# Specification: FHIR Subscriptions

## Overview

Real-time event notification system for FHIR resource changes using the FHIR R5 Subscriptions specification. Supports
topic-based subscriptions, webhook delivery with retry logic, flexible filters, and R4 backport compatibility for
event-driven healthcare workflows.

## ADDED Requirements

### Requirement: Create and manage subscriptions

Handle subscription lifecycle including creation, activation, and deletion.

#### Scenario: Create active subscription for patient updates

**Given** a subscription definition:
```json
{
  "resourceType": "Subscription",
  "status": "active",
  "topic": "http://example.org/fhir/SubscriptionTopic/patient-update",
  "endpoint": "https://app.example.com/notify",
  "channelType": {
    "code": "rest-hook"
  },
  "contentType": "application/fhir+json"
}
```

**When** creating subscription:
```go
manager := subscriptions.NewManager(store, notifier)
err := manager.Subscribe(ctx, subscription)
```

**Then** the manager:
- Validates subscription structure
- Stores subscription in persistence layer
- Sets `status` = `"active"`
- Returns subscription ID for tracking

#### Scenario: Activate requested subscription

**Given** a subscription with `status` = `"requested"`

**When** activating:
```go
err := manager.ActivateSubscription(ctx, subscriptionID)
```

**Then** the subscription:
- Changes `status` to `"active"`
- Begins monitoring for matching events
- Starts webhook delivery

#### Scenario: Deactivate subscription

**Given** an active subscription

**When** deactivating:
```go
err := manager.DeactivateSubscription(ctx, subscriptionID)
```

**Then** the subscription:
- Changes `status` to `"off"`
- Stops monitoring for events
- Stops webhook deliveries
- Remains in storage (can be reactivated)

#### Scenario: Delete subscription

**Given** a subscription (any status)

**When** deleting:
```go
err := manager.DeleteSubscription(ctx, subscriptionID)
```

**Then** the subscription:
- Is removed from storage
- All deliveries cease immediately
- Returns HTTP 204 No Content if successful

---

### Requirement: Monitor resource changes and trigger notifications

Detect resource create/update/delete events and match against active subscriptions.

#### Scenario: Trigger notification on patient update

**Given** an active subscription with:
- `topic` = `"patient-update"`
- `filter` = `"Patient?_tag=test"`

**When** a Patient resource with tag `"test"` is updated:
```go
event := subscriptions.ResourceEvent{
    EventType: "update",
    Resource:  &updatedPatient,
}
err := manager.NotifyChange(ctx, event)
```

**Then** the manager:
- Finds matching subscriptions (1 match)
- Builds notification bundle (history type)
- Triggers webhook delivery
- Logs notification event

#### Scenario: Filter events by resource type

**Given** subscriptions for:
- Subscription A: topic `"patient-update"`
- Subscription B: topic `"observation-create"`

**When** an Observation is created

**Then** the manager:
- Matches Subscription B only
- Does NOT trigger Subscription A
- Delivers 1 notification (to Subscription B webhook)

#### Scenario: Handle no matching subscriptions

**Given** no active subscriptions for Encounter resources

**When** an Encounter is created

**Then** the manager:
- Finds zero matching subscriptions
- Does not trigger any webhooks
- Logs event for debugging
- Returns without error

---

### Requirement: Deliver webhook notifications with retry

POST notification bundles to subscriber endpoints with exponential backoff retry.

#### Scenario: Successful webhook delivery

**Given** a subscription with endpoint `"https://app.example.com/notify"`

**When** delivering notification:
```go
bundle := buildNotificationBundle(event, subscription)
err := manager.DeliverWebhook(ctx, subscription, bundle)
```

**Then** the manager:
- POSTs bundle to `"https://app.example.com/notify"`
- Sets `Content-Type: application/fhir+json`
- Receives HTTP 200 OK
- Logs successful delivery
- Returns no error

#### Scenario: Retry failed webhook with exponential backoff

**Given** a subscription webhook that initially fails

**When** delivering notification and webhook returns HTTP 503

**Then** the manager:
- Retries delivery with delays: 1s, 2s, 4s
- Uses exponential backoff
- Maximum 3 retry attempts
- If all retries fail, logs to dead letter queue
- Returns error after max retries exceeded

#### Scenario: Handle webhook timeout

**Given** a webhook endpoint that times out (no response)

**When** delivering notification with 5-second timeout

**Then** the manager:
- Cancels request after 5 seconds
- Logs timeout error
- Retries per retry policy
- Eventual failure after max retries

#### Scenario: Handle invalid webhook URL

**Given** a subscription with malformed endpoint URL

**When** attempting delivery

**Then** the manager:
- Validates URL before HTTP request
- Returns error immediately: `ErrInvalidWebhookURL`
- Does not retry (configuration error)
- Logs validation failure

---

### Requirement: Build notification bundles

Create FHIR Bundles containing notification metadata and resource content.

#### Scenario: Build history bundle for patient update

**Given** a patient update event with before/after state

**When** building notification bundle:
```go
bundle := subscriptions.BuildNotificationBundle(event, subscription)
```

**Then** the bundle contains:
- `type` = `"history"`
- `entry[0]`: SubscriptionStatus with notification event info
- `entry[1]`: Previous patient state (if available)
- `entry[2]`: Current patient state
- `timestamp` = current time
- `id` = unique notification ID

#### Scenario: Build bundle with full resource content

**Given** subscription with `content` = `"full-resource"`

**When** building notification bundle

**Then** the bundle includes:
- Complete resource in `entry[*].resource`
- All fields populated

#### Scenario: Build bundle with id-only content

**Given** subscription with `content` = `"id-only"`

**When** building notification bundle

**Then** the bundle includes:
- `entry[*].fullUrl` with resource URL
- `entry[*].resource` is empty (no content)
- Subscriber must fetch resource separately

#### Scenario: Include notification event metadata

**Given** any notification event

**When** building bundle

**Then** `entry[0]` (SubscriptionStatus) contains:
- `type` = `"event-notification"`
- `subscription.reference` = subscription ID
- `topic` = topic URL
- `eventsSinceSubscriptionStart` = event count
- `notificationEvent[0].eventNumber` = sequence number
- `notificationEvent[0].focus.reference` = changed resource

---

### Requirement: Apply subscription filters

Filter events based on FHIR search parameters.

#### Scenario: Filter by resource identifier

**Given** a subscription with filter:
- `"Patient?identifier=http://example.org/mrn|12345"`

**When** processing patient update for:
- Patient A: identifier `"http://example.org/mrn|12345"` ✓
- Patient B: identifier `"http://example.org/mrn|67890"` ✗

**Then** the manager:
- Delivers notification for Patient A
- Does NOT deliver for Patient B

#### Scenario: Filter by tag

**Given** a subscription with filter:
- `"Patient?_tag=http://example.org/tags|urgent"`

**When** processing patient with tags:
- Patient A: has `"urgent"` tag ✓
- Patient B: no tags ✗

**Then** notification delivered only for Patient A

#### Scenario: Filter by date range

**Given** a subscription with filter:
- `"Observation?date=ge2024-01-01"`

**When** processing observations:
- Observation A: `effectiveDateTime` = `"2024-06-15"` ✓
- Observation B: `effectiveDateTime` = `"2023-12-31"` ✗

**Then** notification delivered only for Observation A

#### Scenario: Combine multiple filter criteria

**Given** a subscription with filter:
- `"Observation?category=vital-signs&status=final"`

**When** processing observation

**Then** notification delivered only if:
- `category` = `"vital-signs"` AND
- `status` = `"final"`
- Both conditions must match

---

### Requirement: Support R5 topic-based subscriptions

Use SubscriptionTopic resources to define subscription triggers.

#### Scenario: Define patient-update topic

**Given** a SubscriptionTopic definition:
```json
{
  "resourceType": "SubscriptionTopic",
  "url": "http://example.org/fhir/SubscriptionTopic/patient-update",
  "status": "active",
  "resourceTrigger": [{
    "resource": "Patient",
    "supportedInteraction": ["create", "update"]
  }],
  "notificationShape": [{
    "resource": "Patient",
    "include": ["Patient:general-practitioner"]
  }]
}
```

**When** registering topic:
```go
err := manager.RegisterTopic(topic)
```

**Then** subscriptions can reference this topic URL

#### Scenario: Match event against topic triggers

**Given** a topic monitoring Patient create/update

**When** patient is deleted

**Then** the event does NOT match topic (delete not in `supportedInteraction`)

---

### Requirement: Handle subscription errors and heartbeats

Monitor subscription health and send heartbeat notifications.

#### Scenario: Detect consecutive delivery failures

**Given** a subscription with 5 consecutive failed webhooks

**When** processing failures

**Then** the manager:
- Changes subscription `status` to `"error"`
- Stops further delivery attempts
- Logs error state
- Optionally notifies administrator

#### Scenario: Send heartbeat notification

**Given** a subscription with `heartbeatPeriod` = `60` (seconds)

**When** 60 seconds elapse with no events

**Then** the manager:
- Sends heartbeat notification bundle
- Bundle `type` = `"heartbeat"`
- Contains SubscriptionStatus with `type` = `"heartbeat"`
- Verifies subscriber endpoint is alive

#### Scenario: Recover from error state

**Given** a subscription in `"error"` status

**When** administrator manually reactivates:
```go
err := manager.ReactivateSubscription(ctx, subscriptionID)
```

**Then** the subscription:
- Changes `status` to `"active"`
- Resumes monitoring and delivery
- Resets failure counter

---

### Requirement: Support R4 backport subscriptions

Provide compatibility with FHIR R4 Subscriptions (legacy format).

#### Scenario: Create R4-style subscription

**Given** an R4 Subscription:
```json
{
  "resourceType": "Subscription",
  "status": "active",
  "criteria": "Patient?name=Smith",
  "channel": {
    "type": "rest-hook",
    "endpoint": "https://app.example.com/notify",
    "payload": "application/fhir+json"
  }
}
```

**When** creating subscription via R4 compatibility layer:
```go
err := manager.SubscribeR4(ctx, r4Subscription)
```

**Then** the manager:
- Converts R4 format to internal R5 format
- Maps `criteria` to R5 `filter`
- Maps `channel` to R5 `endpoint` and `channelType`
- Creates equivalent R5 subscription internally

---

### Requirement: Provide pluggable subscription storage

Support different persistence backends for subscription state.

#### Scenario: Use in-memory storage (default)

**Given** no external storage configured

**When** creating manager:
```go
store := subscriptions.NewMemoryStore()
manager := subscriptions.NewManager(store, notifier)
```

**Then** subscriptions are:
- Stored in memory
- Lost on process restart
- Suitable for testing/development

#### Scenario: Implement custom SQL storage

**Given** a custom store implementing `SubscriptionStore` interface:
```go
type SQLStore struct {
    db *sql.DB
}

func (s *SQLStore) Save(ctx context.Context, sub *Subscription) error {
    // SQL INSERT
}

func (s *SQLStore) Get(ctx context.Context, id string) (*Subscription, error) {
    // SQL SELECT
}
```

**When** creating manager with custom store:
```go
sqlStore := NewSQLStore(db)
manager := subscriptions.NewManager(sqlStore, notifier)
```

**Then** subscriptions persist across restarts

---

## MODIFIED Requirements

None (new capability)

---

## REMOVED Requirements

None (new capability)

---

## Cross-References

- **May require**: `smart-on-fhir` spec - Subscribers may need authentication for webhook delivery
- **Related to**: `dicom-fhir-mapping` spec - Subscriptions can monitor ImagingStudy creation
- **Enables**: Real-time clinical decision support, event-driven architectures

---

## Implementation Notes

### Package Structure
```
fhir/subscriptions/
├── manager.go            # Subscription lifecycle management
├── subscription.go       # Subscription resource handling
├── topic.go              # SubscriptionTopic handling
├── notifier.go           # Event notification dispatcher
├── webhook.go            # Webhook delivery with retry
├── filters.go            # Filter matching logic
├── bundle.go             # Notification bundle builder
├── store.go              # Subscription storage interface
├── memory_store.go       # In-memory store implementation
├── errors.go             # Subscription-specific errors
└── r4_compat.go          # R4 backport compatibility layer
```

### Key Types
```go
type Manager struct {
    store    SubscriptionStore
    notifier Notifier
    client   *http.Client
    topics   map[string]*SubscriptionTopic
}

type Subscription struct {
    ID          string
    Status      string  // "requested", "active", "error", "off"
    Topic       string
    Endpoint    string
    ChannelType string
    ContentType string
    Content     string  // "empty", "id-only", "full-resource"
    Filters     []Filter
    Heartbeat   int     // Heartbeat period in seconds
}

type ResourceEvent struct {
    EventType string  // "create", "update", "delete"
    Resource  any     // FHIR resource
    Previous  any     // Previous state (for updates)
}

type Filter struct {
    ResourceType string
    SearchParams map[string]string
}

type SubscriptionStore interface {
    Save(ctx context.Context, sub *Subscription) error
    Get(ctx context.Context, id string) (*Subscription, error)
    List(ctx context.Context) ([]*Subscription, error)
    Delete(ctx context.Context, id string) error
    FindByTopic(ctx context.Context, topic string) ([]*Subscription, error)
}

type Notifier interface {
    Notify(ctx context.Context, sub *Subscription, event ResourceEvent) error
}

var (
    ErrSubscriptionNotFound  = errors.New("subscription not found")
    ErrInvalidWebhookURL     = errors.New("invalid webhook URL")
    ErrWebhookDeliveryFailed = errors.New("webhook delivery failed after retries")
    ErrInvalidFilter         = errors.New("invalid subscription filter")
)
```

### Testing Approach
- Mock HTTP server for webhook delivery
- Verify retry logic with controlled failures
- Test filter matching accuracy (100+ test cases)
- Concurrent notification delivery (race conditions)
- Heartbeat scheduling and delivery
- R4 compatibility tests
- Performance: 10,000 notifications/second

### Dependencies
- Standard library `net/http` for webhooks
- Standard library `time` for heartbeats and retry delays
- No external dependencies

### Webhook Delivery Configuration
```go
type DeliveryConfig struct {
    Timeout      time.Duration  // Request timeout (default: 5s)
    MaxRetries   int            // Max retry attempts (default: 3)
    InitialDelay time.Duration  // Initial retry delay (default: 1s)
    BackoffFactor float64       // Exponential backoff factor (default: 2.0)
}
```

### Retry Strategy
- **Initial delay**: 1 second
- **Backoff**: Exponential with factor 2.0 (1s, 2s, 4s)
- **Max retries**: 3 attempts
- **Timeout**: 5 seconds per request
- **Dead letter queue**: Log persistent failures for manual intervention

### Performance Considerations
- **Async delivery**: Use goroutines to avoid blocking
- **Connection pooling**: Reuse HTTP connections per endpoint
- **Batch notifications**: Group rapid changes (optional optimization)
- **Filter caching**: Cache compiled filter expressions
- **Topic indexing**: Efficient topic lookup by resource type

### FHIR Specification Compliance
- **Primary**: FHIR R5 Subscriptions specification
- **Compatibility**: FHIR R4 Subscriptions via backport layer
- **Reference**: [FHIR R5 Subscriptions](http://hl7.org/fhir/R5/subscriptions.html)