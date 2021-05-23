package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/bloops-games/bloops/internal/httputil"
	"github.com/bloops-games/bloops/internal/logging"
	"github.com/bloops-games/bloops/internal/shutdown"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	URL      string `envconfig:"BLOOP_HP_URL"`
	Username string `envconfig:"BLOOP_HP_USERNAME"`
	Password string `envconfig:"BLOOP_HP_PASSWORD"`
}

type OkResponse struct {
	Status string `json:"status"`
}

func main() {
	flag.Parse()
	ctx, cancel := shutdown.New()
	logger := logging.FromContext(ctx)
	defer cancel()
	config := Config{}
	if err := envconfig.Process("", &config); err != nil {
		logger.Fatalf("processing the config: %v", err)
	}

	client := httputil.NewClient(httputil.NewBasicAuthRoundTripper(config.Username, config.Password, &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		DisableCompression:    true,
		IdleConnTimeout:       5 * time.Minute,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, config.URL, nil)
	if err != nil {
		logger.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Fatalf("client get: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			logger.Fatalf("read all body bytes: %v", err)
		}
		var Ok OkResponse
		if err := json.Unmarshal(bytes, &Ok); err != nil {
			logger.Fatalf("body unmarshal: %v", err)
		}
		_, _ = fmt.Fprint(os.Stdout, Ok.Status)
		_, _ = fmt.Fprint(os.Stdout, "\n")
		os.Exit(1)
	}

	_, _ = fmt.Fprint(os.Stdout, strconv.Itoa(resp.StatusCode))
	_, _ = fmt.Fprint(os.Stdout, "\n")
}
