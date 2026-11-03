# Sistema de Gestão de Oficina

Back-end para apoiar o atendimento e a execução de serviços de uma oficina mecânica: clientes, veículos, ordens de serviço, orçamentos, catálogo e estoque.

> Projeto independente baseado nos requisitos do Tech Challenge SOAT. Desenvolvimento incremental, no ritmo do autor. A base da API e os domínios de clientes e veículos estão em implementação.

## Integrante do grupo

- Daniel Henrique dos Santos — RM 377497
- Discord: `medanielsantos`

## Objetivo

Registrar o ciclo de serviço do veículo, da recepção à entrega, gerar orçamento a partir de serviços e peças, obter aprovação do cliente, controlar estoque e permitir acompanhamento autorizado da OS.

## Decisões técnicas atuais

- **Go**, em um monólito organizado por contextos de domínio.
- **DDD e Clean Architecture**: domínio isolado de frameworks e infraestrutura.
- **Echo** para adaptador HTTP REST.
- **Uber Dig** para montar o grafo de dependências no bootstrap.
- **PostgreSQL** para persistência relacional e transações.
- **Redis** para rate limiting distribuído nos endpoints de acesso e acompanhamento do cliente; PostgreSQL mantém dados duráveis (ADR-011).
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

O adaptador PostgreSQL usa GORM. Em desenvolvimento local, copie `.env.example` para `.env` e configure PostgreSQL, Redis, `JWT_SECRET` e SMTP quando for enviar e-mails. O `.env` é carregado automaticamente e ignorado pelo Git. `JWT_SECRET` deve ter pelo menos 32 bytes aleatórios; o valor do exemplo serve somente para desenvolvimento local. O envio de e-mail usa SMTP com STARTTLS. `PASSWORD_RESET_URL` aponta para a página que recebe o parâmetro `token`. Se `DATABASE_URL` estiver definida no ambiente, ela tem prioridade. Aplique as migrações com `make migrate` ou `go run ./cmd/migrate up`.

### Docker Compose local

Requer Docker com Compose. O Compose inicia a API, PostgreSQL e Redis. O container da API executa as migrações antes de iniciar o servidor. Os dados do PostgreSQL ficam em `.docker/postgres/data`; o Redis é efêmero e não persiste seus contadores.

```sh
cp .env.example .env
docker compose --env-file .env -f .docker/local/compose.yaml up --build
```

A API fica em `http://localhost:8080`. Para parar os serviços sem apagar os dados do PostgreSQL:

```sh
docker compose --env-file .env -f .docker/local/compose.yaml down
```

