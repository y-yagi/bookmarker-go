package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"github.com/y-yagi/configure"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Bookmark is a bookmark module.
type Bookmark struct {
	Title     string    `firestore:"title"`
	URL       string    `firestore:"url"`
	CreatedAt time.Time `firestore:"createdAt"`
}

const cmd = "bookmarker"

type config struct {
	AccountKeyFile string `toml:"account_key_file"`
	Browser        string `toml:"browser"`
	FilterCmd      string `toml:"filter_cmd"`
}

var cfg config

func init() {
	if !configure.Exist(cmd) {
		cfg.AccountKeyFile = ""
		cfg.Browser = "google-chrome"
		cfg.FilterCmd = "peco"
		configure.Save(cmd, cfg)
	}
}

func main() {
	var edit bool

	flag.BoolVar(&edit, "c", false, "edit config")
	flag.Parse()

	if edit {
		if err := editConfig(); err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	err := configure.Load(cmd, &cfg)
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.AccountKeyFile == "" {
		fmt.Printf("please set key file to config file\n")
		os.Exit(1)
	}

	var bookmarks []Bookmark
	if err = fetchBookmarks(&bookmarks); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	if err := show(&bookmarks); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func editConfig() error {
	editor := os.Getenv("EDITOR")
	if len(editor) == 0 {
		editor = "vim"
	}

	return configure.Edit(cmd, editor)
}

func fetchBookmarks(bookmarks *[]Bookmark) error {
	opt := option.WithCredentialsFile(cfg.AccountKeyFile)
	ctx := context.Background()
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return fmt.Errorf("rror initializing app: %v", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return fmt.Errorf("error get client: %v", err)
	}
	defer client.Close()

	iter := client.Collection("bookmarks").Documents(ctx)
	var bookmark Bookmark

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to iterate: %v", err)
		}

		if err := doc.DataTo(&bookmark); err != nil {
			return fmt.Errorf("failed to convert to Bookmark: %v", err)
		}

		*bookmarks = append(*bookmarks, bookmark)
	}

	return nil
}

func show(bookmarks *[]Bookmark) error {
	var buf bytes.Buffer
	var r string

	for _, b := range *bookmarks {
		r += "[" + b.Title + "](" + b.URL + ")\n"
	}

	if err := runFilter(cfg.FilterCmd, strings.NewReader(r), &buf); err != nil {
		return err
	}

	if buf.Len() == 0 {
		return errors.New("No bookmark selected")
	}

	re := regexp.MustCompile(`\((.+?)\)\z`)
	matched := re.FindAllStringSubmatch(strings.TrimSpace(buf.String()), -1)

	return exec.Command(cfg.Browser, matched[0][1]).Run()
}

func runFilter(command string, r io.Reader, w io.Writer) error {
	command = os.Expand(command, func(s string) string {
		return os.Getenv(s)
	})

	cmd := exec.Command("sh", "-c", command)

	cmd.Stderr = os.Stderr
	cmd.Stdout = w
	cmd.Stdin = r

	return cmd.Run()
}
