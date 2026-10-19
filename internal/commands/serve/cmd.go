package serve

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	dbcmd "nisaba/internal/commands/db"
	"nisaba/internal/db"
	"nisaba/internal/web"

	"github.com/pressly/goose/v3"
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
			&cli.StringFlag{
				Name:    "db",
				Usage:   "SQLite database DSN",
				Value:   "nisaba.db",
				Sources: cli.EnvVars("NISABA_DB"),
			},
			&cli.BoolFlag{
				Name:    "db-migrate",
				Usage:   "apply pending migrations before serving",
				Sources: cli.EnvVars("NISABA_DB_MIGRATE"),
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
	if c.Bool("db-migrate") {
		if err := migrate(ctx, c); err != nil {
			return err
		}
	}

	conn, err := db.Open(ctx, c.String("db"))
	if err != nil {
		return err
	}
	defer conn.Close()

	server := web.New(web.Deps{
		Config: web.ServerConfig{
			ListenAddr: c.String("address"),
			ListenPort: c.String("port"),
			Docs:       c.Bool("docs"),
		},
	})
	return server.Run(ctx)
}

// migrate runs `db up` as its own command tree, so there is one migration code path.
func migrate(ctx context.Context, c *cli.Command) error {
	up := dbcmd.NewCommand()
	up.Writer = c.Root().Writer
	err := up.Run(ctx, []string{"db", "up", "--db", c.String("db")})
	// A fresh project has nothing to apply yet; that must not stop the server.
	if errors.Is(err, goose.ErrNoMigrations) {
		return nil
	}
	return err
}
