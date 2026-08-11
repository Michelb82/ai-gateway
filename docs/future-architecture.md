# Inference Platform — Future Architecture

# Introduction

The purpose of the platform is to abstract AI inference from the infrastructure required to provide it, while continuously improving the balance between quality, cost and energy consumption.

Applications should not need to know which model is being used, which inference runtime executes it, what hardware it runs on, where that hardware is located, or how the underlying infrastructure is deployed and scaled. They should be able to request an AI capability through a stable interface while the platform determines how that capability is most effectively provided.

This abstraction also creates an opportunity to optimise inference as a complete system. Instead of treating the model and its hardware as fixed components, the platform can make decisions across the entire inference lifecycle. Depending on the requirements of a request, it can select an appropriate model, apply augmentation, choose an inference runtime, route workloads to suitable hardware, manage concurrency and capacity, and avoid unnecessary or disproportionately expensive inference.

The objective is therefore not simply to make inference available, but to improve the quality of the result while reducing the resources required to produce it.

This can be achieved through several complementary mechanisms:

* **Model selection** — using the smallest or most efficient model capable of meeting the required quality.
* **Augmentation** — providing additional context or retrieval when this can improve quality without requiring a substantially larger model.
* **Inference routing** — selecting an appropriate runtime, model or provider based on the characteristics of the request.
* **Workload optimisation** — improving batching, concurrency and scheduling to increase utilisation of available compute.
* **Hardware optimisation** — matching workloads to appropriate CPU, GPU or other accelerator resources.
* **Capacity management** — scaling inference capacity according to demand rather than maintaining unnecessary resources.
* **Policy-driven decisions** — applying quality, security, cost and operational constraints consistently.
* **Runtime lifecycle management** — deploying, replacing and scaling inference runtimes according to the actual requirements of the platform.

These mechanisms require the architecture to understand more than the inference request itself. It must be able to coordinate the request, apply policy, select capabilities, execute inference and manage the infrastructure that makes those capabilities available.

The architecture therefore separates the platform into a data plane and a control plane.

The **data plane** processes inference workloads. It coordinates requests, applies policy, performs augmentation when required, executes inference and returns responses.

The **control plane** manages the services that provide these capabilities. It maintains configuration, registration, service state and desired state without becoming part of the inference request path.

The AI Gateway is therefore renamed to the Inference Gateway. The new name reflects its specific responsibility within this architecture: it provides the abstraction between request orchestration and the underlying inference runtimes, models and hardware.

The resulting architecture separates responsibilities into independently deployable services:

```text
                         CONTROL PLANE
                    ─────────────────────
                       Service Control
                             │
                 configuration / state /
                  registration / desired state
                             │
                             ▼

══════════════════════════════════════════════════════════════════

                          DATA PLANE

Consumer
   │
   ▼
Ingress
   │
   ▼
Orchestration
   │
   ├──────────────► Policy Service
   │
   ▼
Inference Gateway
   │
   ├──────────────► Inference Runtime
   │
   └──────────────► Augmentation
                         │
                         ▼
                    Policy Service
                         │
                         ▼
                  Inference Gateway
                         │
                         ▼
                  Inference Runtime
                         │
                         ▼
                  Inference Gateway
                         │
                         ▼
                    Orchestration
                         │
                         ▼
                    Policy Service
                         │
                         ▼
                       Egress
                         │
                         ▼
                      Consumer
```

The services are composable rather than a mandatory chain. The Inference Gateway can operate without Augmentation, Orchestration can coordinate services other than inference, the Policy Service can serve multiple consumers, and runtime infrastructure can be managed independently of request processing.

The architecture consequently establishes a separation between what a consumer wants to achieve and how the platform achieves it.

The consumer requests a capability.

The platform determines the appropriate inference strategy.

The infrastructure executes that strategy.

This separation is what allows the platform to continuously evolve toward better **quality per unit of cost and energy**, without requiring consumers to change their integration whenever models, runtimes, hardware or infrastructure technologies change.


# Data Plane

The data plane is responsible for receiving, processing and returning inference requests.

It consists of several independently deployable capabilities:

* Ingress
* Orchestration
* Policy Service
* Inference Gateway
* Augmentation
* Inference Runtime
* Egress

These components form a request-processing architecture rather than a single monolithic application.

The central distinction is:

> **Orchestration coordinates the request; the Inference Gateway executes the inference capability.**

---

