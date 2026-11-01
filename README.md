# Sistema de Gestão de Oficina

Back-end para apoiar o atendimento e a execução de serviços de uma oficina mecânica: clientes, veículos, ordens de serviço, orçamentos, catálogo e estoque.

> Projeto independente baseado nos requisitos do Tech Challenge SOAT. Desenvolvimento incremental, no ritmo do autor. A base da API e os domínios de clientes e veículos estão em implementação.

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

O adaptador PostgreSQL usa GORM. Em desenvolvimento local, copie `.env.example` para `.env` e configure `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER` e `DB_PASSWORD`. O arquivo `.env` é carregado automaticamente e ignorado pelo Git. Se `DATABASE_URL` estiver definida no ambiente, ela tem prioridade. Ao executar a API na máquina host, use `DB_HOST=localhost`; em Docker Compose, use o nome do serviço PostgreSQL, como `postgres`. As migrações criam as tabelas `customers` e `vehicles`. Aplique as migrações com `make migrate` ou `go run ./cmd/migrate up`; o comando usa a mesma configuração `.env` da API.

## Clientes

- `POST /api/v1/customers`: cadastra cliente; `document`, `name`, `email` e `phone` são obrigatórios. `status` pode ser omitido e inicia como `active`.
- `GET /api/v1/customers?page=1&per_page=20`: lista clientes paginados; `per_page` aceita até `100` e a resposta inclui `data`, `pagination` e `links`. Retorna `204` se não houver clientes.
- `GET /api/v1/customers/{id}`: consulta um cliente.
- `PUT /api/v1/customers/{id}`: atualiza `document`, `name`, `email`, `phone` e `status` (`active` ou `inactive`).
- `DELETE /api/v1/customers/{id}`: desativa logicamente o cliente; o registro permanece no banco com `deleted_at` preenchido.

CPF/CNPJ são validados e persistidos somente com dígitos. Nas respostas, o documento é mascarado. Erros seguem `application/problem+json`.

## Veículos

- `POST /api/v1/vehicles`: cadastra veículo; `customer_id`, `plate`, `brand`, `model` e `year` são obrigatórios. O cliente precisa existir e estar ativo; caso contrário, retorna `422`.
- `GET /api/v1/vehicles?customer_id=&page=1&per_page=20`: lista veículos paginados, com filtro opcional por cliente. Retorna `204` se não houver veículos.
- `GET /api/v1/vehicles/{id}`: consulta um veículo.
- `PUT /api/v1/vehicles/{id}`: atualiza `customer_id`, `plate`, `brand`, `model` e `year`.
- `DELETE /api/v1/vehicles/{id}`: desativa logicamente o veículo (`deleted_at`).

A placa aceita os formatos antigo (`ABC1234`) e Mercosul (`ABC1D23`), com ou sem hífen, e é persistida em maiúsculas sem separadores. É única entre veículos não excluídos (`409` em caso de duplicidade); após exclusão lógica, pode ser cadastrada novamente. `year` deve estar entre `1900` e o ano seguinte ao atual.

## Catálogo e estoque

- `/api/v1/services`: `POST`, `GET` paginado, `GET /{id}`, `PUT /{id}` e `DELETE /{id}` para serviços. Cada serviço tem `internal_code`, `sale_price_in_cents` e `estimated_duration_minutes`.
- `/api/v1/parts`: mesmos endpoints para peças e insumos. Cada peça tem `sku`, `internal_code`, `purchase_price_in_cents`, `sale_price_in_cents` e `unit`. No cadastro, `initial_stock` inicia o saldo (padrão `0`); atualizações não alteram o saldo.
- `POST /api/v1/parts/{id}/stock-movements`: registra entrada (`{"type":"entry","quantity":5,"created_by":"user-id"}`) ou saída (`{"type":"exit","quantity":2,"created_by":"user-id"}`), com `description` e `created_by` opcionais. Saídas acima do saldo disponível retornam `409`.
- `GET /api/v1/parts/{id}/stock-movements?page=1&per_page=20`: consulta o histórico paginado de entradas, saídas, reservas e consumos vinculados a OS. O saldo informado separa quantidade física, reservada e disponível. Peças com saldo ou reserva não podem ser excluídas.

Todos os valores monetários usam centavos inteiros (`*_price_in_cents`). SKU e códigos internos são únicos entre registros ativos. Entradas, saídas, reservas e consumos são registrados em histórico imutável.

As tabelas de clientes, veículos, serviços e peças têm `created_by` e `updated_by`; movimentos de estoque têm `created_by`. Os campos ainda não têm chave estrangeira; as constraints serão adicionadas quando o cadastro de usuários estiver definido.

## Ordens de serviço

- `POST /api/v1/work-orders` abre uma OS para `customer_id` e `vehicle_id`; valida cliente ativo e vínculo do veículo.
- `GET /api/v1/work-orders?status=received&page=1&per_page=20` lista OS; `GET /api/v1/work-orders/{id}` consulta OS, itens, versões de orçamento e histórico.
- `POST /api/v1/work-orders/{id}/diagnosis` grava `diagnosis` e move a OS para diagnóstico.
- `POST /api/v1/work-orders/{id}/items` inclui snapshot de serviço ou peça (`type`, `catalog_id`, `quantity`); `DELETE /api/v1/work-orders/{id}/items/{item_id}?actor_id=...` remove item antes do envio do orçamento.
- `POST /api/v1/work-orders/{id}/quotes` cria uma versão imutável do orçamento.
- `POST /api/v1/work-orders/{id}/quotes/approve` registra aprovação e reserva as peças atomicamente; falta de saldo retorna `409` sem aprovar.
- `POST /api/v1/work-orders/{id}/quotes/reject` registra rejeição e retorna a OS para diagnóstico sem movimentar estoque.
- `POST /api/v1/work-orders/{id}/consumptions` recebe `part_id` e `quantity`; reduz saldo físico e reserva atomicamente.
- `POST /api/v1/work-orders/{id}/complete` finaliza após consumir todas as peças. `POST /api/v1/work-orders/{id}/deliver` registra a entrega.

As rotas de transição recebem `actor_id`. O JWT administrativo e a autorização de cliente ainda serão implementados; neste momento, o identificador é informado no corpo da requisição.

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

Rotas administrativas exigem JWT; o acesso do cliente ao acompanhamento é autorizado separadamente e limitado à sua OS. A meta é cobertura mínima de 80% nos domínios críticos. Execute `make test` para os testes unitários. Para testes PostgreSQL locais, copie `.env.testing.example` para `.env.testing`, preencha as credenciais de uma base dedicada e execute `make test-integration`; no CI, injete `TEST_DATABASE_URL` ou as variáveis `DB_*` como secrets. Para medir a cobertura dos domínios críticos, use `make test-coverage`.

## Documentação técnica

Consulte o [índice de documentação](docs/README.md) para requisitos, domínio, arquitetura, contrato da API, persistência, segurança e testes.
