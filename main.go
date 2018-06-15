package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	firebase "firebase.google.com/go"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/olekukonko/tablewriter"
	"github.com/y-yagi/configure"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Bookmark struct {
	Title     string    `firestore:"title"`
	Url       string    `firestore:"url"`
	CreatedAt time.Time `firestore:"createdAt"`
}

const cmd = "bookmarker"

type config struct {
	AccountKeyFile string `toml:"account_key_file"`
}

func init() {
	if !configure.Exist(cmd) {
		var cfg config
		cfg.AccountKeyFile = ""
		configure.Save(cmd, cfg)
	}
}

func main() {
	var edit bool

	flag.BoolVar(&edit, "c", false, "edit config")
	flag.Parse()

	if edit {
		os.Exit(msg(cmdEdit()))
	}

	var cfg config
	err := configure.Load(cmd, &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if cfg.AccountKeyFile == "" {
		fmt.Printf("please set key file to config file\n")
		os.Exit(1)
	}

	opt := option.WithCredentialsFile(cfg.AccountKeyFile)
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		fmt.Printf("error initializing app: %v", err)
		os.Exit(1)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		fmt.Printf("error get client: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	iter := client.Collection("bookmarks").Documents(ctx)
	var bookmark Bookmark

	table := tablewriter.NewWriter(os.Stdout)
	table.SetColWidth(80)
	table.SetHeader([]string{"Title", "URL"})

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Printf("failed to iterate\n: %v", err)
			os.Exit(1)
		}

		if err := doc.DataTo(&bookmark); err != nil {
			fmt.Printf("failed to convert to Bookmark\n: %v", err)
			os.Exit(1)
		}

		table.Append([]string{runewidth.Truncate(bookmark.Title, 80, "..."), bookmark.Url})
	}
	table.Render()
}

func msg(err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		return 1
	}
	return 0
}

func cmdEdit() error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return configure.Edit(cmd, editor)
}