# Ingress Layer

The **Ingress Layer** is the external entry point into the data plane.

Its responsibility is to accept requests from consumers and provide them to Orchestration.

Ingress may support different protocols or integration mechanisms without requiring changes to the downstream inference components.

Conceptually:

```text
Consumer
   │
   ▼
Ingress Layer
   │
   ▼
Orchestration
```

The ingress layer does not need to understand which model or runtime will ultimately process the request.

---

# Orchestration

**Orchestration** is the central coordinator of the data-plane request lifecycle.

It receives a request from the Ingress Layer and determines how the request should proceed.

Its responsibilities include:

* Coordinating the request lifecycle
* Calling the Policy Service
* Determining which inference capability should be used
* Routing requests to the appropriate Inference Gateway
* Coordinating augmentation
* Managing the response path
* Calling the Policy Service before returning a response
* Forwarding the final response to Egress

Orchestration therefore sits above the individual data-plane capabilities.

It does **not** execute inference itself.

The orchestration flow begins with:

```text
Ingress
   │
   ▼
Orchestration
   │
   ▼
Policy Service
```

The policy decision can determine whether the request is permitted and what constraints apply before the request proceeds.

Orchestration can then route the request to the appropriate Inference Gateway.

---

# Policy Service

The **Policy Service** provides policy decisions to the data plane.

Policy is deliberately separated from both Orchestration and the Inference Gateway so that security and operational rules do not become embedded in individual inference components.

The Policy Service may make decisions concerning:

* Request authorization
* Capability access
* Model or provider access
* Request constraints
* Augmentation permissions
* Response authorization
* Other platform policies

The service can be used independently by components that require policy decisions.

It participates in multiple points of the request lifecycle.

### Initial request

```text
Ingress
   │
   ▼
Orchestration
   │
   ▼
Policy Service
```

### Augmentation

```text
Inference Gateway
   │
   ▼
Augmentation
   │
   ▼
Policy Service
```

### Response

```text
Inference Gateway
   │
   ▼
Orchestration
   │
   ▼
Policy Service
   │
   ▼
Egress
```

This means policy is applied at the boundaries where decisions are required rather than being implemented directly inside the Inference Gateway.

---

# Inference Gateway

The **Inference Gateway** is the core inference abstraction of the data plane.

Its purpose is to hide inference-provider and runtime-specific implementation details from Orchestration and the consumers of the platform.

The gateway is responsible for:

* Capability resolution
* Request validation
* Inference request construction
* Communication with inference runtimes
* Provider abstraction
* Inference-specific resilience
* Response handling
* Health reporting
* Observability

The Inference Gateway should not need to know how the surrounding application processes the request.

It receives an inference request from Orchestration and determines how that request is executed.

The gateway can route directly to an inference runtime:

```text
Orchestration
      │
      ▼
Inference Gateway
      │
      ▼
Inference Runtime
```

or route through an augmentation capability:

```text
Orchestration
      │
      ▼
Inference Gateway
      │
      ▼
Augmentation
      │
      ▼
Inference Gateway
      │
      ▼
Inference Runtime
```

The second path allows augmentation to enrich or transform a request before the final inference execution.

---

# Inference Runtime

The **Inference Runtime** is responsible for actually executing the model.

Examples may include:

* Ollama
* vLLM
* Other inference runtimes
* OpenAI-compatible inference services

The runtime is deliberately hidden behind the Inference Gateway.

Neither the consumer nor Orchestration should need to know which runtime is being used.

Conceptually:

```text
                         Inference Gateway
                                │
                  ┌─────────────┴─────────────┐
                  │                           │
                  ▼                           ▼
              Ollama                         vLLM
                  │                           │
                  └─────────────┬─────────────┘
                                ▼
                              Model
```

The runtime may itself be deployed on different compute infrastructure without affecting the integration contract exposed by the Inference Gateway.

---

# Augmentation

**Augmentation** is an optional data-plane capability.

It exists outside the Inference Gateway because augmentation is not fundamentally an inference-runtime concern.

Examples include:

* Retrieval-augmented generation
* Context retrieval
* Prompt or context enrichment
* Model adaptation
* LoRA-based augmentation
* Additional preprocessing or post-processing

When augmentation is required, the Inference Gateway delegates to it.

```text
Inference Gateway
        │
        ▼
   Augmentation
        │
        ▼
  Policy Service
        │
        ▼
Inference Gateway
```

