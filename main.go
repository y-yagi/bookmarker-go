package main

import (
	"context"
	"fmt"
	"os"
	"time"

	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Bookmark struct {
	Title     string    `firestore:"title"`
	Url       string    `firestore:"url"`
	CreatedAt time.Time `firestore:"createdAt"`
}

func main() {
	opt := option.WithCredentialsFile(os.Getenv("BOOKMARKER_KEY_FILE"))
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
			fmt.Printf("failed to iterate\n: %v", err)
			os.Exit(1)
		}

		if err := doc.DataTo(&bookmark); err != nil {
			fmt.Printf("failed to convert to Bookmark\n: %v", err)
			os.Exit(1)
		}

		fmt.Printf("[%s](%s)\n", bookmark.Title, bookmark.Url)
	}
}
