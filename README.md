# Banco Central - Cotação (POC)

Este POC consulta os serviços PTAX do Banco Central (BCB) para obter cotações de moedas.

Visão geral

- USD: usa o endpoint `CotacaoDolarPeriodo` (consulta por período).
- Outras moedas: usa `CotacaoMoedaAberturaOuIntermediario`.
- Retries exponenciais em respostas 5xx e fallback para dias anteriores quando não há dado disponível.
- O cliente valida o corpo de resposta e faz "unwrapping" de respostas comentadas do tipo `/* ... */` antes de decodificar JSON.


Flags e variáveis de ambiente

Flags (default entre parênteses):

- `-currency` (USD) — código da moeda (ex: USD, EUR).
- `-price` (venda) — qual preço retornar: `compra` | `venda` | `both`.
- `-timeout` (10) — timeout em segundos para requisições HTTP.
- `-retries` (3) — número máximo de tentativas por data ao receber 5xx.
- `-daysago` (0) — consultar a cotação de N dias atrás explicitamente (0 = hoje). Ex: `-daysago=3` consulta a cotação de 3 dias atrás.
- `-api` — base URL da API do BCB (útil para testes locais).

Variáveis de ambiente (usadas apenas quando a flag correspondente não for fornecida):

- `BCB_API_BASE_URL` — sobrescreve a base URL padrão.
- `BCB_TIMEOUT_SECONDS` — sobrescreve `-timeout`.
- `BCB_MAX_RETRIES` — sobrescreve `-retries`.

Exemplos de uso

```bash
# obter venda do EUR (padrão: venda)
go run main.go -currency=EUR

# obter compra do EUR
go run main.go -currency=EUR -price=compra

# obter ambos os preços do EUR
go run main.go -currency=EUR -price=both

# reduzir timeout e aceitar somente 1 retry
go run main.go -currency=EUR -timeout=5 -retries=1

# consultar data retroativa (ex.: dias atrás = 3)
go run main.go -currency=EUR -daysago=3

# apontar para um servidor local (útil para testes com httptest)
go run main.go -api=http://localhost:8080/ -currency=EUR

# configurar via env (variáveis substituem defaults, flags sobrepõem env)
export BCB_TIMEOUT_SECONDS=8
export BCB_MAX_RETRIES=2
go run main.go -currency=EUR -daysago=3

# rodar testes
go test ./...
```

Exemplo de saída

```text
# saída quando -price=venda (padrão)
Cotação EUR (venda) para 09-15-2025 = 4.5621

# saída quando -price=compra
Cotação EUR (compra) para 09-15-2025 = 4.5432

# saída quando -price=both
Cotação EUR para 09-15-2025 - compra=4.5432 venda=4.5621
```

Notas importantes

- O formato de data usado nas URLs é `MM-DD-YYYY` (ex: `09-18-2025`). O campo `-daysago` aceita um inteiro que determina a data base (0 = hoje).
- O serviço do BCB por vezes devolve um corpo não-JSON (ou JSON dentro de comentários `/* ... */`). Este POC tenta limpar esse wrapper antes de decodificar o JSON.
- Em respostas 5xx são feitas tentativas exponenciais por data; se não houver dado para a data consultada, o cliente tentará a data anterior até um limite configurável.

Contribuições

Pull requests com melhorias (mais opções, logs, CI) são bem-vindos.
