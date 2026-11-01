# Linguagem Ubíqua — Oficina

Este glossário define os termos usados por atendimento, oficina, documentação e código. Os nomes técnicos entre crases apontam para os conceitos presentes na implementação. Quando um termo é uma convenção do negócio e não um tipo formal no código, isso está indicado.

## Termos do domínio

| Termo | Definição compartilhada | Nome no código / API |
|---|---|---|
| Cliente | Pessoa responsável pelo veículo e autorizada a acompanhar ou decidir sobre o orçamento da OS vinculada. | `Customer`, `customer_id`, sessão de cliente |
| Documento | CPF ou CNPJ usado para identificar o cliente; é normalizado para dígitos no armazenamento e mascarado em respostas comuns. | `Document`, `document` |
| Veículo | Bem recebido pela oficina, identificado por placa e associado a um cliente. | `Vehicle`, `vehicle_id`, `plate` |
| Placa | Identificador do veículo nos formatos antigo ou Mercosul; persistido em maiúsculas e sem separadores. | `Plate`, `plate` |
| Ordem de Serviço (OS) | Registro do atendimento de um veículo, incluindo diagnóstico, itens, orçamento, estado e histórico até a entrega. | `WorkOrder`, `/work-orders` |
| Diagnóstico | Avaliação técnica registrada pela oficina para orientar os serviços e peças necessários. | `diagnosis`, `SetDiagnosis` |
| Serviço | Trabalho oferecido pela oficina, com preço de venda e duração estimada. | `Service`, `service` |
| Peça / insumo | Material consumível ou componente controlado em estoque e que pode compor o orçamento de uma OS. | `Part`, `part` |
| Item da OS | Snapshot de serviço ou peça incluído em uma OS, com quantidade e preço copiado do catálogo no momento da inclusão. | `Item`, `service` ou `part` |
| Catálogo | Conjunto de serviços e peças ativos que podem ser selecionados para uma OS. | módulo `servicecatalog` |
| Orçamento | Versão dos itens e valores propostos ao cliente para execução dos serviços. Cada submissão mantém uma versão e total próprios. | `Quote`, `quote_version` |
| Versão do orçamento | Sequência imutável de uma proposta submetida. Uma nova submissão após rejeição cria outra versão. | `Quote.Version` |
| Decisão do orçamento | Aprovação ou rejeição da versão vigente pelo cliente. | `Quote.Decision`: `pending`, `approved`, `rejected` |
| Estoque físico | Quantidade total da peça registrada na oficina, incluindo o que está reservado. | `stock` |
| Estoque reservado | Parte do estoque físico comprometida com orçamentos aprovados e ainda não consumida. | `reserved_stock` |
| Estoque disponível | Quantidade física que ainda pode ser usada em novas saídas ou reservas. | `available_stock = stock - reserved_stock` |
| Movimentação de estoque | Registro de entrada, saída, reserva ou consumo que explica alterações de saldo. | `StockMovement`, `entry`, `exit` e movimentos de OS |
| Reserva | Compromisso de peças para uma OS após aprovação do orçamento; reduz o disponível sem reduzir inicialmente o físico. | `ReserveStock`, reserva associada à OS |
| Consumo | Uso de peça reservada durante a execução; reduz o físico e a reserva. | `ConsumeReservedStock`, `ConsumedQuantity` |
| Histórico da OS | Registro append-only, pela API normal, das mudanças de status e fatos operacionais da OS, com instante e ator. | `HistoryEntry`, `work_order_status_history` |
| Ator | Usuário administrativo ou cliente responsável por iniciar uma ação registrada. | `user_id` ou `client_id` |
| Acompanhamento | Consulta autorizada do status, orçamento e histórico de uma OS pelo cliente. | `/api/v1/client/work-orders` |

## Vocabulário dos status

| Valor persistido | Termo apresentado | Significado |
|---|---|---|
| `received` | Recebida | OS aberta e veículo recebido para atendimento. |
| `diagnosis` | Em diagnóstico | A oficina está avaliando ou revisando o que deve ser feito. |
| `awaiting_approval` | Aguardando aprovação | Há um orçamento vigente esperando decisão do cliente. |
| `in_progress` | Em execução | O orçamento foi aprovado e o trabalho pode ser realizado. |
| `completed` | Finalizada | O serviço foi concluído; todas as peças reservadas da OS foram consumidas. |
| `delivered` | Entregue | O veículo foi entregue ao cliente após a finalização. |

