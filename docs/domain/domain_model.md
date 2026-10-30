# Modelo de domínio inicial

## Módulos de domínio

| Módulo | Responsabilidade |
|---|---|
| `customer` | Identidade, contatos e estado do cliente |
| `vehicle` | Identificação e dados do veículo, associado por `customer_id` |
| `service-catalog` | Serviços, peças, insumos e estoque |
| `work-order` | OS, diagnóstico, itens, orçamento, aprovação e acompanhamento |
| `identity` | Usuários administrativos e emissão/validação de JWT |

São módulos de um monólito, não serviços independentes. Clientes e veículos ficam separados por decisão explícita; relações entre módulos usam identificadores e portas de aplicação.

## Agregados candidatos

- `Customer`: documento e dados de contato.
- `Vehicle`: placa, marca, modelo, ano e referência `customer_id`.
- `Service`: descrição, preço vigente e situação.
- `Part`: descrição, preço, unidade e saldo.
- `WorkOrder`: referências a cliente/veículo, status, itens, orçamento e histórico.
- `User`: credenciais armazenadas por hash e permissões.

As fronteiras finais de agregado devem ser revistas ao detalhar consistência transacional e regras de estoque.

## Value objects e invariantes

- CPF/CNPJ: validar dígitos verificadores; armazenar forma canônica apenas com dígitos.
- Placa: aceitar formatos antigo (`ABC1234`) e Mercosul (`ABC1D23`); persistir em maiúsculas sem separadores; única entre veículos não excluídos logicamente.
- Veículo: só pode ser cadastrado ou transferido para cliente existente e ativo, verificado via porta `CustomerDirectory`.
- Dinheiro: `int64` em centavos, nunca `float`.
- Quantidade: tipo inteiro/decimal conforme unidade do item; não permitir estoque negativo.
- Status: mudança somente por transições permitidas.
- Orçamento: preservar versão, itens e preços usados quando enviado ao cliente.

## Eventos candidatos

`WorkOrderReceived`, `DiagnosisStarted`, `WorkOrderItemAdded`, `QuotePrepared`, `QuoteSent`, `QuoteApproved`, `QuoteRejected`, `WorkStarted`, `WorkCompleted`, `VehicleDelivered`, `StockReceived`, `StockReserved`, `StockConsumed`.

Estes nomes são hipóteses para modelagem, não eventos já implementados.
