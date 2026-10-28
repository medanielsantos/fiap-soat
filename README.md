# Sistema de Gestão de Oficina

Back-end para apoiar o atendimento e a execução de serviços de uma oficina mecânica: clientes, veículos, ordens de serviço, orçamentos, catálogo e estoque.

> Projeto independente baseado nos requisitos do Tech Challenge SOAT. Desenvolvimento incremental, no ritmo do autor. A base da API e o domínio inicial de clientes estão em implementação.

## Objetivo

Registrar o ciclo de serviço do veículo, da recepção à entrega, gerar orçamento a partir de serviços e peças, obter aprovação do cliente, controlar estoque e permitir acompanhamento autorizado da OS.

## Decisões técnicas atuais

- **Go**, em um monólito organizado por contextos de domínio.
- **DDD e Clean Architecture**: domínio isolado de frameworks e infraestrutura.
- **Echo** para adaptador HTTP REST.
- **Uber Dig** para montar o grafo de dependências no bootstrap.
- **PostgreSQL** para persistência relacional e transações.
- **Redis** proposto para dados efêmeros, rate limiting e credenciais temporárias de acompanhamento (ADR-011).
- **OpenAPI** para especificar a API.
- **RFC 9457 Problem Details** para erros HTTP.
- **JWT** para rotas administrativas.
- Dinheiro em `int64` de centavos no Go e `BIGINT` no PostgreSQL; nunca `float`.
- Campos JSON da API em `snake_case`.

## Executar a API

Requer Go 1.27.3 ou superior. Inicie o servidor com:

```sh
go run ./cmd/api
```

A API escuta na porta `8080` por padrão. Defina `APP_HTTP_PORT` para alterar a porta. A rota `GET /health` retorna `{"status":"ok"}`; as rotas da API usam o prefixo `/api/v1`.

O adaptador PostgreSQL usa GORM. Em desenvolvimento local, copie `.env.example` para `.env` e configure `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER` e `DB_PASSWORD`. O arquivo `.env` é carregado automaticamente e ignorado pelo Git. Se `DATABASE_URL` estiver definida no ambiente, ela tem prioridade. Ao executar a API na máquina host, use `DB_HOST=localhost`; em Docker Compose, use o nome do serviço PostgreSQL, como `postgres`. A primeira migração cria apenas a tabela `customers`. Aplique as migrações com `make migrate` ou `go run ./cmd/migrate up`; o comando usa a mesma configuração `.env` da API.

## Clientes

- `POST /api/v1/customers`: cadastra cliente; `document`, `name`, `email` e `phone` são obrigatórios. `status` pode ser omitido e inicia como `active`.
- `GET /api/v1/customers`: lista clientes; retorna `204` se não houver clientes.
- `GET /api/v1/customers/{id}`: consulta um cliente.
- `PUT /api/v1/customers/{id}`: atualiza `document`, `name`, `email`, `phone` e `status` (`active` ou `inactive`).
- `DELETE /api/v1/customers/{id}`: desativa logicamente o cliente; o registro permanece no banco com `deleted_at` preenchido.

CPF/CNPJ são validados e persistidos somente com dígitos. Nas respostas, o documento é mascarado. Erros seguem `application/problem+json`.

## Estrutura Go

Esta é a estrutura-alvo; módulos são criados conforme as capacidades forem implementadas. Diretórios compostos usam kebab-case; nomes de pacotes Go não usam hífen.

```text
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   ├── containers/                 # composition root DI; registra o grafo do Dig
│   ├── customer/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapters/
│   │       ├── in/http/
│   │       └── out/postgres/
│   ├── vehicle/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapters/
│   │       ├── in/http/
│   │       └── out/postgres/
│   ├── service-catalog/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapters/
│   │       ├── in/http/
│   │       └── out/postgres/
│   ├── work-order/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapters/
│   │       ├── in/http/
│   │       └── out/postgres/
│   ├── identity/
│   │   ├── domain/
│   │   ├── application/
│   │   └── adapters/
│   ├── platform/                   # config, logging, HTTP e transações
│   │   └── redis/                  # adaptador Redis proposto
│   └── utils/                      # helpers genéricos sem regra de domínio
├── migrations/
├── api/
│   └── openapi.yaml
├── docs/
├── Dockerfile
├── compose.yaml
├── go.mod
└── README.md
```

`customer` e `vehicle` são módulos distintos. `work-order` referencia seus IDs e conversa por portas de aplicação, sem importar entidades internas desses módulos. O código de helpers compartilhados fica em `internal/utils`; regras de negócio devem permanecer no domínio que as possui. Exemplo de declaração para o diretório `work-order`: `package workorder`.

## Organização das camadas

- **Domain:** entidades, value objects, eventos e invariantes; sem Echo, SQL ou JWT.
- **Application:** casos de uso, portas e coordenação de transações.
- **Adapters:** entrada HTTP/Echo e saída PostgreSQL ou integrações.
- **Platform:** composição transversal, configuração, logging e suporte HTTP.
- **Utils:** funções genéricas realmente reutilizáveis; não deve conter regras de cliente, veículo, estoque ou OS.

## API e dados

Os campos JSON serão `snake_case`, por exemplo `customer_id`, `vehicle_id`, `total_in_cents`. As respostas da API mascaram CPF/CNPJ; a placa pode ser retornada ao consumidor autorizado quando necessária ao fluxo. Nenhuma dessas informações deve aparecer sem proteção em logs ou stack traces.

A API seguirá RFC 9457 com `Content-Type: application/problem+json`. Os campos padronizados da RFC permanecem exatamente `type`, `title`, `status`, `detail` e `instance`; propriedades de extensão próprias usarão `snake_case`.

CPF/CNPJ serão normalizados para somente dígitos antes de persistir. Valores monetários serão armazenados em centavos inteiros. O racional e as regras estão em [`docs/`](docs/README.md).

## PostgreSQL

PostgreSQL foi escolhido por transações ACID, constraints e integridade referencial, especialmente úteis para aprovações de orçamento e mudanças de estoque que precisam ser atômicas. O acesso ficará nos adaptadores de persistência. Redis está proposto apenas para dados efêmeros; sua decisão está em [ADR-011](docs/decisions/ADR-011-redis.md).

## Segurança e qualidade

Rotas administrativas exigem JWT; o acesso do cliente ao acompanhamento é autorizado separadamente e limitado à sua OS. Planejar testes unitários do domínio e casos de uso, testes de integração com PostgreSQL e cobertura mínima de 80% nos domínios críticos. Não há comandos de execução até a implementação da aplicação.

## Documentação técnica

Consulte o [índice de documentação](docs/README.md) para requisitos, domínio, arquitetura, contrato da API, persistência, segurança e testes.
