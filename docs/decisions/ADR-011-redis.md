# ADR-011 — Uso do Redis

- **Estado:** Proposto — validar escopo de uso
- **Data:** 2026-10-25

## Contexto

O sistema precisa proteger endpoints administrativos e de acompanhamento do cliente contra abuso. A credencial de acompanhamento proposta em ADR-006 deve ter escopo limitado e expiração. PostgreSQL deve continuar armazenando os dados de negócio de forma durável.

## Decisão proposta

Usar **Redis apenas para dados efêmeros**, inicialmente:

1. Aplicar rate limiting em login e endpoints de acompanhamento/aprovação.
2. Armazenar credenciais temporárias de acompanhamento com TTL, indexadas por hash do token e vinculadas a uma OS/escopo. Nunca persistir o token em claro.

PostgreSQL permanece a fonte de verdade de clientes, veículos, catálogo, estoque, OS, orçamento e histórico. Redis não é usado como cache de catálogo nem como armazenamento exclusivo de informação necessária à consistência financeira/estoque.

## Alternativas consideradas

- Somente PostgreSQL para credenciais temporárias e rate limits: possível, mas exige limpeza/contadores próprios e aumenta escrita operacional.
- Redis como cache geral: não proposto no MVP por trazer invalidação e risco de dado desatualizado sem necessidade comprovada.
- Não adicionar Redis: reduz componentes, porém deixa os controles temporários propostos sem infraestrutura dedicada.

## Consequências

- Se adotado, Redis será serviço local no Compose e adaptador isolado sob `internal/platform/redis`.
- A indisponibilidade do Redis impede acompanhamento autenticado; operações administrativas baseadas em JWT e PostgreSQL podem continuar, conforme política de rate limit definida.
- Perda/reinício do Redis invalida credenciais temporárias, que deverão poder ser reemitidas.
- Usar TTL, limite de memória, autenticação/configuração segura e não registrar valores de tokens.
- Testes de integração devem cobrir expiração, revogação, rate limit e comportamento sem Redis.

## Pontos a validar

- Confirmar que credencial temporária de acompanhamento será armazenada no Redis.
- Confirmar limites e janela do rate limiting.
- Decidir se Redis será requisito de execução local do MVP ou se entra após os fluxos principais.
