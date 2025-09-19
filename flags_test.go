package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// helper para criar um servidor que responde com a sequência de códigos fornecida
func makeSeqServer(statusSeq []int) *httptest.Server {
	idx := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var status int
		if idx < len(statusSeq) {
			status = statusSeq[idx]
		} else {
			status = statusSeq[len(statusSeq)-1]
		}
		idx++

		if status != 200 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(status)
			_, _ = w.Write([]byte("/* server error */"))
			return
		}

		resp := map[string]interface{}{
			"value": []map[string]interface{}{{
				"cotacaoCompra": 3.21,
				"cotacaoVenda":  3.45,
			}},
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(b)
	}))
}

func TestFlagsCombinations(t *testing.T) {
	bin := filepath.Join(os.TempDir(), "bcb_flags_test_bin")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v, out=%s", err, string(out))
	}
	defer os.Remove(bin)

	cases := []struct {
		name      string
		statusSeq []int
		args      []string
		want      string
	}{
		// single 200 -> should return venda value
		{"simple success", []int{200}, []string{"-timeout", "5", "-retries", "0", "-backdays", "0", "-currency", "EUR", "-price", "venda"}, "3.45"},
		// first 500 then 200, with retries=1 should succeed
		{"with retries", []int{500, 200}, []string{"-timeout", "5", "-retries", "1", "-backdays", "0", "-currency", "EUR", "-price", "compra"}, "3.21"},
		// first 500 then 200, with retries=0 but backdays=1 should succeed via fallback
		{"with fallback", []int{500, 200}, []string{"-timeout", "5", "-retries", "0", "-backdays", "1", "-currency", "EUR", "-price", "both"}, "3.21"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ts := makeSeqServer(c.statusSeq)
			defer ts.Close()

			args := append([]string{"-api", ts.URL}, c.args...)
			cmd := exec.Command(bin, args...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			cmd.Env = append(os.Environ(), "LANG=C", "TZ=UTC")

			done := make(chan error)
			go func() { done <- cmd.Run() }()

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("process failed: %v, stderr=%s", err, stderr.String())
				}
			case <-time.After(6 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatalf("process timed out")
			}

			out := stdout.String() + "\n" + stderr.String()
			if !strings.Contains(out, c.want) {
				t.Fatalf("expected output to contain %q; got: %s", c.want, out)
			}
		})
	}
}
