package scraper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cjang5/cinemaData"
	"github.com/cjang5/ds/queue"
	"golang.org/x/net/html"
)

// iotas representing if the next wiki article to parse is an Actor or a Movie
// 0 = Actor
// 1 = Movie
const (
	actor      = iota
	movie      = iota
	timeFormat = "2006-01-02"
	wikipedia  = "https://en.wikipedia.org"
)

// Scraper will contain methods to scrape data from a given startpoint
// and other methods to
type Scraper struct {
	q *queue.Queue
	g *cinemaData.Graph
}

// target represents a wiki page that will be parsed
// url is the wikipedia URL, and pageType tells whether it is an Actor page or a Movie page
type target struct {
	url      string
	pageType int
}

func New() *Scraper {
	s := new(Scraper)
	s.q = queue.New()
	s.g = cinemaData.New()

	return s
}

// newTarget is given the url of the actor/movie page and an iota
// indicating whether it is an actor/movie and creates a new target
func newTarget(s string, p int) *target {
	t := new(target)
	t.url = s
	t.pageType = p

	return t
}

// AddTarget creates and adds a new target to our Scraper's queue
func (s *Scraper) AddTarget(url string, page int) {
	s.q.Enqueue(newTarget(url, page))
}

func (s *Scraper) GetTarget() (string, int, error) {
	if s.q.IsEmpty() {
		return "", -1, errors.New("no more urls to scrape")
	}

	next := s.q.Dequeue().(*target)
	return next.url, next.pageType, nil
}

func (s *Scraper) scrape() {
	// get next URL
	url, pageType, err := s.GetTarget()
	if err != nil {
		fmt.Println(err)
		return // TODO: Proper error logging
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("error:", err) // TODO: Proper message logging for errors later
	}

	defer resp.Body.Close()

	// TODO: Maybe put this in another func called 'analyze' or something
	// Create a new Tokenizer
	if pageType == actor {
		s.analyzeActorPage(url, resp.Body)
	}

	//fmt.Print("HTML:\n\n")
	//fmt.Println(string(bytes))
}

// findInfobox will scan through the page and look for the infobox
func (s *Scraper) findInfobox(body io.Reader) *html.Tokenizer {
	z := html.NewTokenizer(body)
	//i := 0 // TEMP
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			//fmt.Println("Found", i, "infobox(es)") // TEMP
			return nil
		case tt == html.StartTagToken:
			t := z.Token()
			if t.Data == "table" && t.Attr != nil && t.Attr[0].Key == "class" && strings.Contains(t.Attr[0].Val, "infobox") {
				return z
			}
		}
	}
}
