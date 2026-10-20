package db

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"text/tabwriter"
	"time"

	"nisaba/internal/db"

	"github.com/pressly/goose/v3"
	"github.com/urfave/cli/v3"
)

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:  "db",
		Usage: "manage the database schema",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "db",
				Usage:   "SQLite database file",
				Value:   "nisaba.db",
				Sources: cli.EnvVars("APP_DB"),
			},
		},
		Commands: []*cli.Command{
			{Name: "up", Usage: "apply all migrations (head) or N steps", ArgsUsage: "[head|N]", Action: up},
			{Name: "down", Usage: "roll back all migrations (base) or N steps", ArgsUsage: "[base|-N]", Action: down},
			{Name: "history", Usage: "list migrations and when they were applied", Action: history},
			{Name: "new", Usage: "create a migration in " + db.MigrationsDir, ArgsUsage: "<name>", Action: create},
		},
	}
}

func withMigrator(ctx context.Context, c *cli.Command, fn func(*goose.Provider) error) error {
	conn, err := db.Open(ctx, c.String("db"))
	if err != nil {
		return err
	}
	defer conn.Close()
	migrator, err := db.Migrator(conn)
	if err != nil {
		return err
	}
	return fn(migrator)
}

func up(ctx context.Context, c *cli.Command) error {
	target, err := parseTarget(c, "head")
	if err != nil {
		return err
	}
	return withMigrator(ctx, c, func(m *goose.Provider) error {
		if target == "head" {
			results, err := m.Up(ctx)
			return bulk(c, results, err)
		}
		n, err := strconv.Atoi(target)
		if err != nil || n < 1 {
			return fmt.Errorf("up target must be head or a positive step, got %q", target)
		}
		return step(ctx, c, n, m.UpByOne)
	})
}

func down(ctx context.Context, c *cli.Command) error {
	target, err := parseTarget(c, "-1")
	if err != nil {
		return err
	}
	return withMigrator(ctx, c, func(m *goose.Provider) error {
		if target == "base" {
			results, err := m.DownTo(ctx, 0)
			return bulk(c, results, err)
		}
		n, err := strconv.Atoi(target)
		if err != nil || n > -1 {
			return fmt.Errorf("down target must be base or a negative step, got %q", target)
		}
		return step(ctx, c, -n, m.Down)
	})
}

func step(
	ctx context.Context, c *cli.Command, n int, move func(context.Context) (*goose.MigrationResult, error),
) error {
	for range n {
		r, err := move(ctx)
		if errors.Is(err, goose.ErrNoNextVersion) {
			return nil
		}
		if err != nil {
			return err
		}
		report(c, r)
	}
	return nil
}

func parseTarget(c *cli.Command, fallback string) (string, error) {
	if c.Args().Len() > 1 {
		return "", fmt.Errorf("expected one target, got %q", c.Args().Slice())
	}
	return cmp.Or(c.Args().First(), fallback), nil
}

func bulk(c *cli.Command, results []*goose.MigrationResult, err error) error {
	if partial, ok := errors.AsType[*goose.PartialError](err); ok {
		results = partial.Applied
	}
	report(c, results...)
	return err
}

func report(c *cli.Command, results ...*goose.MigrationResult) {
	for _, r := range results {
		fmt.Fprintln(c.Root().Writer, r.Direction, path.Base(r.Source.Path))
	}
}

func history(ctx context.Context, c *cli.Command) error {
	return withMigrator(ctx, c, func(m *goose.Provider) error {
		statuses, err := m.Status(ctx)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(c.Root().Writer, 0, 0, 2, ' ', 0)
		for _, s := range statuses {
			applied := "pending"
			if s.State == goose.StateApplied {
				applied = s.AppliedAt.Local().Format(time.DateTime)
			}
			fmt.Fprintf(w, "%s\t%s\n", path.Base(s.Source.Path), applied)
		}
		return w.Flush()
	})
}

func create(_ context.Context, c *cli.Command) error {
	name := c.Args().First()
	if name == "" {
		return errors.New("migration name is required")
	}
	return goose.Create(nil, db.MigrationsDir, name, "sql")
}
