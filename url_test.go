package main

import (
	"testing"
	"time"
)

func TestBuildBCBURL_USD(t *testing.T) {
	base := "https://olinda.bcb.gov.br/olinda/servico/PTAX/versao/v1/odata/"
	date, _ := time.Parse("2006-01-02", "2025-09-18")
	u := buildBCBURL(base, "USD", date)
	want := "CotacaoDolarPeriodo(dataInicial=@dataInicial,dataFinalCotacao=@dataFinalCotacao)?@dataInicial='09-18-2025'&@dataFinalCotacao='09-18-2025'"
	if !contains(u, want) {
		t.Fatalf("url não contém expected fragment. got=%s want fragment=%s", u, want)
	}
}

func TestBuildBCBURL_EUR(t *testing.T) {
	base := "https://olinda.bcb.gov.br/olinda/servico/PTAX/versao/v1/odata/"
	date, _ := time.Parse("2006-01-02", "2025-09-18")
	u := buildBCBURL(base, "EUR", date)
	want := "CotacaoMoedaAberturaOuIntermediario(codigoMoeda=@codigoMoeda,dataCotacao=@dataCotacao)?@codigoMoeda='EUR'&@dataCotacao='09-18-2025'"
	if !contains(u, want) {
		t.Fatalf("url não contém expected fragment. got=%s want fragment=%s", u, want)
	}
}

// small helper to avoid importing strings repeatedly
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
