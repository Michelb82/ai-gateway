# Future Architecture

The AI Gateway ecosystem is designed around a clear separation between the **control plane** and the **data plane**.

## AI Gateway (this repository)

The AI Gateway is the runtime component responsible for executing AI workloads.

Its responsibilities include:

* Processing incoming requests
* Authentication and validation
* Priority scheduling
* Capability resolution
* Prompt construction
* Invoking inference providers
* Returning responses
* Health reporting

The current implementation uses **Redis CloudEvents** as its ingestion mechanism.

In future releases, request ingestion will become a pluggable adapter layer. Redis will remain a supported adapter, while additional adapters may be introduced to support other messaging systems or protocols without changing the gateway's processing pipeline.

Conceptually:

```text
                    Ingestion Adapters
       ┌──────────────┬──────────────┬──────────────┬─────────┐
       │ Redis Events │    Kafka     │     HTTP     │   ...   │
       └───────┬──────┴───────┬──────┴───────┬──────┴────┬────┘
               │              │              │         │
               └──────────────┴──────────────┴─────────┘
                              │
                              ▼
                        AI Gateway Core
                              │
                    Authentication
                    Validation
                    Priority Scheduling
                    Capability Resolution
                    Prompt Construction
                    Inference Execution
                              │
                              ▼
              Ollama / vLLM / OpenAI-compatible APIs
```

This separation allows the gateway core to remain transport-agnostic while supporting multiple integration patterns.

## AI Gateway Manager *(planned repository)*

The AI Gateway Manager will become the control plane for one or more gateway instances.

Its responsibilities include:

* Capability management
* Model registry
* Inference provider management
* Compute node management
* Configuration publishing
* Gateway discovery
* Cluster health aggregation
* Web-based management interface

Today the gateway publishes a **bindings-only** manifest contract: the manager (or a local file) catalogs models and binds them to gateway-built-in capability ids. Capability prompts, I/O shapes, and parsers remain in the gateway data plane until a later control-plane design moves versioned capability definitions into the manager.

The manager will configure gateway instances but will not participate in inference execution.

This separation allows the management layer to evolve independently from the runtime gateway while enabling centralized configuration and operational visibility across a distributed AI Gateway deployment.
