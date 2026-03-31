# Arquitetura do Sistema

## Topologia Inicial (Fase 1)

A infraestrutura inicial do projeto é composta por duas camadas principais executando localmente:

1. **Rede Blockchain (Hyperledger Besu):**
   Executada em contêineres separados (rede de 4 nós), fornecendo o ambiente de execução de contratos inteligentes e o estado global da aplicação.

2. **Banco de Dados (PostgreSQL):**
   Executado via `docker-compose` (versão 15-alpine, mapeado na porta 5432). O banco `besu_sync` atua como nosso estado local/cache.

Essa separação estabelece a fundação para a comunicação entre o estado persistente on-chain (lento, consistente) e o cache off-chain (rápido, altamente disponível).

## Camada de Delivery via gRPC (Fase 2)

A interface de comunicação da nossa aplicação é desenhada sobre **gRPC** com Serialização via **Protocol Buffers (Protobuf)**. O contrato da API está centralizado no serviço `OracleService`, que garante o *Design by Contract* através de métodos estritos:

- `Set`: Escreve e muda o estado global (Besu).
- `Get`: Lê o estado global atual da blockchain.
- `Sync`: Puxa a verdade da blockchain e força a consistência eventual no nosso banco PostgreSQL.
- `Check`: Pondera as duas fontes de estado para avaliar divergências.

**Design e Observabilidade:**

- O servidor gRPC inicializa na porta `50051`.
- O pacote `log/slog` foi configurado para estruturar os logs nativamente em JSON.
- A funcionalidade `grpc-reflection` está embutida por padrão, permitindo auto-descobrimento de endpoints para ferramentas de debug como gRPCurl ou Postman (essencial para testes funcionais sem a necessidade de partilhar o arquivo `.proto` em todos os ambientes).

## Camada de Repositório e Conexão de Banco de Dados (Fase 3)

Alinhados com a Arquitetura Hexagonal (Ports & Adapters) e o *Design By Contract*, blindamos as operações de persistência:
1. **A Porta (Interface `OracleRepository`):** Define estritamente o que a aplicação pode pedir ao banco.
2. **O Adaptador (Struct `postgresOracleRepo`):** É a única peça que de fato interage com o driver `pgx/v5` gerenciando queries de leitura e Escrita (UPSERT).

O gerencialmento base do app é orquestrado através do `pgxpool`, oferecendo um pool eficiente de conexões ao em vez de criar novas reconexões do zero para cada hit do end-point gRPC, prevenindo gargalos assíncronos.  

**Teorema CAP (`state_cache`):** Para lidar com a assimetria de tempo entre a Blockchain (foco em Consistência/Partição) e o App local (foco em Disponibilidade), o adaptador auto-executa a criação da tabela `state_cache` onde o "ID" é eternamente 1. Este registro único atua fundamentalmente como um *cache* espelho para mitigar os gargalos em operações de leitura, transferindo a carga do nó do Besu para a tabela rápida do Postgres.
