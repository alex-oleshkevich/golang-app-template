package health

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "health",
		Usage: "check that the local server is responding",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "port",
				Usage:     "port the server is listening on",
				Value:     "8080",
				Validator: validatePort,
			},
		},
		Action: run,
	}
}

func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("port %q is outside 1-65535", port)
	}
	return nil
}

func run(ctx context.Context, c *cli.Command) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	url := "http://" + net.JoinHostPort("127.0.0.1", c.String("port")) + "/healthz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET /healthz returned %s", resp.Status)
	}
	return nil
}
