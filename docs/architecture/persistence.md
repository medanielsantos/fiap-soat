# Persistência com PostgreSQL

## Decisão

PostgreSQL é o banco do MVP. Transações ACID, constraints, índices e integridade referencial atendem ao cadastro e à consistência dos fluxos de OS/orçamento/estoque. É suportado por ferramentas maduras e ambiente containerizado.

## Tipos e normalização

- Valores monetários: Go `int64`, sempre em centavos; PostgreSQL `BIGINT`. Nunca usar `float`.
- CPF/CNPJ: coluna textual canônica contendo somente dígitos; validar antes de gravar e aplicar constraint de unicidade.
- Placa: persistir forma normalizada, conforme regra de normalização definida no domínio.
- Quantidades: escolher unidade e precisão por tipo de item; estoque não pode ficar negativo.
- Itens de orçamento guardam snapshots de descrição e preço para preservar histórico.

## Transações

Salvar aprovação da versão do orçamento e movimentos de estoque relacionados em uma única transação. Erro em qualquer etapa reverte o conjunto. Cobrir concorrência, repetição da requisição e saldo insuficiente em testes de integração.

## Migrações

Versionar mudanças de schema em `migrations/`; não editar silenciosamente migrações já aplicadas em ambientes compartilhados. Credenciais vêm de configuração externa, nunca de commit.
