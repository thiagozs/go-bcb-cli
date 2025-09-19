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

// Test the -price flag by running the compiled binary with different flags
func TestPriceFlagOutputs(t *testing.T) {
	// Start a test server that mimics the BCB response for a currency with compra and venda
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"value": []map[string]interface{}{{
				"cotacaoCompra": 4.12,
				"cotacaoVenda":  4.34,
			}},
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	// Build the binary
	bin := filepath.Join(os.TempDir(), "bcb_test_bin")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build test binary: %v, out=%s", err, string(out))
	}
	defer os.Remove(bin)

	cases := []struct {
		priceFlag  string
		wantCompra string
		wantVenda  string
	}{
		{"compra", "4.1200", ""},
		{"venda", "", "4.3400"},
		{"both", "4.1200", "4.3400"},
	}

	for _, c := range cases {
		t.Run(c.priceFlag, func(t *testing.T) {
			cmd := exec.Command(bin, "-api", ts.URL, "-currency", "EUR", "-price", c.priceFlag, "-timeout", "2", "-retries", "1", "-backdays", "0")
			// ensure deterministic output by setting LANG and TZ
			cmd.Env = append(os.Environ(), "LANG=C", "TZ=UTC")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			// give it a timeout in case something hangs
			done := make(chan error)
			go func() {
				done <- cmd.Run()
			}()
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("process failed: %v, stderr=%s", err, stderr.String())
				}
			case <-time.After(5 * time.Second):
				// try to kill
				_ = cmd.Process.Kill()
				t.Fatalf("process timed out")
			}

			combined := stdout.String() + "\n" + stderr.String()
			if c.wantCompra != "" && !strings.Contains(combined, c.wantCompra) {
				t.Fatalf("expected output to contain compra value %q; got: %s", c.wantCompra, combined)
			}
			if c.wantVenda != "" && !strings.Contains(combined, c.wantVenda) {
				t.Fatalf("expected output to contain venda value %q; got: %s", c.wantVenda, combined)
			}
		})
	}
}

func TestCLIHandlesCommentedJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("/*{" +
			"\"value\":[{\"cotacaoCompra\":7.11,\"cotacaoVenda\":7.22}] }*/"))
	}))
	defer ts.Close()

	// Build the binary
	bin := filepath.Join(os.TempDir(), "bcb_test_bin_cli")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = os.Environ()
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build test binary: %v, out=%s", err, string(out))
	}
	defer os.Remove(bin)

	cmd := exec.Command(bin, "-api", ts.URL, "-currency", "USD", "-price", "both", "-timeout", "2", "-retries", "1", "-backdays", "0")
	cmd.Env = append(os.Environ(), "LANG=C", "TZ=UTC")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error)
	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("process failed: %v, stderr=%s", err, stderr.String())
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatalf("process timed out")
	}

	combined := stdout.String() + "\n" + stderr.String()
	if !strings.Contains(combined, "7.11") || !strings.Contains(combined, "7.22") {
		t.Fatalf("expected output to contain compra and venda values; got: %s", combined)
	}
}