O SonarQube Community Build local fica disponível em `http://localhost:9001` (ou na porta definida por `SONARQUBE_HOST_PORT`). Na primeira execução, o Docker baixa a imagem `sonarqube:community`; os dados ficam em volumes Docker persistentes. Acesse com usuário `admin` e senha `admin`; no primeiro login, o SonarQube solicita a troca da senha. [Credenciais padrão documentadas pelo SonarSource](https://docs.sonarsource.com/sonarqube-community-build/server-installation/from-zip-file/overview).

Também é possível usar `make docker-up` e `make docker-down`. O `.docker/Dockerfile` usa build multistage e executa a imagem final como usuário não root. `.docker/entrypoint.sh` aplica as migrações e inicia a API.

## Usuários administrativos

- `make seed`: após `make migrate`, popula `roles`, `permissions` e `role_permissions` e cria três usuários de desenvolvimento com a senha de `DEFAULT_PASSWORD_SEED`: `admin@oficina.local` (admin), `gerente@oficina.local` (manager) e `atendente@oficina.local` (attendant). Só roda com `APP_ENV=local`; em qualquer outro ambiente o comando falha. Usuários já existentes são mantidos sem alteração.
- `POST /api/v1/admin/users`: administrador ou gerente cria usuário; `roles` aceita uma lista de perfis (`admin`, `manager`, `attendant`). Gerentes só podem criar atendentes.
- `GET /api/v1/admin/users?page=1&per_page=20` e `GET /api/v1/admin/users/{id}`: lista paginada e consulta de usuários.
- `PATCH /api/v1/admin/users/{id}/status`: altera o status para `active`, `inactive`, `pending`, `suspended`, `banned` ou `deleted`. Somente usuários ativos podem entrar; uma alteração de status bloqueante também invalida o acesso às rotas protegidas imediatamente.
- `PUT /api/v1/admin/users/{id}/roles`: somente administrador substitui os perfis associados ao usuário.
- `POST` e `DELETE /api/v1/admin/users/{id}/permissions/{permission}`: concede ou revoga uma permissão direta; a autorização final combina permissões diretas e herdadas dos perfis. Restrições do gerente e do atendente são aplicadas também às permissões diretas.
- `DELETE /api/v1/admin/users/{id}`: somente administrador exclui logicamente um usuário; um administrador não pode excluir a própria conta. O RBAC controla quais ações cada perfil pode executar.
- `POST /api/v1/auth/login`: recebe `email` e `password`, retorna `access_token`, `expires_at` e o usuário.
- `POST /api/v1/auth/password/forgot`: recebe `email` e envia o link de redefinição pelo adaptador SMTP. Retorna `202` sem revelar se o email existe.
- `POST /api/v1/auth/password/reset`: recebe `token` e `password`; o token expira em 30 minutos e só pode ser usado uma vez.

Senhas precisam ter entre 12 e 72 bytes e o hash bcrypt fica na coluna `password`. Apenas usuários com status `active` autenticam. Perfis: `admin` pode executar todas as ações; `manager` não pode excluir usuários; `attendant` não pode excluir serviços ou peças, nem alterar preços. Rotas administrativas recebem `Authorization: Bearer <access_token>`; o `user_id` das movimentações da OS vem do JWT validado.

## Clientes

- `POST /api/v1/customers`: cadastra cliente; `document`, `name`, `email` e `phone` são obrigatórios. `password` é opcional e, quando enviado, deve ter de 12 a 72 bytes; somente o hash bcrypt é armazenado. `status` pode ser omitido e inicia como `active`.
- `GET /api/v1/customers?page=1&per_page=20`: lista clientes paginados; `per_page` aceita até `100` e a resposta inclui `data`, `pagination` e `links`. Retorna `204` se não houver clientes.
- `GET /api/v1/customers/{id}`: consulta um cliente.
- `PUT /api/v1/customers/{id}`: atualiza `document`, `name`, `email`, `phone` e `status` (`active` ou `inactive`).
- `DELETE /api/v1/customers/{id}`: desativa logicamente o cliente; o registro permanece no banco com `deleted_at` preenchido.

CPF/CNPJ são validados e persistidos somente com dígitos. Nas respostas, o documento é mascarado. Erros seguem `application/problem+json`.

### Acesso do cliente

- `POST /api/v1/client/auth/login`: recebe `email` e `password`; retorna um token de cliente com acesso aos veículos e às OS do cliente.
- `POST /api/v1/client/auth/plate`: recebe `plate` e `document_prefix` (os cinco primeiros dígitos do CPF/CNPJ; pontuação pode ser omitida); retorna token limitado ao veículo identificado.
- `POST /api/v1/client/auth/password/forgot`: solicita link de redefinição por e-mail. A resposta `202` não revela se a conta existe.
- `POST /api/v1/client/auth/password/reset`: recebe `token` e nova `password`; o token expira em 30 minutos e só pode ser usado uma vez.
- `GET /api/v1/client/vehicles`: lista veículos acessíveis pela sessão.
- `GET /api/v1/client/work-orders` e `GET /api/v1/client/work-orders/{id}`: lista ou consulta OS dentro do escopo da sessão.
- `POST /api/v1/client/work-orders/{id}/quotes/approve` e `/quotes/reject`: aprova ou rejeita orçamento; o cliente é determinado pelo token.

Tokens são enviados em `Authorization: Bearer <access_token>`. Login, acesso por placa e recuperação/redefinição de senha são limitados a cinco tentativas por IP por minuto. As rotas autenticadas do cliente aceitam até 60 requisições por minuto por cliente e IP. Os contadores compartilhados usam Redis; se o Redis estiver indisponível, esses endpoints retornam `503`, enquanto as rotas administrativas continuam disponíveis. Configure `CUSTOMER_PASSWORD_RESET_URL` e SMTP para habilitar o envio do link.

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
- `POST /api/v1/work-orders/{id}/items` inclui snapshot de serviço ou peça (`type`, `catalog_id`, `quantity`); `DELETE /api/v1/work-orders/{id}/items/{item_id}` remove item antes do envio do orçamento.
- `POST /api/v1/work-orders/{id}/quotes` cria uma versão imutável do orçamento.
- `POST /api/v1/work-orders/{id}/quotes/approve` recebe `client_id`, registra aprovação e reserva as peças atomicamente; falta de saldo retorna `409` sem aprovar.
- `POST /api/v1/work-orders/{id}/quotes/reject` recebe `client_id`, registra rejeição e retorna a OS para diagnóstico sem movimentar estoque.
- `POST /api/v1/work-orders/{id}/consumptions` recebe `part_id` e `quantity`; reduz saldo físico e reserva atomicamente.
- `POST /api/v1/work-orders/{id}/complete` finaliza após consumir todas as peças. `POST /api/v1/work-orders/{id}/deliver` registra a entrega.

As ações administrativas usam o `user_id` do JWT; aprovação e rejeição de orçamento registram `client_id`. O histórico da OS armazena exatamente um dos identificadores por evento.

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
│   │   ├── ratelimit/              # contrato para rate limiting
│   │   └── redis/                  # adaptador Redis para rate limiting
│   └── utils/                      # helpers genéricos sem regra de domínio
├── migrations/
├── api/
│   └── openapi.yaml
├── docs/
├── .docker/
│   ├── Dockerfile                  # build multistage
│   ├── entrypoint.sh               # migra e inicia a API
│   ├── local/compose.yaml          # API, PostgreSQL e Redis
│   └── postgres/data/              # dados locais do PostgreSQL
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

PostgreSQL foi escolhido por transações ACID, constraints e integridade referencial, especialmente úteis para aprovações de orçamento e mudanças de estoque que precisam ser atômicas. O acesso fica nos adaptadores de persistência. Redis é usado somente para contadores temporários de rate limit e não substitui o PostgreSQL; veja [ADR-011](docs/decisions/ADR-011-redis.md).

## Segurança e qualidade

Rotas administrativas exigem JWT; o acesso do cliente ao acompanhamento é autorizado separadamente e limitado à sua OS. A meta é cobertura mínima de 80% nos domínios críticos. Execute `make test-unit` para os testes unitários/de aplicação e `make test-integration` para os testes PostgreSQL. Para testes de integração locais, copie `.env.testing.example` para `.env.testing` e preencha as credenciais de uma base dedicada; no CI, as credenciais são criadas para o serviço PostgreSQL do job. `make test` continua executando todos os pacotes disponíveis. Para medir a cobertura dos domínios críticos, use `make test-coverage`.

## Documentação técnica

Consulte o [índice de documentação](docs/README.md) para requisitos, domínio, arquitetura, contrato da API, persistência, segurança e testes.
