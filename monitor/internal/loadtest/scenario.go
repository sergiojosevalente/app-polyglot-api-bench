package loadtest

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

var (
	Scenarios = []string{"hello", "json", "cpu", "store", "mixed"}
	Modes     = []string{"load", "stress", "spike"}
)

type request struct {
	method string
	path   string
	body   []byte
}

var (
	statsBody = buildStatsBody(100)
	putBody   = []byte(`{"name":"item-updated","price":9.99}`)
)

func buildStatsBody(n int) []byte {
	nums := make([]string, n)
	for i := range nums {
		nums[i] = strconv.FormatFloat(float64(i)*1.5-37.25, 'f', -1, 64)
	}
	return []byte(`{"numbers":[` + strings.Join(nums, ",") + `]}`)
}

// nextRequest memilih request berikutnya sesuai skenario.
// math/rand/v2 global function aman & cepat dipakai banyak goroutine.
func nextRequest(scenario string, cpuN int) request {
	switch scenario {
	case "hello":
		return request{method: "GET", path: "/api/hello?name=bench"}
	case "json":
		return request{method: "POST", path: "/api/json", body: statsBody}
	case "cpu":
		return request{method: "GET", path: "/api/cpu?n=" + strconv.Itoa(cpuN)}
	case "store":
		id := rand.IntN(1000) + 1
		if rand.IntN(10) < 2 { // 20% tulis
			return request{method: "PUT", path: fmt.Sprintf("/api/items/%d", id), body: putBody}
		}
		return request{method: "GET", path: fmt.Sprintf("/api/items/%d", id)}
	default: // mixed: 40% hello, 20% json, 30% store, 10% cpu
		switch r := rand.IntN(10); {
		case r < 4:
			return nextRequest("hello", cpuN)
		case r < 6:
			return nextRequest("json", cpuN)
		case r < 9:
			return nextRequest("store", cpuN)
		default:
			return nextRequest("cpu", cpuN)
		}
	}
}
