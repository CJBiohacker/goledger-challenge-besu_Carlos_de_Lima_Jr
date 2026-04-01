# System Architecture

## Initial Topology (Phase 1)

The project's initial infrastructure consists of two main layers running locally:

1. **Blockchain Network (Hyperledger Besu):**
   Executed in separate containers (4-node network), providing the smart contract execution environment and the application's global state.
2. **Database (PostgreSQL):**
   Executed via `docker-compose` (version 15-alpine, mapped to port 5432). The `besu_sync` database acts as the local state/cache.

This separation establishes the foundation for communication between the persistent on-chain state (slow, consistent) and the off-chain cache (fast, highly available).

## Delivery Layer via gRPC (Phase 2)

The application's communication interface is designed over **gRPC** with Serialization via **Protocol Buffers (Protobuf)**. The API contract is centralized in the `OracleService` service, which guarantees *Design by Contract* through strict methods:

- `Set`: Writes and mutates the global state (Besu).
- `Get`: Reads the current global state from the blockchain.
- `Sync`: Pulls the truth from the blockchain and enforces eventual consistency in the PostgreSQL database.
- `Check`: Weighs both state sources to evaluate divergences.

**Design and Observability:**

- The gRPC server initializes on port `50051`.
- The `log/slog` package was configured to structure logs natively in JSON.
- The `grpc-reflection` functionality is embedded by default, allowing auto-discovery of endpoints for debugging tools like gRPCurl or Postman (essential for functional testing without needing to share the `.proto` file across all environments).

## Repository Layer and Database Connection (Phase 3)

Aligned with Hexagonal Architecture (Ports & Adapters) and *Design By Contract*, the persistence operations were shielded:

1. **The Port (`OracleRepository` Interface):** Strictly defines what the application can ask the database.
2. **The Adapter (`postgresOracleRepo` Struct):** Is the only piece that actually interacts with the `pgx/v5` driver managing Read and Write (UPSERT) queries.

The base management of the app is orchestrated through `pgxpool`, offering an efficient connection pool instead of creating new reconnections from scratch for every hit to the gRPC endpoint, thereby preventing asynchronous bottlenecks.

**CAP Theorem (`state_cache`):** To deal with the time asymmetry between the Blockchain (focused on Consistency/Partition) and the local App (focused on Availability), the adapter auto-executes the creation of the `state_cache` table where the "ID" is eternally 1. This single record basically acts as a mirror *cache* to mitigate read operation bottlenecks, transferring the load from the Besu node to the fast Postgres table.

## Blockchain Layer and Orchestration (Phase 4)

The adapter for the Blockchain Network solves three of the biggest problems found in standard snippets:

1. **Dynamic ABI Management**: The client locates and reads the static `.json` file generated natively by Foundry through the *Factory Pattern* technique in its instantiation, avoiding the *hardcode* of the `abi` block signature.
2. **Safe Serialization and Casting:** All string returns pass through conversion verification to the language driver's `math/big.Int`, maintaining alignment with the EVM (`uint256`) and guaranteeing no panics.
3. **Receipt Validation**: The `Set` API's `SetValue` logic doesn't emit false successes — it waits for the `bind.WaitMined()` event and actively verifies if the Hash return status is `1` (Success at the base layer).

## Polishing, Documentation, and Standardization (Phase 5)

The conclusive phase of the project focused on visual, grammatical, and architectural consolidation, documenting all the cogs created so that other developers and evaluators can understand the ecosystem dynamically and fluidly:

1. **Master Hub Documentation**: Refactoring of the main showcase file `README.md`, restructured with installation guidelines (Quick Start), security rationale for `.env` environment variables, and extensive detailing of the Payload and Response schemas in JSON for the gRPC protocol (`Set`, `Get`, `Sync`, `Check`).
2. **Globalization and Standardization (`en-US`)**: A complete scan was executed across the repository (`.go` files, `bash` scripts, `.proto` protobuf artifacts, and dotfiles) with the intention of translating all interactive technical comments from Portuguese (pt-BR) to the English language, aligning with major Open Source projects.
3. **Technical Specifications (`TECH_SPECS.md`)**: Creation of the unified summary detailing all involved technologies via *Markdown Tables*, specifying versions and substantiating the reasoning behind each decision (e.g., *Golang 1.21+*, *Hyperledger Besu*, *Postgres 15-alpine*, *gRPC*, *pgx/v5*, and *Foundry*).
4. **Visual Diagramming and References (`ARCHITECTURE.md`)**: Visual and academic enhancement of the structure via integration with dynamic **Mermaid** graphs directly in the Markdown body to show the flows of the Hexagonal Architecture (Ports & Adapters). Technical literature validating the adopted premises was also added, such as the CAP Theorem and *Refactoring Guru*.
