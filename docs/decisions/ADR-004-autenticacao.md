# ADR-004 — Autenticação administrativa

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

As operações administrativas precisam de autenticação. O MVP deve expor APIs REST sem permitir acesso anônimo a cadastros, estoque e gestão de OS.

## Decisão

Usar **JWT Bearer** nas rotas administrativas. Validar assinatura, expiração e claims necessárias; autorizar cada rota além de apenas autenticar o token. Segredos e credenciais vêm de configuração externa e nunca são versionados.

## Alternativas consideradas

- Sessões com cookie: adequadas a aplicações web stateful, mas não escolhidas para esta API.
- API key compartilhada: não identifica bem usuários e é difícil de revogar por pessoa.
- Provedor externo de identidade: fora do escopo inicial.

## Consequências

- Endpoints de login e administração serão descritos no OpenAPI.
- Senhas, se armazenadas localmente, serão guardadas somente como hash apropriado.
- Definição de papéis granulares pode ser adicionada quando necessária; JWT não substitui autorização por recurso.
- A autenticação do cliente é separada e está em ADR-006.
