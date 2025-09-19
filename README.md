# Banco Central - Cotação (POC)

Este POC consulta serviços PTAX do Banco Central (BCB) para obter cotações de moedas.

Funcionalidades principais

- Consulta por período para USD usando `CotacaoDolarPeriodo`.

- Consulta para outras moedas (ex: EUR) usando `CotacaoMoedaAberturaOuIntermediario`.

- Retries exponenciais em respostas 5xx e fallback para dias anteriores.

- Validação do corpo de resposta (garante JSON) e mensagens de erro úteis.


Flags e variáveis de ambiente

- `-currency` (flag) — código da moeda (ex: USD, EUR). Default: `USD`.

  Exemplo: `-currency=EUR` — usa o endpoint `CotacaoMoedaAberturaOuIntermediario`.

- `-price` (flag) — qual preço mostrar: `compra` | `venda` | `both`. Default: `venda`.

  Exemplo: `-price=both` → exibe compra e venda.

- `-timeout` (flag) — timeout em segundos para requisições HTTP. Default: `10`.

  Exemplo: `-timeout=5` — reduz o timeout para 5s (útil em testes).

- `-retries` (flag) — tentativas por data para 5xx. Default: `3`.

  Exemplo: `-retries=1` — fará 1 retry exponencial antes de falhar na mesma data.

- `-backdays` (flag) — número de dias para tentar datas anteriores. Default: `5`.

  Exemplo: `-backdays=2` — se o serviço não tiver dados para hoje, tenta até 2 dias anteriores.

- `-api` (flag) — base URL da API do BCB (útil para testes).

  Exemplo: `-api=http://localhost:8080/` — útil para apontar para um servidor `httptest`.

Variáveis de ambiente equivalentes (usadas se flags não fornecidas)

- `BCB_API_BASE_URL`
- `BCB_TIMEOUT_SECONDS`
- `BCB_MAX_RETRIES`
- `BCB_MAX_BACK_DAYS`


Exemplos de uso

```bash
# obter venda do EUR (padrão)
go run main.go -currency=EUR

# obter ambos os preços do EUR
go run main.go -currency=EUR -price=both

# reduzir timeout e aceitar somente 1 retry
go run main.go -currency=EUR -timeout=5 -retries=1

# tentar datas retroativas (ex.: 3 dias)
go run main.go -currency=EUR -backdays=3

# apontar para um servidor local (útil para testes com httptest)
go run main.go -api=http://localhost:8080/ -currency=EUR

# configurar via env (variáveis substituem defaults, flags sobrepõem env)
export BCB_TIMEOUT_SECONDS=8
export BCB_MAX_RETRIES=2
go run main.go -currency=EUR -backdays=3

# rodar testes
go test ./...
```

Notas

- O formato de data usado nas URLs é `MM-DD-YYYY` (ex: `09-18-2025`) para compatibilidade com exemplos do serviço.

- Em caso de erro 500 o serviço do BCB pode retornar um corpo não-JSON; o POC registra um snippet desse corpo para ajudar no diagnóstico.

Contribuições

- Pull requests com melhorias (mais opções, logs, CI) são bem-vindos.
