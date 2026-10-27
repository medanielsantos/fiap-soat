# ADR-009 — Contrato HTTP, snake_case e RFC 9457

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

Clientes da API precisam de nomes consistentes e erros previsíveis. O projeto definiu `snake_case` e RFC 9457.

## Decisão

Usar JSON com propriedades em `snake_case` em requests e responses (por exemplo `customer_id`, `created_at`, `total_in_cents`). Manter OpenAPI versionado em `api/openapi.yaml`. Erros usam `application/problem+json` conforme RFC 9457.

Os campos padronizados da RFC permanecem `type`, `title`, `status`, `detail`, `instance`. Propriedades de extensão seguem `snake_case`, por exemplo `invalid_params` e `request_id`.

## Alternativas consideradas

- `camelCase`: não corresponde à convenção escolhida para esta API.
- Renomear campos base da RFC para snake_case: quebra interoperabilidade com Problem Details.

## Consequências

- DTOs HTTP definem tags JSON explicitamente; não serializar entidades diretamente.
- Campos de erro de domínio são mapeados para códigos HTTP e Problem Details no adaptador.
- Não incluir stack trace, SQL, token ou PII em erros.