The augmented request is then returned to the Inference Gateway for inference execution.

This allows the gateway to remain responsible for inference while augmentation remains independently deployable.

A deployment that does not require augmentation does not need to deploy the service.

---

# Response Flow

The response follows a separate return path through Orchestration.

After the Inference Gateway has completed inference, the result is returned to Orchestration.

```text
Inference Gateway
       │
       ▼
Orchestration
       │
       ▼
Policy Service
       │
       ▼
Egress Layer
       │
       ▼
Consumer
```

Orchestration therefore controls the complete lifecycle of the request:

```text
                    REQUEST
                       │
                       ▼
                    Ingress
                       │
                       ▼
                 Orchestration
                       │
                       ▼
                 Policy Service
                       │
                       ▼
               Inference Gateway
                       │
              ┌────────┴────────┐
              │                 │
              ▼                 ▼
      Inference Runtime    Augmentation
                                │
                                ▼
                          Policy Service
                                │
                                ▼
                       Inference Gateway
                                │
                                ▼
                         Inference Runtime
                                │
                                ▼
                       Inference Gateway
                                │
                                ▼
                         Orchestration
                                │
                                ▼
                         Policy Service
                                │
                                ▼
                            Egress
                                │
                                ▼
                            Consumer
```

The augmentation path is optional and may be invoked zero or more times depending on the capability and orchestration strategy.

---

# Inference Gateway Ingestion

The Inference Gateway itself should remain independent from the transport used to communicate with it.

The existing Redis CloudEvents implementation can therefore be treated as an **ingress adapter** rather than a fundamental dependency of the gateway.

Future adapters may include:

* Redis
* Kafka
* HTTP
* Other messaging systems
* Other internal service protocols

Conceptually:

```text
                 Gateway Adapters
        ┌────────────┬────────────┬──────────┐
        │   Redis    │   Kafka    │   HTTP   │
        └─────┬──────┴─────┬──────┴────┬─────┘
              │            │           │
              └────────────┼───────────┘
                           ▼
                  Inference Gateway
                           │
                  Inference abstraction
                           │
                           ▼
                  Inference Runtime
```

This keeps transport concerns separate from inference concerns.

---

# Observability and Resilience

Observability and resilience are cross-cutting concerns of the data plane.

The Inference Gateway should expose standard operational interfaces and telemetry, including:

* Health endpoints
* Metrics
* Logs
* Traces
* OpenTelemetry integration

Inference-specific resilience can include:

* Circuit breakers
* Exponential backoff
* Retry handling
* Provider failure detection
* Runtime health monitoring

These mechanisms protect the inference boundary without requiring Orchestration to implement inference-runtime-specific failure handling.

Orchestration remains responsible for the broader request lifecycle and service-level decisions.

---

# Service Control

**Service Control** is the control-plane component responsible for centralized management of the data-plane services.

It is deliberately outside the request path.

```text
                    CONTROL PLANE

                    Service Control
                          │
             ┌────────────┼────────────┐
             │            │            │
        Configuration   Registry      State
             │            │            │
             └────────────┼────────────┘
                          │
                          ▼

══════════════════════════════════════════════════════════════════

                    DATA PLANE

Ingress → Orchestration → Policy → Inference Gateway
                                      │
                         ┌────────────┴────────────┐
                         ▼                         ▼
                  Runtime                   Augmentation
                                                   │
                                                   ▼
                                                Policy
                                                   │
                                                   ▼
                                             Gateway
                                                   │
                                                   ▼
                                             Runtime

Inference Gateway → Orchestration → Policy → Egress
```

Service Control may provide:

* Service registration
* Service discovery
* Configuration publishing
* Manifest management
* Capability configuration
* Model/provider configuration
* Compute-node information
* Health aggregation
* Management interfaces

The data-plane services can operate independently when centralized control is not required.

When integrated with Service Control, they can register themselves, receive configuration and participate in centralized platform management.

---

# Architectural Evolution

## V1.x — Inference Gateway + Service Control

The first version establishes the fundamental boundaries.

### Inference Gateway

The initial gateway provides:

* Request processing
* Authentication and validation
* Priority scheduling
* Capability resolution
* Prompt construction
* Inference execution
* Health endpoints
* OpenTelemetry
* Circuit breaking
* Exponential backoff
* Service registration

### Service Control

Service Control initially provides:

