package main

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/arunaShakthi/web-scraper/internal/db"
	"github.com/google/uuid"
)

func startScraping(
	dbQueries *db.Queries,
	concurrency int,
	timeBetweenRequest time.Duration,
) {
	log.Printf("Scraping on %v go routines every %s duration", concurrency, timeBetweenRequest)
	ticker := time.NewTicker(timeBetweenRequest)
	for ; ; <-ticker.C {
		feeds, err := dbQueries.GetNextFeedsToFetch(
			context.Background(),
			int32(concurrency))
		if err != nil {
			log.Println("error fetching feeds", err)
			continue
		}

		wg := &sync.WaitGroup{}
		for _, feed := range feeds {
			wg.Add(1)
			go scrapeFeed(dbQueries, wg, feed)
		}
		wg.Wait()

	}
}

func scrapeFeed(dbQueries *db.Queries, wg *sync.WaitGroup, feed db.Feed) {
	defer wg.Done()

	_, err := dbQueries.MarkFeedFetched(
		context.Background(),
		feed.ID)

	if err != nil {
		log.Println("error marking feed fetched", err)
		return
	}

	rssFeed, err := urlToFeed(feed.Url)
	if err != nil {
		log.Println("error fetching feed", err)
		return
	}

	for _, item := range rssFeed.Channel.Items {
		pubAt := time.Now().UTC()
		if item.PubDate != "" {
			t, err := time.Parse(time.RFC1123Z, item.PubDate)
			if err == nil {
				pubAt = t.UTC()
			} else {
				t, err = time.Parse(time.RFC1123, item.PubDate)
				if err == nil {
					pubAt = t.UTC()
				} else {
					log.Printf("couldn't parse date %q: %v", item.PubDate, err)
				}
			}
		}

		_, err = dbQueries.CreatePost(context.Background(),
			db.CreatePostParams{
				ID:        uuid.New(),
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
				Title:     item.Title,
				Description: sql.NullString{
					String: item.Description,
					Valid:  item.Description != "",
				},
				PublishedAt: pubAt,
				Url:         item.Link,
				FeedID:      feed.ID,
			},
		)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			log.Printf("error creating post: %v", err)
			continue
		}
	}
	log.Printf("Feed %s collected, %d items found", feed.Url, len(rssFeed.Channel.Items))
}
