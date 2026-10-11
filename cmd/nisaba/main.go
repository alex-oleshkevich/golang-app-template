package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"nisaba/internal/commands/db"
	"nisaba/internal/commands/health"
	"nisaba/internal/commands/serve"
	"nisaba/internal/logging"

	"github.com/urfave/cli/v3"
)

func main() {
	os.Exit(run(os.Args))
}

func run(args []string) int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cmd := &cli.Command{
		Name:           "nisaba",
		Usage:          "Nisaba CLI",
		DefaultCommand: "serve",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-format",
				Usage: "log output format: text or json",
				Value: string(logging.LogText),
			},
			&cli.StringFlag{
				Name:  "log-level",
				Usage: "minimum log level: debug, info, warn or error",
				Value: "info",
			},
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			format, err := logging.ParseFormat(c.String("log-format"))
			if err != nil {
				return ctx, err
			}
			var level slog.Level
			if err := level.UnmarshalText([]byte(c.String("log-level"))); err != nil {
				return ctx, err
			}
			slog.SetDefault(logging.New(format, level, os.Stderr))
			return ctx, nil
		},
		Commands: []*cli.Command{
			db.NewCommand(),
			health.NewCommand(),
			serve.NewCommand(),
		},
	}

	err := cmd.Run(ctx, args)
	if err == nil {
		return 0
	}
	if ctx.Err() != nil {
		return 130
	}

	if coded, ok := errors.AsType[cli.ExitCoder](err); ok {
		fmt.Fprintf(os.Stderr, "error: %s\n", coded.Error())
		return coded.ExitCode()
	}
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
	return 1
}
