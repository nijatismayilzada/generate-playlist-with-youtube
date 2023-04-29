package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"sort"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"

	"github.com/blevesearch/bleve/v2"
)

type Artist struct {
	Name  string
	Songs []Song
}

type Song struct {
	RelativePath string
	Title        string
}

func main() {

	var path string
	flag.StringVar(&path, "path", "", "Working directory path")

	var devKey string
	flag.StringVar(&devKey, "devKey", "", "YouTube API Developer Key")

	var maxResults int64
	flag.Int64Var(&maxResults, "maxResults", 10, "Maximum number of YouTube Search result to analyse")

	var m3uFileName string
	flag.StringVar(&m3uFileName, "m3uFileName", "top10", "M3U filename")

	var skip int
	flag.IntVar(&skip, "skip", 0, "Skip this many artists in the working folder")

	flag.Parse()

	if devKey == "" {
		log.Fatalf("Please provide Youtube API Developer Key. devKey: %s", devKey)
	}

	if path == "" {
		log.Fatalf("Please provide working directory path. path: %s", path)
	}

	client := &http.Client{
		Transport: &transport.APIKey{Key: devKey},
	}

	service, err := youtube.New(client)
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}

	artists := getArtists(path)

	f, err := os.Create(fmt.Sprintf("%s/%s.m3u", path, m3uFileName))
	if err != nil {
		log.Fatalf("Error creating new m3u file: %v", err)
	}
	f.WriteString(fmt.Sprintln("#EXTM3U"))

	playlist := make(map[string]bool)

	for i, artist := range artists {
		fmt.Println(artist.Name)
		if i < skip {
			continue
		}

		call := service.Search.List([]string{"id,snippet"}).
			Q(artist.Name).
			MaxResults(maxResults).
			Order("viewCount").
			SafeSearch("none")
		response, err := call.Do()
		if err != nil {
			log.Printf("Error calling YouTube client: %v", err)
			continue
		}

		videos := make(map[string]string)

		for _, item := range response.Items {
			switch item.Id.Kind {
			case "youtube#video":
				videos[item.Id.VideoId] = item.Snippet.Title
			}
		}

		index, err := bleve.NewMemOnly(bleve.NewIndexMapping())
		if err != nil {
			log.Printf("Error creating bleve index: %v", err)
			continue
		}

		for _, s := range artist.Songs {
			index.Index(s.RelativePath, s.Title)
		}

		for _, title := range videos {
			query := bleve.NewMatchQuery(title)
			query.SetFuzziness(1)
			search := bleve.NewSearchRequest(query)
			searchResults, err := index.Search(search)
			if err != nil {
				log.Printf("Error while search: %v", err)
				continue
			}

			if len(searchResults.Hits) > 0 {
				fmt.Println(strings.TrimPrefix(searchResults.Hits[0].ID, path))
				playlist[fmt.Sprintln(strings.TrimPrefix(searchResults.Hits[0].ID, path))] = true
			}

		}

	}

	keys := make([]string, 0, len(playlist))
	for k := range playlist {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f.WriteString(k)
	}

	f.Close()
}

func getArtists(path string) []Artist {
	artists := []Artist{}

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Println(err)
		return artists
	}

	for _, entry := range entries {

		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			artists = append(artists, Artist{
				Name:  entry.Name(),
				Songs: getSongs(fmt.Sprintf("%s/%s", path, entry.Name())),
			})
		}
	}

	return artists
}

func getSongs(path string) []Song {
	files := []Song{}

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Println(err)
		return files
	}

	for _, entry := range entries {
		if entry.IsDir() {
			files = append(files, getSongs(fmt.Sprintf("%s/%s", path, entry.Name()))...)
		} else {
			files = append(files, Song{
				RelativePath: fmt.Sprintf("%s/%s", path, entry.Name()),
				Title:        entry.Name(),
			})
		}
	}

	return files
}
