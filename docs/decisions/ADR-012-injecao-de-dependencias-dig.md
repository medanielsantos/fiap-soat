# ADR-012 — Injeção de dependências com Uber Dig

- **Estado:** Aceito
- **Data:** 2026-10-25

## Contexto

O monólito tem diversos módulos e adaptadores. A composição manual da árvore de dependências exige manter a ordem de construção à mão; ao mesmo tempo, o projeto não precisa de um framework completo de ciclo de vida nem geração de código.

## Decisão

Usar [`go.uber.org/dig`](https://github.com/uber-go/dig) para registrar e resolver o grafo de dependências durante o bootstrap.

- Grafo formado a partir das assinaturas dos construtores.
- Resolução lazy: somente construtores necessários ao `Invoke` são executados.
- Uma instância de cada provider/tipo é reutilizada dentro do mesmo container.
- Construtores permanecem funções Go comuns `NewX(deps...) IX`.
- Somente `internal/containers/` e `cmd/api/main.go` podem importar Dig. Domínio, aplicação e adaptadores não recebem container nem dependem de Dig.
- `Invoke` da raiz acontece no startup; erro de dependência impede a aplicação de iniciar.

## Bootstrap

```text
main()
 ├─ carregar .env local (opcional; fora de container)
 ├─ config.Boot()                  # env → configuração tipada
 ├─ containers.Boot(config)        # registra providers no Dig
 └─ c.Invoke(func(runtime *Runtime) error {
        defer runtime.Shutdown()   # flush e fechamento explícito
        return runtime.Run(args)  # escolhe o modo de execução
    })
```

O runtime controla explicitamente a sequência de boot: aplicação, banco de dados, feature flags e módulos condicionais habilitados. Se uma feature flag depender do banco, inicializar o DB antes de carregar as flags; providers opcionais só são disponibilizados/acionados depois que a decisão da flag estiver conhecida.

## Alternativas consideradas

- **Injeção manual:** sem dependência externa e erros visíveis em compilação, mas exige manter a ordem do grafo manualmente.
- **Uber Fx:** adiciona `OnStart`/`OnStop` e ciclo de vida integrado; não necessário para o escopo atual, que terá shutdown explícito.
- **Wire:** gera código e detecta grafo em tempo de compilação; não escolhido para manter o bootstrap sem etapa de geração.

## Consequências

- Dig resolve o grafo em runtime, portanto erros de wiring são detectados no startup, não pelo compilador.
- Dig não gerencia `OnStart`/`OnStop`; servidor, banco e telemetria precisam de shutdown ordenado explícito no runtime.
- Componentes são singleton por container/provider; não usar estado mutável compartilhado sem sincronização.
- O container não deve escapar do composition root como service locator.
