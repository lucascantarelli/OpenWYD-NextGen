package metrics

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// HealthcheckArgs atende o contrato `<binário> healthcheck`, usado pelo
// healthcheck do compose. Existe porque as imagens finais são distroless: não há
// shell, `wget` nem `curl` lá dentro, então o próprio binário consulta o seu
// /healthz e traduz em código de saída — o mesmo desenho do logpipe.
//
// Devolve só quando o argumento NÃO é "healthcheck" (o processo segue normal).
// A porta vem de W2PP_METRICS_ADDR e, na ausência dela, de defAddr (o default de
// cada serviço). Chamar no topo do main(), antes de interpretar as flags.
func HealthcheckArgs(defAddr string) {
	if len(os.Args) < 2 || strings.ToLower(os.Args[1]) != "healthcheck" {
		return
	}
	addr := os.Getenv("W2PP_METRICS_ADDR")
	if addr == "" {
		addr = defAddr
	}
	os.Exit(Healthcheck(addr))
}

// Healthcheck consulta http://127.0.0.1:<porta>/healthz e devolve 0 quando a
// resposta é 2xx, 1 em qualquer outro caso. Endereço vazio (endpoint de métricas
// desligado) conta como falha: sem /healthz não há como atestar saúde.
func Healthcheck(addr string) int {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: endereço inválido %q: %v\n", addr, err)
		return 1
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(host, port) + "/healthz"

	client := &http.Client{Timeout: 2 * time.Second}
	res, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %s: %v\n", url, err)
		return 1
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		fmt.Fprintf(os.Stderr, "healthcheck: %s: status %d\n", url, res.StatusCode)
		return 1
	}
	return 0
}
