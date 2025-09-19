package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type CotacaoResponse struct {
	Value []struct {
		CotacaoCompra float64 `json:"cotacaoCompra"`
		CotacaoVenda  float64 `json:"cotacaoVenda"`
		DataHora      string  `json:"dataHoraCotacao"`
	} `json:"value"`
}

// apiBaseURL pode ser sobrescrito em testes
var apiBaseURL = "https://olinda.bcb.gov.br/olinda/servico/PTAX/versao/v1/odata/"

// buildBCBURL constrói a URL correta para o BCB dependendo da moeda.
// Para USD usa CotacaoDolarPeriodo com formato MM-DD-YYYY (01-02-2006).
// Para outras moedas usa CotacaoMoedaAberturaOuIntermediario com formato MM-DD-YYYY (01-02-2006)
func buildBCBURL(baseURL, currency string, tryDate time.Time) string {
	if strings.ToUpper(currency) == "USD" {
		// Usar formato MM-DD-YYYY conforme exemplo fornecido pelo usuário
		return fmt.Sprintf(baseURL+"CotacaoDolarPeriodo(dataInicial=@dataInicial,dataFinalCotacao=@dataFinalCotacao)?@dataInicial='%s'&@dataFinalCotacao='%s'&$top=100&$format=json&$select=cotacaoCompra,cotacaoVenda,dataHoraCotacao",
			tryDate.Format("01-02-2006"), tryDate.Format("01-02-2006"))
	}
	// moeda diferente de USD
	return fmt.Sprintf(baseURL+"CotacaoMoedaAberturaOuIntermediario(codigoMoeda=@codigoMoeda,dataCotacao=@dataCotacao)?@codigoMoeda='%s'&@dataCotacao='%s'&$format=json&$select=cotacaoCompra,cotacaoVenda,dataHoraCotacao,tipoBoletim",
		strings.ToUpper(currency), tryDate.Format("01-02-2006"))
}

// GetUSDRateWithConfigAndCurrency tenta obter a cotação para a data fornecida.
// Se o serviço do BCB devolver um 5xx a função tentará datas anteriores até maxBackDays.
// timeoutSeconds e maxRetries controlam o cliente e tentativas para cada data.
func GetUSDRateWithConfigAndCurrency(date time.Time, maxBackDays int, timeoutSeconds int, maxRetries int, currency string) (float64, float64, time.Time, error) {
	client := &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second}

	// normalize apiBaseURL to always end with '/'
	baseURL := strings.TrimRight(apiBaseURL, "/") + "/"

	for i := 0; i <= maxBackDays; i++ {
		tryDate := date.AddDate(0, 0, -i)
		url := buildBCBURL(baseURL, strings.ToUpper(currency), tryDate)
		fmt.Printf("Consultando URL: %s\n", url)

		var resp *http.Response
		var err error
		for attempt := 0; attempt <= maxRetries; attempt++ {
			resp, err = client.Get(url)
			if err != nil {
				return 0, 0, time.Time{}, err
			}

			if resp.StatusCode == http.StatusOK {
				break
			}

			// handle 5xx with retries (per-date)
			if resp.StatusCode >= 500 && attempt < maxRetries {
				bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
				resp.Body.Close()
				snippet := strings.TrimSpace(string(bodySnippet))
				if len(snippet) > 200 {
					snippet = snippet[:200]
				}
				wait := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				fmt.Printf("BCB retornou %d para %s (attempt %d/%d) — retry em %s; body=%q\n", resp.StatusCode, tryDate.Format("01-02-2006"), attempt+1, maxRetries, wait, snippet)
				time.Sleep(wait)
				continue
			}

			if resp.StatusCode >= 500 {
				bodySnippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
				resp.Body.Close()
				snippet := strings.TrimSpace(string(bodySnippet))
				if len(snippet) > 200 {
					snippet = snippet[:200]
				}
				fmt.Printf("BCB retornou %d para %s — tentando dia anterior; body=%q\n", resp.StatusCode, tryDate.Format("01-02-2006"), snippet)
				break
			}

			return 0, 0, time.Time{}, fmt.Errorf("status %d", resp.StatusCode)
		}

		if resp == nil {
			return 0, 0, time.Time{}, fmt.Errorf("sem resposta do servidor")
		}

		if resp.StatusCode >= 500 {
			resp.Body.Close()
			continue
		}

		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return 0, 0, time.Time{}, readErr
		}

		ct := resp.Header.Get("Content-Type")
		trimmed := strings.TrimSpace(string(bodyBytes))
		if !(strings.Contains(ct, "application/json") || strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[")) {
			snippet := trimmed
			if len(snippet) > 200 {
				snippet = snippet[:200]
			}
			return 0, 0, time.Time{}, fmt.Errorf("esperado json, recebido Content-Type=%s, corpo=%q", ct, snippet)
		}

		// remove possible /* ... */ wrapper
		cleaned := strings.TrimSpace(string(bodyBytes))
		if strings.HasPrefix(cleaned, "/*") && strings.HasSuffix(cleaned, "*/") {
			cleaned = strings.TrimPrefix(cleaned, "/*")
			cleaned = strings.TrimSuffix(cleaned, "*/")
			cleaned = strings.TrimSpace(cleaned)
		}

		var data CotacaoResponse
		if err := json.Unmarshal([]byte(cleaned), &data); err != nil {
			return 0, 0, time.Time{}, err
		}

		if len(data.Value) == 0 {
			if i < maxBackDays {
				fmt.Printf("Nenhuma cotação encontrada para %s — tentando dia anterior\n", tryDate.Format("01-02-2006"))
				continue
			}
			return 0, 0, time.Time{}, fmt.Errorf("nenhuma cotação encontrada para %s", tryDate.Format("01-02-2006"))
		}

		return data.Value[0].CotacaoCompra, data.Value[0].CotacaoVenda, tryDate, nil
	}

	return 0, 0, time.Time{}, fmt.Errorf("nenhuma cotação encontrada nos últimos %d dias", maxBackDays)
}

