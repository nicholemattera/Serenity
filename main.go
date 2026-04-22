package main

import (
	"log/slog"
	"os"

	"github.com/nicholemattera/serenity/cmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	cmd.Execute()
}
