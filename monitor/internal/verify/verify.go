// Package verify membuktikan kedua service berperilaku identik (contract test):
// request yang sama harus menghasilkan status & body yang sama.
package verify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/gorust/monitor/internal/target"
)

type testCase struct {
	Name   string
	Method string
	Path   string
	Body   string
}

// Urutan penting: PUT item 7 dulu supaya GET item 7 deterministik
// walaupun skenario "store" sudah mengubah data.
var cases = []testCase{
	{"health", "GET", "/health", ""},
	{"hello default", "GET", "/api/hello", ""},
	{"hello dengan nama", "GET", "/api/hello?name=Gopher", ""},
	{"json stats", "POST", "/api/json", `{"numbers":[1,2,3,4.5,-2]}`},
	{"json kosong", "POST", "/api/json", `{"numbers":[]}`},
	{"json tipe salah", "POST", "/api/json", `{"numbers":"x"}`},
	{"json rusak", "POST", "/api/json", `{"numbers":[1,2`},
	{"cpu fib(20)", "GET", "/api/cpu?n=20", ""},
	{"cpu default", "GET", "/api/cpu", ""},
	{"cpu n terlalu besar", "GET", "/api/cpu?n=99", ""},
	{"cpu n bukan angka", "GET", "/api/cpu?n=abc", ""},
	{"item upsert", "PUT", "/api/items/7", `{"name":"verified","price":12.5}`},
	{"item get", "GET", "/api/items/7", ""},
	{"item tidak ada", "GET", "/api/items/999999", ""},
	{"item id bukan angka", "GET", "/api/items/abc", ""},
	{"item nama kosong", "PUT", "/api/items/5", `{"name":"","price":1}`},
	{"item harga negatif", "PUT", "/api/items/5", `{"name":"x","price":-1}`},
	{"item harga hilang", "PUT", "/api/items/5", `{"name":"x"}`},
}

type Response struct {
	Status    int     `json:"status"`
	Body      string  `json:"body"`
	LatencyMs float64 `json:"latency_ms"`
	Err       string  `json:"err,omitempty"`
}

type CaseResult struct {
	Name    string              `json:"name"`
	Method  string              `json:"method"`
	Path    string              `json:"path"`
	Request string              `json:"request,omitempty"`
	Results map[string]Response `json:"results"`
	Match   bool                `json:"match"`
	Note    string              `json:"note"`
}

func Run(ctx context.Context, targets []target.Target) []CaseResult {
	client := &http.Client{Timeout: 5 * time.Second}
	out := make([]CaseResult, 0, len(cases))
	for _, c := range cases {
		cr := CaseResult{Name: c.Name, Method: c.Method, Path: c.Path, Request: c.Body, Results: map[string]Response{}}
		for _, t := range targets {
			cr.Results[t.Name] = do(ctx, client, t, c)
		}
		cr.Match, cr.Note = compare(cr.Results)
		out = append(out, cr)
	}
	return out
}

func do(ctx context.Context, client *http.Client, t target.Target, c testCase) Response {
	var body io.Reader
	if c.Body != "" {
		body = strings.NewReader(c.Body)
	}
	req, err := http.NewRequestWithContext(ctx, c.Method, t.URL+c.Path, body)
	if err != nil {
		return Response{Err: err.Error()}
	}
	if c.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return Response{Err: err.Error()}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return Response{
		Status:    resp.StatusCode,
		Body:      strings.TrimSpace(string(b)),
		LatencyMs: float64(time.Since(start).Microseconds()) / 1000,
	}
}

// compare: status harus sama. Untuk sukses, body JSON harus sama persis (field
// "lang" diabaikan). Untuk error, cukup keduanya berisi field "error" — pesan
// boleh beda karena berasal dari library parsing yang berbeda.
func compare(results map[string]Response) (bool, string) {
	var first *Response
	var firstBody any
	for _, r := range results {
		if r.Err != "" {
			return false, "request gagal: " + r.Err
		}
		body := normalize(r.Body)
		if first == nil {
			r := r
			first, firstBody = &r, body
			continue
		}
		if r.Status != first.Status {
			return false, "status code berbeda"
		}
		if r.Status >= 400 {
			if !hasErrorField(body) || !hasErrorField(firstBody) {
				return false, "body error tidak berformat {\"error\": ...}"
			}
			continue
		}
		if !reflect.DeepEqual(body, firstBody) {
			return false, "body berbeda"
		}
	}
	if first != nil && first.Status >= 400 {
		return true, "status sama, pesan error boleh beda"
	}
	return true, "identik"
}

func normalize(s string) any {
	var v any
	if json.Unmarshal([]byte(s), &v) != nil {
		return s
	}
	if m, ok := v.(map[string]any); ok {
		delete(m, "lang")
	}
	return v
}

func hasErrorField(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	_, ok = m["error"]
	return ok
}
