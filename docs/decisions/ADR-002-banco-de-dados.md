# ADR-002 — Banco de dados

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

Clientes, veículos, OS, itens orçados e movimentos de estoque têm relações e operações que precisam manter consistência.

## Decisão

Usar **PostgreSQL** como banco relacional. Persistência fica atrás de adaptadores/portas; domínio e casos de uso não importam driver ou tipos SQL.

## Alternativas consideradas

- SQLite: simples localmente, mas menos adequado ao ambiente concorrente e ao uso de containers como ambiente compartilhado.
- Banco documental: flexibilidade de schema não compensa as relações e garantias transacionais necessárias ao MVP.

## Consequências

- Usar migrações versionadas, constraints, índices e queries parametrizadas.
- Aprovação de orçamento e movimentação correlata de estoque devem ser atômicas.
- Executar PostgreSQL localmente via Docker Compose quando a aplicação for criada.
- Valores monetários serão `BIGINT` em centavos, conforme ADR-010.
