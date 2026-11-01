# ADR-006 — Acesso do cliente à Ordem de Serviço

- **Estado:** Definido
- **Data:** 2026-11-01

## Contexto

O cliente precisa consultar o andamento e decidir sobre orçamentos sem receber acesso às APIs administrativas. Um cadastro de cliente pode ter vários veículos.

## Decisão

O sistema oferecerá dois meios de acesso:

1. **Conta do cliente:** autenticação por e-mail e senha. A sessão fica vinculada ao cliente e permite consultar todos os seus veículos e respectivas ordens de serviço. A senha será armazenada somente como hash seguro e nunca será devolvida pela API.
2. **Acesso por veículo:** autenticação pela placa e pelos cinco primeiros dígitos do CPF/CNPJ. Pontuação e espaços do documento são ignorados antes da comparação. A sessão fica limitada ao veículo informado e às ordens de serviço daquele veículo.

As operações de cliente não usam credenciais administrativas. O servidor valida a sessão e o escopo em cada consulta ou decisão de orçamento. O acesso por veículo exige rate limiting e respostas que não revelem se placa ou documento existem. Credenciais temporárias devem expirar e poder ser revogadas; seus tokens não devem ser registrados em logs.

## Consequências

- O cadastro de cliente precisa armazenar hash de senha e oferecer autenticação e recuperação/troca de senha.
- O acesso por placa permite acesso somente ao veículo correspondente, mesmo quando o cliente possui outros veículos.
- A API de cliente deve listar veículos e OS conforme o escopo da sessão.
- A aprovação/rejeição de orçamento deve obter o cliente autenticado da sessão, em vez de confiar em `client_id` enviado no corpo.
- Os cinco dígitos do documento são um segredo de baixa entropia; tentativas devem ser limitadas e monitoradas, e o acesso não deve revelar dados pessoais além do necessário.
- Redis pode armazenar sessões efêmeras e contadores de rate limit, caso o ADR-011 seja adotado.

## Fora do escopo

Envio real de notificações por e-mail/SMS continua fora do MVP. A senha do cliente não será exposta nos endpoints administrativos de cadastro ou consulta.
