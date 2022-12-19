package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/davexpro/m3u8-dl/internal/download"
	"github.com/davexpro/m3u8-dl/util"
	"github.com/urfave/cli/v2"
)

var (
	flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "disable-update",
			Value: false,
			Usage: "disable self update",
		},
	}
	commands = []*cli.Command{
		{
			Name:        "down",
			Description: "download the video file from m3u8",
			Action:      actionDown,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "input",
					Aliases: []string{"i"},
				},
				&cli.StringFlag{
					Name:    "output",
					Aliases: []string{"o"},
					Value:   "./",
				},
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Value:   "main.mp4",
				},
				&cli.IntFlag{
					Name:    "thread",
					Aliases: []string{"t"},
					Value:   8,
				},
				&cli.BoolFlag{
					Name:  "use-ffmpeg",
					Value: true,
				},
				&cli.StringFlag{
					Name: "origin",
				},
				&cli.StringFlag{
					Name: "referer",
				},
				&cli.StringFlag{
					Name: "cookie",
				},
				&cli.StringFlag{
					Name: "user-agent",
				},
			},
		},
	}
)

func actionDown(c *cli.Context) error {
	m3u8Url, outPath := strings.TrimSpace(c.String("input")), strings.TrimSpace(c.String("output"))
	if len(m3u8Url) <= 0 || len(outPath) <= 0 {
		return fmt.Errorf("input & output are required")
	}
	filename := strings.TrimSpace(c.String("name"))
	if len(filename) <= 0 {
		filename = fmt.Sprintf("merged_%s.ts", util.MD5Short(m3u8Url))
	}

	// set http client config
	util.Origin = c.String("origin")
	util.Cookies = c.String("cookie")
	util.Referer = c.String("referer")
	util.UserAgent = c.String("user-agent")

	thread := c.Int("thread")
	if thread <= 0 {
		thread = 1
	}
	down, err := download.NewDownloader(m3u8Url, outPath, filename, thread, c.Bool("use-ffmpeg"))
	if err != nil {
		panic(err)
	}
	if err = down.Start(); err != nil {
		panic(err)
	}
	fmt.Println("[*] All Done!")

	return nil
}

func main() {
	// init cli
	app := &cli.App{
		Name:     util.Name,
		Usage:    "m3u8-dl <https://github.com/DavexPro/m3u8-dl>",
		Version:  util.Version,
		Writer:   os.Stdout,
		Flags:    flags,
		Commands: commands,
	}
	rand.Seed(time.Now().UnixNano())

	// run the cli
	err := app.Run(os.Args)
	if err != nil {
		log.Println(err.Error())
	}
}
