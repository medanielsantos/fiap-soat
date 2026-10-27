# Convenções da API

## Formato

- REST/JSON com prefixo proposto `/api/v1`.
- Nomes de propriedades JSON em `snake_case` tanto em requests quanto responses: `customer_id`, `vehicle_id`, `created_at`, `total_in_cents`.
- Valores monetários usam campos inteiros explicitamente em centavos, como `unit_price_in_cents` e `total_in_cents`.
- Campos de entrada para CPF/CNPJ podem conter pontuação; o adaptador normaliza para dígitos antes de chamar o caso de uso.
- Respostas comuns não retornam CPF/CNPJ completos; retornam valor mascarado quando ele precisa ser exibido. Placas podem ser retornadas a consumidores autorizados.
- Não serializar diretamente entidades do domínio ou modelos do banco; usar DTOs de resposta explícitos.

## RFC 9457 e snake_case

Os campos normativos de Problem Details permanecem exatamente `type`, `title`, `status`, `detail` e `instance`, conforme a RFC. Propriedades de extensão próprias seguem `snake_case`, por exemplo `invalid_params` e `request_id`. Não renomear campos padrão para `problem_type` etc., pois isso prejudicaria a interoperabilidade.

## Segurança de transporte e respostas

Usar HTTPS fora do ambiente local. Rotas administrativas exigem JWT. Acompanhamento do cliente usa autorização própria limitada à OS. Retornar apenas propriedades necessárias e códigos HTTP coerentes; o contrato OpenAPI documentará autenticação, DTOs e erros.
