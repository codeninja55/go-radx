# Software Design Principles

## KISS (Keep It Simple, Stupid)

- Solutions must be straightforward and easy to understand.
- Avoid over-engineering or unnecessary abstraction.
- Prioritise code readability and maintainability.

## YAGNI (You Aren’t Gonna Need It)

- Do not add speculative features or future-proofing unless explicitly required.
- Focus only on immediate requirements and deliverables.
- Minimise code bloat and long-term technical debt.

## SOLID Principles

- **Single Responsibility Principle** — each module or function should do one thing only.
- **Open-Closed Principle** — software entities should be open for extension but closed for modification.
- **Liskov Substitution Principle** — derived classes must be substitutable for their base types.
- **Interface Segregation Principle** — prefer many specific interfaces over one general-purpose interface.
- **Dependency Inversion Principle** — depend on abstractions, not concrete implementations.

## The Twelve-Factor App

1. **Codebase** - One codebase tracked in revision control, many deploys

- Every twelve-factor app must be tracked in a version control system with a strict one-to-one correlation between
  the codebase and the app
- A codebase can be a single repository or a set of repos sharing a root commit
- One codebase can have multiple deploys (production, staging, local development), where each deploy is a running
  instance
- Different deploys may run different versions, but share the same fundamental codebase
- Multiple codebases means it's a distributed system, not a single app
- Shared code should be extracted into libraries rather than copied across apps

2. **Dependencies** - Explicitly declare and isolate dependencies

- Applications must declare all dependencies completely and exactly via a dependency declaration manifest
- Never rely on implicit system-wide package availability
- Use dependency isolation tools to prevent unintended system dependencies
- Apply dependency specifications consistently across development and production environments
- Simplifies onboarding for new developers and enables deterministic build processes
- Do not assume system tools will be available; vendor them into the app if necessary

3. **Config** - Store config in the environment

- Store configuration in environment variables, not as constants in code
- Config includes resource handles, credentials, and per-deploy values that vary between environments
- Environment variables should be granular controls, fully orthogonal to other env vars
- Ensures the codebase can be made potentially open-sourceable without exposing credentials
- Avoid scattered config files in different formats to prevent accidental commits

4. **Backing Services** - Treat backing services as attached resources

- A backing service is any network-based service the app consumes (databases, messaging systems, SMTP, caching)
- Make no distinction between local and third-party services
- Services are attached resources accessed via URL or credentials stored in configuration
- Can be swapped out without changing application code
- Each distinct service is considered a separate resource
- Enables flexibility like replacing a misbehaving database or switching between local and third-party services

5. **Build, Release, Run** - Strictly separate build and run stages

- Build stage: transforms code into an executable bundle, fetches dependencies, compiles binaries and assets
- Release stage: combines the build with current configuration, creating a release with a unique identifier
- Run stage: executes the app in the runtime environment, launching app processes
- Releases are append-only and cannot be mutated
- Impossible to make changes to the code at runtime
- Deployment tools should support release management including rollback capabilities

6. **Processes** - Execute the app as one or more stateless processes

- Twelve-factor processes are stateless and share-nothing
- Any data requiring persistence must be stored in a stateful backing service
- Process memory or filesystem can be used for brief, single-transaction caching only
- Never assume cached data will be available in future requests
- Sticky sessions are explicitly discouraged; use time-expiring datastores like Memcached or Redis
- Design processes that can be easily scaled, replaced, and restarted without losing functionality

7. **Port Binding** - Export services via port binding

- Be completely self-contained and not depend on external webserver containers
- Export HTTP as a service by binding to a port and listening to requests
- Implement web services directly within the application's code using webserver libraries
- In development, services are accessed locally (e.g., http://localhost:5000/)
- Routing between public hostnames and app processes is handled by a separate routing layer
- One app can become a backing service for another by providing its URL as a resource handle

8. **Concurrency** - Scale out via the process model

- Processes are first-class citizens in application architecture
- Each type of work can be assigned to a specific process type (web processes, worker processes)
- Applications should scale horizontally across multiple processes and physical machines
- Processes should be share-nothing and horizontally partitionable
- Avoid daemonizing or writing PID files; rely on OS process managers or cloud platform tools
- Primary focus is on distributing workloads across multiple independent processes

9. **Disposability** - Maximize robustness with fast startup and graceful shutdown

- Processes should have fast startup time (ideally a few seconds)
- Must shut down gracefully when receiving a SIGTERM signal
- Web processes should stop listening, complete current requests, and exit cleanly
- Processes should be robust against sudden death
- Worker processes should return incomplete jobs to work queues
- All jobs should be reentrant (able to be safely retried)

10. **Dev/Prod Parity** - Keep development, staging, and production as similar as possible

- Minimize Time Gap (delay between code development and deployment)
- Minimize Personnel Gap (involve developers directly in deployment and monitoring)
- Minimize Tools Gap (use similar technology stacks across environments)
- Use the same type and version of backing services in all environments
- Avoid using lightweight local services that differ from production services
- Leverage tools like Docker and Vagrant to create consistent environments

11. **Logs** - Treat logs as event streams

- Logs are time-ordered events from all running processes and services
- A twelve-factor app never concerns itself with routing or storage of its output stream
- Processes should write event streams directly to stdout
- The execution environment captures and manages log routing
- Logs can be routed to files, terminal monitoring, log indexing systems, or data warehousing systems
- Enables finding past events, creating trend graphs, and setting up active alerting

12. **Admin Processes** - Run admin/management tasks as one-off processes

- Admin tasks (database migrations, console shells, one-time scripts) run as one-off processes
- Use the same environment as regular long-running processes
- Run against a specific release and use identical dependency isolation techniques
- Admin code must ship with application code to avoid synchronization issues
- In local environments, run via direct shell commands; in production, use remote command execution
- Treat administrative tasks with the same rigor as main application processes

You can refer to the following [link here](ttps://12factor.net).
