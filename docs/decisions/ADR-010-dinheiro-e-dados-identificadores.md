# ADR-010 — Dinheiro e dados identificadores

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

Orçamentos e preços requerem cálculos exatos. CPF/CNPJ e placas são necessários a fluxos do negócio, mas não devem ser expostos em logs sem controle.

## Decisão

- Valores BRL usam `int64` de centavos no Go e `BIGINT` no PostgreSQL; nunca `float`. JSON usa nomes explícitos como `unit_price_in_cents`.
- CPF/CNPJ são validados e persistidos como texto com somente dígitos. Respostas comuns mascaram CPF/CNPJ; não retornam o valor completo.
- Placa é normalizada e pode ser retornada completa ao consumidor autorizado quando necessária ao atendimento.
- Não registrar corpo de requisição/resposta, CPF/CNPJ, placa, tokens ou credenciais. Stack traces ficam internos, sem PII, e não são enviados ao cliente.

## Alternativas consideradas

- `float64` para moeda: pode introduzir erros de arredondamento.
- Guardar CPF/CNPJ com pontuação: complica unicidade e comparação canônica.
- Registrar payload integral para diagnóstico: aumenta exposição e não é necessário para logs operacionais.

## Consequências

- Máscara de CPF/CNPJ e redação de logs pertencem à camada de apresentação/observabilidade; não alteram valor canônico.
- Mascaramento reduz exposição, mas não equivale a anonimização garantida.
- Dados de demonstração devem ser sintéticos.
