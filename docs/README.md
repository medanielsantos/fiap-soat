# Documentação técnica

Esta documentação foi escrita para este projeto independente. Ela registra decisões técnicas atuais e pontos ainda em aberto, sem pressupor cronograma de grupo ou etapas de apresentação.

| Área | Documento |
|---|---|
| Decisões técnicas | [decisions/README.md](decisions/README.md) |
| Bootstrap e injeção de dependências | [ADR-012](decisions/ADR-012-injecao-de-dependencias-dig.md) |
| Produto e escopo | [product/scope.md](product/scope.md) |
| Regras do domínio | [domain/domain_model.md](domain/domain_model.md) |
| Ciclo da Ordem de Serviço | [domain/work_order_lifecycle.md](domain/work_order_lifecycle.md) |
| Arquitetura e limites | [architecture/system_design.md](architecture/system_design.md) |
| Estrutura Go | [architecture/go_structure.md](architecture/go_structure.md) |
| PostgreSQL e persistência | [architecture/persistence.md](architecture/persistence.md) |
| Convenções HTTP/JSON | [api/conventions.md](api/conventions.md) |
| Erros RFC 9457 | [api/problem_details.md](api/problem_details.md) |
| Segurança e privacidade | [quality/security.md](quality/security.md) |
| Testes | [quality/testing.md](quality/testing.md) |

## Estados dos documentos

- Decisões marcadas como **definido** refletem o alinhamento atual.
- Itens marcados como **a decidir** devem ser resolvidos antes de depender deles na implementação.
- Diagramas e modelos são propostas técnicas e devem acompanhar a implementação real.