func main() {
	// flags com defaults
	defTimeout := 10
	defRetries := 3
	defBackDays := 5

	timeoutSeconds := defTimeout
	maxRetries := defRetries
	maxBackDays := defBackDays

	// ler de variáveis de ambiente, se presentes
	if v := os.Getenv("BCB_API_BASE_URL"); v != "" {
		apiBaseURL = v
	}
	if v := os.Getenv("BCB_TIMEOUT_SECONDS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			timeoutSeconds = parsed
		}
	}
	if v := os.Getenv("BCB_MAX_RETRIES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			maxRetries = parsed
		}
	}
	if v := os.Getenv("BCB_MAX_BACK_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			maxBackDays = parsed
		}
	}

	// flags (sobrepõem variáveis de ambiente)
	flag.IntVar(&timeoutSeconds, "timeout", timeoutSeconds, "timeout em segundos para requisições HTTP")
	flag.IntVar(&maxRetries, "retries", maxRetries, "número máximo de tentativas por data (retries 5xx)")
	flag.IntVar(&maxBackDays, "backdays", maxBackDays, "dias máximos para tentar data retroativa")
	flag.StringVar(&apiBaseURL, "api", apiBaseURL, "base URL da API do BCB")
	var currency string
	flag.StringVar(&currency, "currency", "USD", "código da moeda (ex: USD, EUR)")

	// flags adicionais
	var price string
	flag.StringVar(&price, "price", "venda", "qual preço retornar: compra|venda|both")

	// consultar N dias atrás explicitamente (ex: -daysago=3)
	var daysAgo int
	flag.IntVar(&daysAgo, "daysago", 0, "consultar a cotação de N dias atrás (0 = hoje)")

	// parse após definir todas as flags
	flag.Parse()

	// determina a data base (hoje ou N dias atrás) e tenta a cotação com as configurações fornecidas
	baseDate := time.Now().AddDate(0, 0, -daysAgo)
	compra, venda, usedDate, err := GetUSDRateWithConfigAndCurrency(baseDate, maxBackDays, timeoutSeconds, maxRetries, currency)
	if err != nil {
		fmt.Printf("erro ao consultar o endpoint: %v\n", err)
		return
	}

	switch strings.ToLower(price) {
	case "compra":
		fmt.Printf("Cotação %s (compra) para %s = %.4f\n", strings.ToUpper(currency), usedDate.Format("01-02-2006"), compra)
	case "both":
		fmt.Printf("Cotação %s para %s - compra=%.4f venda=%.4f\n", strings.ToUpper(currency), usedDate.Format("01-02-2006"), compra, venda)
	default:
		fmt.Printf("Cotação %s (venda) para %s = %.4f\n", strings.ToUpper(currency), usedDate.Format("01-02-2006"), venda)
	}
}
