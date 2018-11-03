package main

import (
	"flag"
	"fmt"
	"github.com/caser/gophernews"
	"github.com/jzelinskie/geddit"
	"os"
	"sync"
)

var redditSession *geddit.LoginSession
var hackerNewsClient *gophernews.Client

type Story struct {
	title  string
	url    string
	author string
	source string
}

func getHnStoryDetails(id int, c chan<- Story, wg *sync.WaitGroup) {
	defer wg.Done()
	story, err := hackerNewsClient.GetStory(id)

	if err != nil {
		return
	}

	newStory := Story{
		title:  story.Title,
		url:    story.URL,
		author: story.By,
		source: "hn",
	}

	c <- newStory
}

func newHnStories(c chan<- Story) {
	defer close(c)
	changes, err := hackerNewsClient.GetChanges()

	if err != nil {
		fmt.Println(err)
		return
	}

	var wg sync.WaitGroup

	for _, id := range changes.Items {
		wg.Add(1)
		go getHnStoryDetails(id, c, &wg)
	}

	wg.Wait()
}

func newRedditStories(c chan<- Story) {
	defer close(c)
	sort := geddit.PopularitySort(geddit.NewSubmissions)
	var listingOptions geddit.ListingOptions
	submissions, err := redditSession.SubredditSubmissions("programming", sort, listingOptions)

	if err != nil {
		fmt.Println(err)
		return
	}

	for _, s := range submissions {
		newStory := Story{
			title:  s.Title,
			url:    s.URL,
			author: s.Author,
			source: "Reddit",
		}

		c <- newStory
	}
}

func outputToConsole(c <-chan Story) {
	for {
		s := <-c
		fmt.Printf("Title: %s\n Link: %s\n Author: %s\n Source: %s\n\n", s.title, s.url, s.author, s.source)
	}
}

func outputToFile(c <-chan Story, file *os.File) {
	for {
		s := <-c
		fmt.Fprintf(file, "Title: %s\n Link: %s\n Author: %s\n Source: %s\n\n", s.title, s.url, s.author, s.source)
	}
}

func main() {
	redditUser := flag.String("redditUser", "", "Reddit user name")
	redditPwd := flag.String("redditPwd", "", "Reddit user password")
	flag.Parse()

	if *redditUser == "" || *redditPwd == "" {
		panic("reddit user and pwd is required")
	}

	hackerNewsClient = gophernews.NewClient()
	var err error
	redditSession, err = geddit.NewLoginSession(*redditUser, *redditPwd, "gdAgent v0")
	if err != nil {
		panic(err)
	}

	fromHn := make(chan Story, 8)
	fromReddit := make(chan Story, 8)
	toFile := make(chan Story, 8)
	toPrint := make(chan Story, 8)

	go newHnStories(fromHn)
	go newRedditStories(fromReddit)

	file, err := os.Create("stories.txt")
	if err != nil {
		panic(err)
	}

	go outputToConsole(toPrint)
	go outputToFile(toFile, file)

	hnOpen := true
	redditOpen := true

	for hnOpen || redditOpen {
		select {
		case story, open := <-fromHn:
			if open {
				toFile <- story
				toPrint <- story
			} else {
				hnOpen = false
			}
		case story, open := <-fromReddit:
			if open {
				toFile <- story
				toPrint <- story
			} else {
				redditOpen = false
			}
		}
	}
}
