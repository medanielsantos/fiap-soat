# Escopo do sistema

## Problema

A operação da oficina depende de anotações e planilhas para clientes, veículos, serviços, peças, diagnóstico e aprovação de orçamento. O sistema centraliza esses registros e torna o estado de cada serviço consultável.

## Capacidades do MVP

- Cadastrar clientes por CPF/CNPJ e veículos por placa.
- Manter catálogo de serviços e peças/insumos.
- Controlar entradas, reservas/baixas e saldo de estoque.
- Abrir OS, registrar diagnóstico e itens, calcular orçamento e disponibilizá-lo ao cliente.
- Registrar aprovação ou rejeição do orçamento.
- Controlar status e histórico da OS, da recepção à entrega.
- Oferecer acompanhamento autorizado ao cliente.
- Oferecer conta do cliente por e-mail e senha, com acesso a todos os veículos cadastrados, e acesso limitado por placa + cinco primeiros dígitos do CPF/CNPJ normalizado.
- Proteger administração por JWT e disponibilizar métricas de duração dos serviços.

## Fora do escopo inicial

Interface web/mobile, pagamentos, agendamento, integração real com fornecedores e envio real de notificações. Essas capacidades podem ser avaliadas depois que o fluxo central estiver definido.

## Atores

- **Atendente:** identifica cliente/veículo, abre OS e registra entrega.
- **Mecânico:** diagnostica, propõe itens, executa e finaliza o serviço.
- **Administrador:** mantém cadastros e estoque e consulta operação.
- **Cliente:** acompanha a própria OS e decide sobre o orçamento.
- **Sistema:** recalcula orçamento, valida transições e mantém histórico.
