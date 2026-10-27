# ADR-001 — Linguagem e framework HTTP

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

O projeto precisa de uma API REST monolítica, simples de executar e organizada para separar domínio, aplicação e infraestrutura.

## Decisão

Usar **Go** como linguagem e **Echo** para o adaptador HTTP. Echo fica restrito à borda da aplicação: roteamento, middleware e handlers. Casos de uso e domínio não dependem de tipos Echo.

## Alternativas consideradas

- Go com `net/http`: opção mais enxuta, mas exigiria montar mais infraestrutura HTTP manualmente.
- Outras linguagens/frameworks: não escolhidos porque Go e Echo já foram definidos para este projeto.

## Consequências

- Dependências serão fixadas no `go.mod`.
- Handlers traduzem DTOs HTTP para comandos/casos de uso.
- O domínio pode ser testado sem iniciar servidor Echo.
- A escolha do framework não determina a organização de domínio.
