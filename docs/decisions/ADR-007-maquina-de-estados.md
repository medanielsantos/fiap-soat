# ADR-007 — Máquina de estados da OS

- **Estado:** Proposto — validar regra de orçamento rejeitado
- **Data:** 2026-10-25

## Contexto

O sistema precisa controlar as etapas obrigatórias da OS e impedir mudanças arbitrárias de status.

## Decisão proposta

Representar status como enumeração do domínio e permitir apenas as transições:

| De | Ação | Para |
|---|---|---|
| Recebida | Iniciar diagnóstico | Em diagnóstico |
| Em diagnóstico | Enviar orçamento | Aguardando aprovação |
| Aguardando aprovação | Cliente aprova | Em execução |
| Em execução | Finalizar serviço | Finalizada |
| Finalizada | Registrar entrega | Entregue |

Rejeição de orçamento deve ser registrada como decisão/histórico. **Proposta:** retornar OS para Em diagnóstico para revisão e emissão de nova versão do orçamento. Transições inválidas retornam conflito e não alteram a OS.

## Alternativas consideradas

- Status livre editável pelo operador: permite estados impossíveis e elimina garantias de fluxo.
- Adicionar status `REJECTED` permanente: não permite revisar e reenviar orçamento sem reabrir OS.

## Consequências

- Cada transição registra instante, ator e origem/destino.
- Mudanças de status são realizadas por casos de uso específicos, não por CRUD genérico.
- A resposta HTTP usa `409` para conflito de estado.
- **Pendente:** confirmar retorno a diagnóstico após rejeição e se haverá cancelamento de OS no MVP.
