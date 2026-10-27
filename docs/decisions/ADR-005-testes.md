# ADR-005 — Estratégia e organização de testes

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

O projeto precisa verificar regras de domínio e fluxos integrados, respeitando ferramentas e convenções Go sem gerar uma árvore de testes difícil de manter.

## Decisão

Manter testes unitários em arquivos `*_test.go` junto ao pacote testado. Manter testes de integração em `tests/integration/`, em poucos arquivos agrupados por contexto/fluxo. Não espelhar toda a estrutura de produção nem criar um arquivo por cenário.

Nomes de cenários em `t.Run`:

- Positivo: `test_it_should_<behavior>_when_<condition>`.
- Negativo: `test_it_should_not_<behavior>_when_<condition>`.

Funções externas de teste continuam com prefixo `Test`, conforme descoberta do `go test`.

## Alternativas consideradas

- Todos os testes em uma pasta central: piora teste de detalhes privados e aumenta distância do pacote.
- Uma subpasta por teste/caso: gera fragmentação excessiva.

## Consequências

- Testes de domínio são rápidos, sem banco ou HTTP.
- Integração cobre PostgreSQL, handlers Echo, autorização e transações.
- Meta de cobertura: pelo menos 80% em `work-order` e `service-catalog` nos domínios críticos.
- Agrupar cenários relacionados com `t.Run`; separar arquivo quando a navegação ou responsabilidades pedirem.
