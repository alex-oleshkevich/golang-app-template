package serve

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"nisaba/internal/web"

	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "serve the nisaba server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:      "port",
				Usage:     "port to serve on",
				Value:     "8080",
				Validator: validatePort,
			},
			&cli.StringFlag{
				Name:      "address",
				Usage:     "address to serve on",
				Value:     "127.0.0.1",
				Validator: validateAddress,
			},
			&cli.BoolFlag{
				Name:    "docs",
				Usage:   "serve the OpenAPI document and the Swagger UI / Scalar pages",
				Sources: cli.EnvVars("NISABA_DOCS"),
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

// An empty host would silently bind every interface; 0.0.0.0 must be asked for explicitly.
func validateAddress(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return errors.New("address must not be empty; use 0.0.0.0 to listen on all interfaces")
	}
	return nil
}

func run(ctx context.Context, c *cli.Command) error {
	server := web.New(web.Deps{
		Config: web.ServerConfig{
			ListenAddr: c.String("address"),
			ListenPort: c.String("port"),
			Docs:       c.Bool("docs"),
		},
	})
	return server.Run(ctx)
}
