# Erros HTTP — RFC 9457

Respostas de erro usam `Content-Type: application/problem+json`.

```json
{
  "type": "https://example.invalid/problems/validation-error",
  "title": "Validation failed",
  "status": 422,
  "detail": "One or more fields are invalid.",
  "instance": "/api/v1/work_orders",
  "invalid_params": [
    { "field_name": "customer.document", "reason": "invalid_format" }
  ]
}
```

Os cinco campos padrão devem usar os nomes definidos pela RFC. Extensões são opcionais e usam `snake_case`. Não incluir stack trace, query SQL, caminho local, token, CPF/CNPJ ou corpo da requisição. Mensagens para o cliente devem explicar como corrigir sem revelar detalhes internos.

Mapeamento inicial: validação semântica `422`; não autenticado `401`; não autorizado `403`; inexistente `404`; conflito de estado ou unicidade `409`; falha inesperada `500` com detalhe genérico.