* Health endpoints
* Service registration
* Manifest endpoints
* Basic configuration
* Gateway discovery
* Model and capability bindings

Deployment and automated scaling remain out of scope.

The initial manifest remains deliberately limited to **bindings**. Capability implementation details such as prompts, I/O shapes and parsers remain within the data plane until a later control-plane design moves those definitions into Service Control.

---

## V2.x — Orchestration + Policy Service

V2.x introduces the request-level coordination layer.

The data-plane flow becomes:

```text
Ingress
   │
   ▼
Orchestration
   │
   ▼
Policy Service
   │
   ▼
Inference Gateway
```

The gateway's direct coupling to Redis is removed by introducing the ingestion adapter boundary.

Orchestration becomes responsible for coordinating the request while the Policy Service provides centralized policy decisions.

The Inference Gateway remains responsible for inference-specific processing.

---

## V3.x — Augmentation

V3.x introduces Augmentation as an optional data-plane capability.

The flow can now become:

```text
Orchestration
      │
      ▼
Inference Gateway
      │
      ▼
Augmentation
      │
      ▼
Policy Service
      │
      ▼
Inference Gateway
      │
      ▼
Inference Runtime
```

The gateway remains independent of Augmentation.

---

# V4.x — Runtime Handler

V4.x introduces the **Runtime Handler** as the infrastructure abstraction for managing inference runtimes.

The Runtime Handler is deliberately **not part of the data-plane request path**. An inference request should never need to pass through the Runtime Handler.

Instead, it operates at the boundary between the platform's desired state and the infrastructure required to provide that state.

```text
                         Service Control
                               │
                               │ desired state
                               ▼
                         Orchestration
                               │
                               │ runtime operation
                               ▼
                        Runtime Handler
                               │
                  ┌────────────┼────────────┐
                  │            │            │
                  ▼            ▼            ▼
              Containers     Nodes       Runtime
                  │            │            │
                  └────────────┼────────────┘
                               ▼
                      Inference Runtime
                               │
                               ▼
                            Model
```

The Runtime Handler is responsible for translating platform-level requirements into infrastructure-specific operations.

For example, the platform may determine that a capability requires:

```text
Capability
    │
    ▼
Model
    │
    ▼
Inference Runtime
    │
    ▼
GPU node
```

The Runtime Handler determines how that requirement is actually realised.

This may involve:

* Creating or removing inference runtime instances
* Selecting or preparing compute nodes
* Deploying inference runtimes
* Updating runtime configuration
* Replacing failed runtimes
* Scaling runtime instances
* Draining runtimes before removal
* Managing runtime lifecycle
* Interacting with container or orchestration infrastructure

The implementation of these operations remains hidden behind the Runtime Handler interface.

The underlying infrastructure could therefore change without requiring changes to the Inference Gateway.

For example:

```text
Runtime Handler
      │
      ├── Docker
      ├── Kubernetes
      ├── Docker Swarm
      ├── Ansible
      ├── Bare Metal
      └── Other infrastructure
```

The Runtime Handler may also consume runtime health and lifecycle information to determine whether the infrastructure has successfully reached the desired state.

## Relationship with Service Control

Service Control maintains the platform-level state and configuration.

The Runtime Handler does not become another central management system.

Instead, the responsibilities remain separated:

| Component             | Responsibility                                                       |
| --------------------- | -------------------------------------------------------------------- |
| **Service Control**   | Maintains configuration, registrations, manifests and platform state |
| **Orchestration**     | Determines and coordinates required operational changes              |
| **Runtime Handler**   | Executes infrastructure-specific runtime operations                  |
| **Inference Gateway** | Processes inference requests                                         |
| **Inference Runtime** | Executes models                                                      |

This creates a clear control loop:

```text
        Desired State
             │
             ▼
      Service Control
             │
             ▼
       Orchestration
             │
             ▼
      Runtime Handler
             │
             ▼
       Infrastructure
             │
             ▼
    Inference Runtime
             │
             ▼
       Observed State
             │
             └──────────────► Service Control
```

The important distinction is that the Runtime Handler **implements infrastructure changes; it does not decide the platform's overall desired state**.

## Independent Use

Like the other platform components, the Runtime Handler should remain independently usable.

It should expose a stable contract for runtime lifecycle operations without requiring the Inference Gateway to exist.

For example, another service could use the Runtime Handler to deploy an inference runtime without sending inference traffic through the rest of the platform.

