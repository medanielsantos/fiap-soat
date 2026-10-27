# ADR-008 — Controle de estoque relacionado à OS

- **Estado:** Proposto — confirmar momento da reserva/baixa
- **Data:** 2026-10-25

## Contexto

Uma OS pode incluir peças e insumos. O estoque deve permanecer consistente quando o cliente aprova o orçamento e a oficina inicia execução.

## Decisão proposta

Ao aprovar o orçamento, reservar as quantidades da OS na mesma transação que registra aprovação e muda a OS para Em execução. A reserva reduz disponibilidade, mas não o saldo físico. No início efetivo do consumo ou conclusão, converter reserva em saída/consumo. Operações devem ser idempotentes e impedir saldo disponível negativo.

## Alternativas consideradas

- Baixar estoque ao montar orçamento: inadequado, pois cliente pode rejeitar.
- Baixar somente ao concluir: permite que duas OS aprovem as mesmas últimas unidades.
- Reservar na aprovação: mantém disponibilidade e histórico, com maior complexidade de estados/movimentos.

## Consequências

- Registrar movimentações imutáveis e saldo/reservas com concorrência controlada no PostgreSQL.
- Rejeição antes de aprovação não movimenta estoque.
- Cancelamento após aprovação precisa liberar reserva; fluxo de cancelamento permanece fora do escopo inicial até ser definido.
- **Pendente:** confirmar se MVP implementa reserva e consumo separados ou simplifica para baixa atômica na aprovação.
