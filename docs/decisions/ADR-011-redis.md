# ADR-011 — Uso do Redis

- **Estado:** Definido
- **Data:** 2026-11-01

## Contexto

O acesso do cliente precisa limitar tentativas de autenticação e abuso nos endpoints de acompanhamento. As contagens devem ser compartilhadas entre instâncias da API. PostgreSQL continua armazenando os dados de negócio e os tokens de redefinição de senha.

## Decisão proposta

Usar **Redis somente para contadores efêmeros de rate limit**:

1. Limitar login por e-mail, acesso por placa e solicitações/redefinições de senha do cliente a cinco tentativas por IP por minuto.
2. Limitar endpoints autenticados de acompanhamento e decisão a 60 requisições por minuto por cliente e IP.
3. Incrementar contadores atomicamente com `INCR` e `PEXPIRE`; guardar no Redis somente a chave derivada por SHA-256, nunca o IP ou identificador original.

PostgreSQL permanece a fonte de verdade de clientes, veículos, catálogo, estoque, OS, orçamento e histórico. JWTs de cliente e administração são assinados e validados pela aplicação. Tokens de redefinição permanecem armazenados como hash no PostgreSQL. Redis não armazena sessões ou credenciais e não é usado como cache de catálogo.

## Alternativas consideradas

- Somente PostgreSQL para rate limits: possível, mas exige contadores e limpeza próprios e aumenta escrita operacional.
- Redis como cache geral: não proposto no MVP por trazer invalidação e risco de dado desatualizado sem necessidade comprovada.
- Não adicionar Redis: reduz componentes, mas deixa os limites de requisições locais a cada instância e não compartilhados entre réplicas.

## Consequências

- Redis é serviço no Compose local e adaptador isolado sob `internal/platform/redis`.
- A indisponibilidade ou falta de memória no Redis retorna `503` nos endpoints de cliente sujeitos ao rate limit (fail closed). Operações administrativas e persistência em PostgreSQL continuam independentes do Redis.
- O Redis local não persiste contadores; reiniciar o serviço inicia novas janelas de rate limit.
- O serviço usa senha, limite de 64 MB, política `noeviction` e expiração dos contadores. Não registrar chaves ou tokens.
- Contadores usam janela fixa de um minuto e script Lua para que incremento e expiração sejam atômicos.

## Pontos para produção

- Configurar proxy confiável para que o IP usado no rate limit não aceite cabeçalhos encaminhados arbitrários.
- Avaliar alta disponibilidade, TLS, rotação de senha e capacidade do Redis para o ambiente de produção.
- Credenciais revogáveis ou de uso único, caso sejam introduzidas, exigem uma decisão própria; elas não são armazenadas no Redis nesta versão.
