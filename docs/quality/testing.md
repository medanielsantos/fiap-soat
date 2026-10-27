# Estratégia de testes

## Unitários

Testar value objects e regras sem HTTP ou PostgreSQL: CPF/CNPJ, placa, dinheiro em centavos, cálculo de orçamento, transições da OS, aprovação e invariantes de estoque.

## Convenção de nomes

Os cenários devem usar estes padrões em `snake_case`:

- Positivo: `test_it_should_<expected_behavior>_when_<condition>`.
- Negativo: `test_it_should_not_<behavior>_when_<condition>`.

Como o `go test` descobre funções externas pelo prefixo `Test`, a função Go usa `Test<Subject>` e cada cenário nomeado segue um dos padrões acima em `t.Run`:

```go
func TestWorkOrderApproval(t *testing.T) {
	t.Run("test_it_should_start_work_when_quote_is_approved", func(t *testing.T) {
		// arrange, act, assert
	})
	t.Run("test_it_should_not_start_work_when_quote_is_not_approved", func(t *testing.T) {
		// arrange, act, assert
	})
}
```

Escrever o comportamento esperado e a condição relevante; evitar nomes vagos como `success`, `valid_case` ou `test_1`.

## Integração

Com PostgreSQL, validar migrações, repositórios, constraints, rollback e concorrência. Validar handlers Echo, autorização JWT e respostas RFC 9457 sem stack traces nem dados sensíveis. Se ADR-011 for aceita, incluir testes de integração de TTL/revogação de credenciais temporárias e rate limiting no Redis.

## Onde organizar os testes

Não replicar toda a árvore de `internal/` sob `tests/`. Manter testes unitários junto do pacote que exercitam, em arquivos `*_test.go`; isso facilita testar regras internas e manter o teste perto do código. Começar com um arquivo de teste por pacote ou responsabilidade coesa, agrupando vários cenários relacionados nele. Não criar um arquivo por função ou por caso de teste; dividir apenas quando o arquivo ficar difícil de navegar ou misturar responsabilidades diferentes. Usar `tests/integration/` para poucos fluxos que cruzam módulos, banco e HTTP, agrupados por capacidade em vez de copiar a estrutura de produção.

```text
internal/
  work-order/
    domain/
      work_order.go
      work_order_test.go          # testes unitários do pacote
tests/
  integration/
    work_order_test.go           # principais fluxos da OS com HTTP + PostgreSQL
    catalog_inventory_test.go    # catálogo e estoque em integração
```

Testes de integração devem verificar comportamento observável pelos contratos públicos. Se um teste precisar validar detalhe privado de um pacote, ele provavelmente deve ficar junto daquele pacote como teste unitário. Agrupar cenários com `t.Run` usando os nomes `test_it_should_...` e `test_it_should_not_...` para manter poucos arquivos sem perder clareza.

## Fluxo principal

Cobrir abertura de OS, vínculo cliente/veículo, diagnóstico, inclusão de serviços/peças, geração e aprovação/rejeição de orçamento, atualização de status, movimentação de estoque, consulta do cliente e entrega.

## Cobertura

Meta exigida: mínimo de 80% nos domínios críticos (`work-order` e `service-catalog`). Publicar resultado por pacote e revisar lacunas; não substituir testes de comportamento por cobertura numérica.

Os comandos de execução serão definidos quando as dependências e a aplicação forem implementadas.
