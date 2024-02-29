package main

import (
	"context"
	"time"

	"github.com/alecthomas/kong"
)

type CLI struct {
	AppID         int64         `required:"" env:"GHA_APP_ID" help:"GitHub App ID of application to authenticate as"`
	AppKey        string        `required:"" env:"GHA_APP_KEY" help:"Private key of GitHub App for the App ID"`
	Owner         string        `required:"" env:"GHA_OWNER" help:"GitHub owner of repositories to monitor"`
	Repos         []string      `required:"" help:"Repositories to monitor (must be owned by --owner)"`
	Sleep         time.Duration `default:"1m" help:"Sleep time (duration string)between polls of GitHub Actions"`
	InitialWindow time.Duration `default:"2h" help:"Initial time to look back for runs"`
	Backfill      bool          `default:"false" help:"Backfill completed runs from initial window"`
}

func main() {
	var cli CLI
	kctx := kong.Parse(&cli)
	err := kctx.Run(&cli)
	kctx.FatalIfErrorf(err)
}

func (c *CLI) Run() error {
	ctx := context.Background()
	collector := NewCollector(c)
	go collector.Run(ctx)
	return RunServer(c)
}