This preserves the architectural principle that the services are **composable capabilities rather than mandatory stages of one pipeline**.

## Why V4.x

Runtime management is intentionally introduced after the request-processing architecture has stabilised.

The earlier versions establish:

```text
V1  → Inference + basic control
V2  → Request orchestration + policy
V3  → Augmentation
V4  → Mature data-plane integration
V5  → Infrastructure/runtime lifecycle
```

This keeps infrastructure automation separate from inference execution.

The final architecture therefore has two distinct flows:

### Request flow

```text
Ingress
   ↓
Orchestration
   ↓
Policy
   ↓
Inference Gateway
   ↓
[Augmentation]
   ↓
Inference Runtime
   ↓
Inference Gateway
   ↓
Orchestration
   ↓
Policy
   ↓
Egress
```

### Infrastructure flow

```text
Service Control
      ↓
Orchestration
      ↓
Runtime Handler
      ↓
Infrastructure
      ↓
Inference Runtime
      ↓
Observed State
      ↓
Service Control
```

**The request flow executes workloads.
The infrastructure flow creates and maintains the capability to execute those workloads.**


---

# End-State Architecture

The final architecture separates the responsibilities into clear boundaries.

```text
                              CONSUMER
                                  │
                                  ▼
                         ┌────────────────┐
                         │ Ingress Layer  │
                         └───────┬────────┘
                                 │
                                 ▼
                         ┌────────────────┐
                         │ Orchestration  │
                         └───────┬────────┘
                                 │
                                 ▼
                         ┌────────────────┐
                         │ Policy Service │
                         └───────┬────────┘
                                 │
                                 ▼
                       ┌────────────────────┐
                       │ Inference Gateway  │
                       └───────┬────────────┘
                               │
                    ┌──────────┴───────────┐
                    │                      │
                    ▼                      ▼
          ┌──────────────────┐    ┌────────────────┐
          │ Inference Runtime│    │  Augmentation  │
          └────────┬─────────┘    └───────┬────────┘
                   │                      │
                   │                      ▼
                   │              ┌────────────────┐
                   │              │ Policy Service │
                   │              └───────┬────────┘
                   │                      │
                   │                      ▼
                   │              Inference Gateway
                   │                      │
                   └──────────────┬───────┘
                                  ▼
                         ┌────────────────┐
                         │ Orchestration  │
                         └───────┬────────┘
                                 │
                                 ▼
                         ┌────────────────┐
                         │ Policy Service │
                         └───────┬────────┘
                                 │
                                 ▼
                         ┌────────────────┐
                         │  Egress Layer  │
                         └───────┬────────┘
                                 │
                                 ▼
                              CONSUMER


════════════════════════ CONTROL PLANE ═════════════════════════

                         ┌────────────────┐
                         │ Service Control│
                         └───────┬────────┘
                                 │
                 configuration / registration /
                    state / discovery / policy
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                  ▼
        Orchestration      Inference Gateway   Augmentation
              │
              ▼
        Runtime Handler
              │
              ▼
       Infrastructure
```

The fundamental architectural principles are:

### 1. The Inference Gateway abstracts inference

Consumers and Orchestration do not need to understand the implementation of the inference runtime.

### 2. Orchestration owns the request lifecycle

It coordinates ingress, policy, inference, augmentation and egress.

### 3. Policy is independent

Security and policy decisions are provided by a dedicated service rather than being embedded into every component.

### 4. Augmentation is optional

A deployment can use the Inference Gateway directly against an inference runtime without deploying Augmentation.

### 5. The control plane is outside the request path

Service Control manages and configures the data plane but does not process inference requests.

### 6. Services remain independently deployable

The architecture defines contracts between capabilities rather than requiring a single monolithic deployment.

### 7. Infrastructure is abstracted

The Inference Gateway does not know how the inference runtime is deployed, and consumers do not need to know that an inference runtime exists at all.

The resulting abstraction is:

```text
Consumer
   │
   │ "I need capability X"
   ▼
Ingress
   │
   ▼
Orchestration
   │
   │ "How should this request be handled?"
   ▼
Policy + Inference Gateway + optional Augmentation
   │
   │ "How is capability X actually executed?"
   ▼
Inference Runtime
   │
   ▼
Infrastructure
```

The platform therefore separates **request coordination, policy, inference execution, augmentation and infrastructure management** into independent capabilities while maintaining a single coherent request lifecycle.
