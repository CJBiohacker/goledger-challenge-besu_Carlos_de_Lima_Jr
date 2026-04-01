# GoLedger Challenge - Besu/PostgreSQL Oracle

A middleware developed in **Go 1.21+** that acts as a bidirectional oracle, connecting a **Hyperledger Besu (QBFT)** network and a relational **PostgreSQL** database.

This project demonstrates the construction of a reliable bridge between the decentralized state (on-chain) and local cache systems (off-chain), utilizing a hexagonal architecture and gRPC communication.

---

## 📚 Central Documentation

To dive deeper into technical decisions, adopted patterns, and infrastructure evolution, consult the documents below:

- [🏗️ Architecture and Decisions (ADR)](docs/ARCHITECTURE.md): Details the system topology, the use of Hexagonal Architecture, and the management of eventual consistency.
- [📈 Build Phases](docs/BUILD_PHASES.md): Documents the evolution of development phase by phase, from node topologies to the gRPC layer and integration with Foundry.
- [⚙️ Technical Specifications (Tech Specs)](docs/TECH_SPECS.md): Lists and details all languages, frameworks, libraries, and infrastructure tools consolidated in the repository.

---

## 🚀 Quick Start

The infrastructure was designed to be completely isolated via local containers (Docker). Follow the steps below to spin up and test the application.

### 1. Initializing the Blockchain and Private Contract

At the repository root, start the devnet (4-node Besu network) and make the automatic deploy of the `SimpleStorage` smart contract:

```bash
make devnet-deploy
```

> **Attention:** In your terminal, note down the `Contract Address` printed at the end of the deployment execution via Foundry. You will need it in the next step.

### 2. Environment Configuration

Inside the `app/` folder, create the environment variables file from the provided template:

```bash
cd app
cp .env.example .env
```

**Why is this important?**
After running Step 1, it is a security best practice to define credentials locally instead of leaving them in the source code. Open the newly created `.env` file, read it carefully, and define the mandatory variables before proceeding with the Docker commands:

1. **Database Credentials (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`):**
   Define your access user, the desired password, and the initial database name the cluster should launch with. Without these inputs, the container will launch with an inevitable failure.

2. **App ↔ DB Connection String (`DB_URL`):**
   Change the connection URL to accurately reflect the credentials defined above.  
   *Formatted example:* `postgres://johndoe:pass123@localhost:5432/besu_sync?sslmode=disable`

3. **Besu Blockchain Injections:**
   Ensure the default `PRIVATE_KEY` for local testing is configured, and **mandatorily fill** the `CONTRACT_ADDRESS` variable with the address generated in the terminal at the end of the Foundry deployment script.

### 3. Spinning Up the Data Layer (PostgreSQL)

Execute the command below to start the relational database container, which will function as the read cache:

```bash
docker compose up -d
```

> **Attention (Security & Infrastructure):** Notice that by using the `env_file: - ./app/.env` declaration in [docker-compose.yaml](../docker-compose.yml), it is guaranteed that database credentials follow an injection of dynamic parameters when building the container (avoiding the hardcoded sharing of secrets across the shell). Consequently, security remains fully intact without the need to write large bash scripts!

### 4. Starting the gRPC Server

With all infrastructure running, install dependencies (if necessary) and start the Go server:

```bash
go mod tidy
go run cmd/server/main.go
```

The server will be initialized on port `:50051`.

---

## 🛠️ Using the API (Tests via Postman)

The project exposes a server with **gRPC Reflection enabled**, eliminating the need to manually import `.proto` files into your client, providing automatic endpoint discovery.

**To test the operations in Postman:**

1. Create a new **gRPC** request pointing to `localhost:50051`.
2. In the corresponding tab, turn on the button to use **"Server Reflection"**. The methods (`Set`, `Get`, `Sync`, `Check`) will load automatically.
3. Call the desired method informing the compatible message.

### Main Endpoints and Responses

- **`Set`**: Registers a new numeric value *on-chain* (performs a paid transaction in the Besu network contract).
  - *Request Payload:*

    ```json
    { "value": "42" }
    ```

  - *Expected Return (`JSON`):*

    ```json
    {
      "success": true,
      "tx_hash": "0x12b23a54f0a20e4c8be8..."
    }
    ```

- **`Get`**: Performs a read based on the view state directly on the blockchain. *Gas Free.*
  - *Request Payload:* `{}`
  - *Expected Return (`JSON`):*

    ```json
    { "value": "42" }
    ```

- **`Sync`**: Extracts the version of the truth from the contract (current on-chain state) and performs the corresponding *UPSERT*, inserting it into the PostgreSQL database as a fixed ID (`state_cache id=1`).
  - *Request Payload:* `{}`
  - *Expected Return (`JSON`):*

    ```json
    {
      "success": true,
      "synced_value": "42"
    }
    ```

- **`Check`**: Audits and crosses the DB vs Blockchain data, indicating its match via a boolean matrix.
  - *Request Payload:* `{}`
  - *Expected Return (`JSON` - Synchronized State):*

    ```json
    {
      "in_sync": true,
      "db_value": "42",
      "chain_value": "42"
    }
    ```

  - *Expected Return (`JSON` - Local Divergence/Asynchrony):*

    ```json
    {
      "in_sync": false,
      "db_value": "10",
      "chain_value": "42"
    }
    ```
