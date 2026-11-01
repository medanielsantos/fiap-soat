# Segurança e privacidade

## Fronteira de autenticação da API

- Rotas públicas limitam-se à verificação operacional `GET /health`, autenticação administrativa (`/api/v1/auth/*`) e emissão de sessão de cliente (`POST /api/v1/client/auth/login` e `/api/v1/client/auth/plate`). A recuperação e redefinição de senha administrativa também ficam sob `/auth/*` porque precisam estar acessíveis antes da sessão.
- Toda operação de cadastro, consulta ou alteração administrativa exige JWT administrativo. Isso inclui `/customers`, `/vehicles`, `/services`, `/parts`, `/work-orders` e `/admin/users`; o RBAC ainda restringe cada operação conforme perfil e permissão.
- Rotas de consulta e decisão disponíveis ao cliente exigem token de cliente. Esse token tem escopo de cliente completo ou de um veículo, conforme o método de autenticação.
- A especificação OpenAPI aplica autenticação administrativa por padrão e declara explicitamente as operações públicas e as que usam token de cliente.

- Solicitar apenas dados necessários ao atendimento da oficina.
- Receber CPF/CNPJ e placa via API quando necessários; validar e normalizar no limite do sistema.
- Persistir CPF/CNPJ apenas com dígitos; mascarar CPF/CNPJ nas respostas comuns. A placa pode aparecer integralmente na resposta autorizada quando necessária ao atendimento.
- Não gravar bodies de request/response, CPF/CNPJ, placa, tokens ou senhas em logs.
- Redigir ou mascarar dados em qualquer diagnóstico excepcional. Mascaramento não é anonimização garantida; preferir não registrar o dado.
- Stack traces ficam em telemetria interna restrita, sem PII/payload, e nunca são devolvidos ao consumidor. Erros públicos usam RFC 9457 sanitizada.
- JWT administrativo deve validar assinatura, expiração e claims esperadas; credenciais nunca ficam no repositório.
- Usar SQL parametrizado, autorização por recurso, limites de payload e CORS explícito.
- Usar dados sintéticos em demonstrações e exemplos.
