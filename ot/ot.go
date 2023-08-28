package main

import (
    "fmt"
    "log"
    "os"
	// "hello/replicate"
    // "hello/scan"
    "github.com/urfave/cli/v2"
)

func main() {
    app := &cli.App{
        Name:  "ot",
        Usage: "Replicate and Scan images",
        Commands: []*cli.Command{
            {
                Name:    "replicator",
                Usage:   "replicate Images",
                Action: func(cCtx *cli.Context) error {
                    // replicate(cCtx.Args())
                    fmt.Println("Replicator task: ", cCtx.Args())
                    return nil
                },
            },
            {
                Name:    "scanner",
                Usage:   "scan Images",
                Action: func(cCtx *cli.Context) error {
                    // scan(cCtx.Args())
                    fmt.Println("Scanned task: ", cCtx.Args())
                    return nil
                },
            },
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}
