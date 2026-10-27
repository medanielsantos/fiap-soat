# ADR-006 — Acesso do cliente à Ordem de Serviço

- **Estado:** Proposto — confirmar antes da implementação
- **Data:** 2026-10-25

## Contexto

O cliente precisa consultar o andamento e aprovar/rejeitar o orçamento sem receber acesso às APIs administrativas. O requisito não define cadastro de conta ou provedor de identidade do cliente.

## Decisão proposta

Emitir uma credencial aleatória, de alta entropia, limitada a uma única OS e às operações de acompanhamento/aprovação. Guardar apenas hash da credencial; suportar expiração e revogação. Validar credencial e vínculo com a OS em toda requisição. Não aceitar apenas CPF, placa ou ID sequencial como fator de autorização.

## Alternativas consideradas

- Login/senha para todos os clientes: mais gestão de conta do que o MVP exige.
- JWT administrativo compartilhado: privilégio excessivo e fronteira de acesso incorreta.
- OS ID ou CPF como segredo: previsível e enumerável.

## Consequências

- O fluxo precisa definir como entregar a credencial ao cliente; envio real de e-mail/SMS está fora do escopo e pode ser simulado localmente.
- Token não deve aparecer em logs ou URLs; preferir header de autorização.
- A credencial deve permitir somente ações da OS correspondente.
- **Pendente:** duração, momento de emissão e processo de reemissão/revogação.
