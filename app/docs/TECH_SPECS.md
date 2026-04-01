# Technical Specifications (Tech Specs)

This document centralizes the fundamental versions and technological inputs used in the development of this Oracle. To facilitate reproducibility and access to usage guides, all tools are arranged in the reference table below and detailed subsequently.

## 🧰 Technology Stack

| Technology / Tool | Version | General Project Role | Reference / Documentation |
| :--- | :---: | :--- | :--- |
| **Go (Golang)** | `1.21+` | Base language. Runs the gRPC server, hexagonal interfaces, and business rules. | [Access Documentation](https://go.dev/doc/) |
| **Hyperledger Besu** | `25.4.1` | Local Enterprise Ethereum Client software. Runs the network and consensus (QBFT). | [Access Documentation](https://besu.hyperledger.org/25.4.1/public-networks/get-started/install) |
| **PostgreSQL** | `15-alpine` | Relational Database. Acts as an off-chain fast cache/replica of the contract. | [Access Documentation](https://www.postgresql.org/docs/15/) |
| **Solidity** | `0.8.27` | Smart Contract Language (`SimpleStorage`). | [Access Documentation](https://docs.soliditylang.org/en/v0.8.27/) |
| **Foundry / Forge** | `latest` | Ultra-fast CLI toolkit for building, testing, and deploying the Solidity Contract. | [Access Documentation](https://book.getfoundry.sh/) |
| **Docker & Compose** | `latest` | Isolated containerization for the distributed nodes and the PostgreSQL server. | [Access Documentation](https://docs.docker.com/compose/) |
| **gRPC & Protobuf** | `proto3` | Modern RPC Communication framework and Oracle call serialization. | [Access Documentation](https://grpc.io/docs/) |
| **go-ethereum** | `v1.13.x` | Official Go module (Geth). Adapts the ABI into callable methods and signs transactions. | [Access Documentation](https://geth.ethereum.org/docs) |
| **pgx** | `v5` | Database driver and advanced connectivity/pool toolkit for Postgres in Go. | [Access Documentation](https://pkg.go.dev/github.com/jackc/pgx/v5) |

---

## 🔬 Technical Detailing

To ensure clarity in understanding the Architecture described in [ARCHITECTURE.md](ARCHITECTURE.md), the main components were divided and their explicit functional roles exposed:

### 1. Core Service and Backend (Golang)

The core of the Oracle mechanism was built in Go language due to its optimized concurrency processing and strong typing. The architectural design purposefully chose not to use robust Web frameworks (like Gin or Fiber) and instead opted for the bare implementation of *RPC* ports made possible by the native `net` package associated with `google.golang.org/grpc`.

### 2. Decentralized Infrastructure and State Machine (Besu & Solidity)

The "Source of Truth" is executed via **Hyperledger Besu** Docker containers. The network's storage logic requires calling Smart Contracts in EVM (Ethereum Virtual Machine) compatible bytecode format, programmed using modern compilers provided by the **Foundry** kit (replacing Hardhat), compiling the `.sol` at the exact version `0.8.27`.

### 3. Blockchain Integration (go-ethereum)

In the Blockchain adaptation repository layer, the bindings and parsers provided by the `go-ethereum` ("Geth") library modules are used. It acts specifically by parsing an *ABI (Application Binary Interface)* to generate transacting Go interfaces, linking up with standard extended modules like `math/big` for the unrestricted representation demanded by the blockchain's `uint256` syntax.

### 4. Cache and Local Data Layer (PostgreSQL & pgx)

Seeking an asynchronous *I/O* shielded against fluctuations and P2P Latencies of the block network, **PostgreSQL** is instantiated via a local container acting strictly as a *State-Cache*. The specialized driver `jackc/pgx/v5` handles the injection of the Intelligent Connection Pool (*Connection Pooling*).
