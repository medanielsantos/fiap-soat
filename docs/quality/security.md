# Segurança e privacidade

- Solicitar apenas dados necessários ao atendimento da oficina.
- Receber CPF/CNPJ e placa via API quando necessários; validar e normalizar no limite do sistema.
- Persistir CPF/CNPJ apenas com dígitos; mascarar CPF/CNPJ nas respostas comuns. A placa pode aparecer integralmente na resposta autorizada quando necessária ao atendimento.
- Não gravar bodies de request/response, CPF/CNPJ, placa, tokens ou senhas em logs.
- Redigir ou mascarar dados em qualquer diagnóstico excepcional. Mascaramento não é anonimização garantida; preferir não registrar o dado.
- Stack traces ficam em telemetria interna restrita, sem PII/payload, e nunca são devolvidos ao consumidor. Erros públicos usam RFC 9457 sanitizada.
- JWT administrativo deve validar assinatura, expiração e claims esperadas; credenciais nunca ficam no repositório.
- Usar SQL parametrizado, autorização por recurso, limites de payload e CORS explícito.
- Usar dados sintéticos em demonstrações e exemplos.