Rejeitar um orçamento não cria um status permanente: a decisão fica registrada na versão do orçamento e a OS retorna a `diagnosis` para revisão e eventual nova submissão.

## Frases de negócio

Use estas formulações de forma consistente em conversas, documentação, API e código:

- “Abrir uma OS para um cliente e um veículo.”
- “Registrar o diagnóstico da OS.”
- “Adicionar um serviço ou uma peça à OS usando os dados atuais do catálogo.”
- “Submeter uma nova versão do orçamento para aprovação.”
- “O cliente aprova ou rejeita a versão vigente do orçamento.”
- “A aprovação reserva as peças necessárias para a OS.”
- “A oficina registra o consumo das peças durante a execução.”
- “A OS só pode ser finalizada depois do consumo integral das peças reservadas.”
- “O cliente acompanha somente as OS dentro do escopo da sua sessão.”
- “O veículo é entregue depois que a OS está finalizada.”

## Contextos delimitados

Os contextos abaixo são os módulos do monólito e compartilham identificadores por interfaces de aplicação; não são serviços independentes.

| Contexto | Responsabilidade | Conceitos principais |
|---|---|---|
| Clientes (`customer`) | Identidade, documento, contato, status e credenciais do cliente. | Cliente, Documento |
| Veículos (`vehicle`) | Identificação do veículo e vínculo com cliente. | Veículo, Placa |
| Catálogo e estoque (`servicecatalog`) | Serviços, peças, preços, saldo e movimentações de estoque. | Serviço, Peça, Estoque, Movimentação |
| Ordens de serviço (`workorder`) | Atendimento, diagnóstico, itens, orçamento, decisão, ciclo de vida e histórico. | OS, Item da OS, Orçamento, Histórico |
| Identidade administrativa (`identity`) | Usuários administrativos, autenticação JWT, perfis e permissões. | Usuário, Perfil, Permissão |

## Agregados, entidades e objetos de valor

| Conceito | Classificação no modelo | Responsabilidade / composição |
|---|---|---|
| `WorkOrder` | Agregado raiz e entidade | Protege status, diagnóstico, itens, versões de orçamento e regras do ciclo da OS. |
| `Item` | Entidade interna da OS | Snapshot de serviço/peça, quantidade, preço unitário e quantidade consumida. |
| `Quote` | Entidade interna da OS | Versão, total, itens congelados e decisão do cliente. |
| `HistoryEntry` | Registro associado à OS | Fato operacional/status, ator, nota e instante. |
| `Customer` | Agregado raiz | Identidade e dados de contato; documento é validado pelo objeto de domínio `Document`. |
| `Vehicle` | Agregado raiz | Dados do veículo e referência ao cliente; placa é validada pelo objeto de domínio `Plate`. |
| `Service` | Agregado raiz do catálogo | Código, descrição, preço, duração estimada e situação ativa. |
| `Part` | Agregado raiz do catálogo/estoque | Identificação, unidade, preços, saldo físico, reserva e situação ativa. |
| `StockMovement` | Registro de movimentação | Tipo, quantidade, peça e ator; preserva a explicação de alteração de saldo. |
| `Document` e `Plate` | Objetos de valor | Validam e normalizam identificadores; não têm identidade própria. |
| Dinheiro em centavos | Tipo conceitual / convenção | Valores são inteiros (`int64`) em centavos e nunca `float`. |

**Nota sobre o modelo:** o código implementa `WorkOrder` como agregado com métodos que protegem transições e invariantes. As fronteiras transacionais também envolvem estoque, coordenadas pelo caso de uso de aplicação; isso não transforma catálogo/estoque em parte do agregado `WorkOrder`.

## Eventos e comandos

Os fatos registrados pelo domínio incluem: OS recebida, diagnóstico registrado, item incluído/removido, orçamento submetido, orçamento aprovado/rejeitado, peças consumidas, OS finalizada e veículo entregue. No estoque, são registrados movimentos de entrada, saída, reserva e consumo. Estes são fatos persistidos em histórico/movimentações; **não são eventos publicados para consumidores externos** na implementação atual.
