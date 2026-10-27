# Estrutura do código Go

```text
cmd/api/main.go                  entrada do processo; bootstrap e seleção do modo
internal/containers/             registro/resolução do grafo Dig
internal/customer/               domínio e casos de uso de clientes
internal/vehicle/                domínio e casos de uso de veículos
internal/service-catalog/        serviços, peças e estoque
internal/work-order/             OS, orçamento e acompanhamento
internal/identity/               autenticação e autorização
internal/platform/               configuração, HTTP e logging
internal/utils/                  helpers técnicos genéricos
migrations/                      migrações PostgreSQL
api/openapi.yaml                 contrato REST
```

Dentro dos módulos, separar `domain`, `application`, `adapters/in/http` e `adapters/out/postgres` conforme exista código para cada responsabilidade. Não criar pacotes vazios por obrigação.

`internal/utils` é solicitado para helpers reutilizáveis, como conversões técnicas pequenas. Não deve acumular regras de negócio, validação que pertença a um value object, acesso a banco, dependências Echo nem funções com comportamento específico de um módulo. Se um helper só é usado em `work-order`, mantenha-o nesse módulo.

Diretórios compostos usam kebab-case, mas identificadores de pacote Go não aceitam hífen: `internal/work-order` pode declarar `package workorder`; `internal/service-catalog` pode declarar `package servicecatalog`.

## Injeção de dependências

Somente `internal/containers/` e `cmd/api/main.go` podem importar `go.uber.org/dig`. O restante do código expõe construtores Go comuns, por exemplo `NewWorkOrderService(repo Repository) *WorkOrderService`; não recebe `*dig.Container` nem usa Dig como service locator.

`internal/containers` registra os construtores com `Provide`; o grafo é resolvido via `Invoke` a partir das dependências pedidas pelo ponto de entrada. Dig instancia sob demanda e mantém uma instância por tipo/construtor dentro do container. Erros de resolução devem falhar no bootstrap, antes de servir tráfego.
