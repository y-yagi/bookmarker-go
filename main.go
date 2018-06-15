package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	firebase "firebase.google.com/go"
	"github.com/jroimartin/gocui"
	"github.com/y-yagi/configure"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Bookmark struct {
	Title     string    `firestore:"title"`
	Url       string    `firestore:"url"`
	CreatedAt time.Time `firestore:"createdAt"`
}

type config struct {
	AccountKeyFile string `toml:"account_key_file"`
}

const cmd = "bookmarker"

var bookmarks []Bookmark

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

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Printf("failed to iterate: %v\n", err)
			os.Exit(1)
		}

		if err := doc.DataTo(&bookmark); err != nil {
			fmt.Printf("failed to convert to Bookmark: %v\n", err)
			os.Exit(1)
		}
		bookmarks = append(bookmarks, bookmark)
	}

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		fmt.Printf("GUI create error: %v\n", err)
		os.Exit(1)
	}
	defer g.Close()

	g.Cursor = true
	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		os.Exit(msg(err))
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		os.Exit(msg(err))
	}
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

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, open); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'j', gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'k', gocui.ModNone, cursorUp); err != nil {
		return err
	}

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("main", 0, 0, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "BookMarker"
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		for _, b := range bookmarks {
			fmt.Fprintf(v, "[%s](%s)\n", b.Title, b.Url)
		}
	}
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func open(g *gocui.Gui, v *gocui.View) error {
	var l string
	var err error

	if v == nil {
		if v, err = g.SetCurrentView("main"); err != nil {
			return err
		}
	}

	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}

	url := strings.TrimLeft(strings.Split(l, "]")[0], "[")
	return exec.Command("google-chrome", url).Run()
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v == nil {
		if v, err = g.SetCurrentView("main"); err != nil {
			return err
		}
	}

	cx, cy := v.Cursor()
	lineCount := len(strings.Split(v.ViewBuffer(), "\n"))
	if cy+1 == lineCount-2 {
		return nil
	}
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	var err error

	if v == nil {
		if v, err = g.SetCurrentView("main"); err != nil {
			return err
		}
	}

	ox, oy := v.Origin()
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}

	return nil
}
