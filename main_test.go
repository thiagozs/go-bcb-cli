package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetUSDRateWithConfig_Success(t *testing.T) {
	// servidor que retorna JSON válido
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value":[{"cotacaoCompra":5.2,"cotacaoVenda":5.301,"dataHoraCotacao":"2025-09-18T12:00:00"}]}`))
	}))
	defer srv.Close()

	apiBaseURL = srv.URL + "/"
	compra, venda, usedDate, err := GetUSDRateWithConfigAndCurrency(time.Date(2025, 9, 18, 0, 0, 0, 0, time.UTC), 0, 5, 1, "USD")
	if err != nil {
		t.Fatalf("esperava sucesso, erro: %v", err)
	}
	if venda != 5.301 {
		t.Fatalf("valor errado: %v", venda)
	}
	_ = compra
	_ = usedDate
}

func TestGetUSDRateWithConfig_Server500(t *testing.T) {
	// servidor que retorna 500 com corpo não-json
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("/*{\n  \"codigo\" : 500,\n  \"mensagem\" : \"Erro desconhecido\"\n}*/"))
	}))
	defer srv.Close()

	apiBaseURL = srv.URL + "/"
	_, _, _, err := GetUSDRateWithConfigAndCurrency(time.Date(2025, 9, 18, 0, 0, 0, 0, time.UTC), 0, 2, 1, "USD")
	if err == nil {
		t.Fatalf("esperava erro quando servidor retorna 500")
	}
}

func TestGetUSDRateWithConfigAndCurrency_EUR(t *testing.T) {
	// servidor que retorna JSON válido no formato esperado para CotacaoMoedaAberturaOuIntermediario
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value":[{"cotacaoCompra":5.4,"cotacaoVenda":5.5,"dataHoraCotacao":"2025-09-18T12:00:00","tipoBoletim":"A"}]}`))
	}))
	defer srv.Close()

	apiBaseURL = srv.URL + "/"
	compra, venda, _, err := GetUSDRateWithConfigAndCurrency(time.Date(2025, 9, 18, 0, 0, 0, 0, time.UTC), 0, 5, 1, "EUR")
	if err != nil {
		t.Fatalf("esperava sucesso, erro: %v", err)
	}
	if venda != 5.5 {
		t.Fatalf("valor errado para EUR venda: %v", venda)
	}
	_ = compra
}

func TestCleanCommentedJSON(t *testing.T) {
	// servidor que retorna JSON válido, mas envolvido em comentários /* ... */
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("/*{" +
			"\"value\":[{\"cotacaoCompra\":6.1,\"cotacaoVenda\":6.2,\"dataHoraCotacao\":\"2025-09-18T12:00:00\"}] }*/"))
	}))
	defer srv.Close()

	apiBaseURL = srv.URL + "/"
	compra, venda, _, err := GetUSDRateWithConfigAndCurrency(time.Date(2025, 9, 18, 0, 0, 0, 0, time.UTC), 0, 5, 1, "USD")
	if err != nil {
		t.Fatalf("esperava sucesso, erro: %v", err)
	}
	if venda != 6.2 {
		t.Fatalf("valor errado: %v", venda)
	}
	_ = compra
}
