# ADR-003 — Arquitetura

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

O escopo exige um back-end monolítico, mas contém capacidades distintas para clientes, veículos, catálogo/estoque, OS e identidade. As regras de negócio precisam ser testáveis sem infraestrutura.

## Decisão

Adotar **monólito modular com DDD e Clean Architecture**. Separar módulos por contexto e camadas `domain`, `application` e adaptadores. A dependência aponta para o domínio; `cmd/api` compõe implementações concretas.

## Alternativas consideradas

- Monólito em camadas técnicas globais: simples inicialmente, mas tende a misturar regras de contextos distintos.
- Microsserviços: adicionam rede, deploy e consistência distribuída sem necessidade para o MVP.

## Consequências

- `customer` e `vehicle` permanecem módulos separados.
- Módulos cooperam por IDs e portas/DTOs públicos, não por entidades internas.
- Sem comunicação remota entre módulos no MVP.
- Helpers compartilhados ficam em `internal/utils` apenas quando forem genéricos; regras de negócio ficam no módulo dono.
