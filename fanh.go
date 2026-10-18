package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
)

type Args struct {
	Key   string `arg:"-k, --key" help:"nhentai API key"`
	Log   bool   `arg:"-l, --log" help:"with logging (request rate limiting warning will NOT be displayed if this is not set)"`
	Sort  bool   `arg:"-s, --sort" help:"sorted favourites"`
	Title bool   `arg:"-t, --title" help:"favourites with titles"`
}

func (Args) Version() string {
	return "1.2.0"
}

func (Args) Description() string {
	return "fanh - fetch your favourites on nhentai"
}

type Error struct {
	Error string `json:"error"`
}

type Book struct {
	ID    int    `json:"id"`
	Title string `json:"english_title"`
}

type Favourite struct {
	Result []Book `json:"result"`
	Length int    `json:"num_pages"`
}

func main() {
	var args Args
	p := arg.MustParse(&args)

	if args.Key == "" {
		p.Fail("nhentai API key not set")
	}

	if !args.Log {
		log.SetOutput(io.Discard)
	}

	page := 1
	list := make([]int, 0)
	title := make(map[int]string)
	client := &http.Client{}

	for {
		log.Printf("fetching page %d\n", page)
		url := fmt.Sprintf("https://nhentai.net/api/v2/favorites?page=%d", page)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("failed to create request for page %d: %s\n", page, err.Error())
		}

		req.Header.Set("User-Agent", fmt.Sprintf("fanh/%s", args.Version()))
		req.Header.Set("Authorization", fmt.Sprintf("Key %s", args.Key))

		res, err := client.Do(req)
		if err != nil {
			log.Fatalf("failed to fetch page %d: %s\n", page, err.Error())
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Fatalf("failed to read response body: %s\n", err.Error())
		}

		if res.StatusCode != http.StatusOK {
			var data Error
			if err := json.Unmarshal(body, &data); err != nil {
				log.Fatalf("failed to unmarshal response: %s\n", err.Error())
			}

			log.Fatalf("failed to fetch page %d: %s\n", page, data.Error)
		}

		var data Favourite
		if err := json.Unmarshal(body, &data); err != nil {
			log.Fatalf("failed to unmarshal response: %s\n", err.Error())
		}

		for _, book := range data.Result {
			list = append(list, book.ID)
			title[book.ID] = book.Title
		}

		log.Printf("succeeded in fetching page %d\n", page)

		if page < data.Length {
			if data.Length > 15 {
				log.Println("sleeping for 4 seconds due to request rate limiting")
				time.Sleep(4 * time.Second)
			}

			page++
		} else {
			break
		}
	}

	if args.Sort {
		sort.Ints(list)
	}

	if args.Title {
		for _, id := range list {
			fmt.Printf("%d: %s\n", id, title[id])
		}
	} else {
		slist := make([]string, len(list))
		for i, id := range list {
			slist[i] = strconv.Itoa(id)
		}

		fmt.Println(strings.Join(slist, ", "))
	}
}
