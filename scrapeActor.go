package scraper

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// analyzeActorPage will call other helper funcs that will analyze the actor'scrape
// infobox, and look for the actor's filmography/filmography page
func (s *Scraper) analyzeActorPage(url string, body io.Reader) {
	actorName, actorBirthdate := s.analyzeActorInfobox(s.findInfobox(body))

	var z *html.Tokenizer

	href, err := s.findFilmographyPage(body)
	if err != nil {
		fmt.Println(err)
		z = html.NewTokenizer(body)
	} else { // we found the link
		fmt.Println("Going to actor's filmography page...")
		resp, err := http.Get(wikipedia + href)
		if err != nil {
			fmt.Println(err)
			return
		}
		z = html.NewTokenizer(resp.Body)
	}

	// analyzeFilmography
	z, err = s.findFilmographySection(z)
	if err != nil {
		fmt.Println(err)
		return // couldnt find filmography for this actor
	}
	actorMovies := s.analyzeFilmography(z)

	// Create a new actornode
	fmt.Printf("Adding actor...\n\turl: %s\n\tname: %s\n\tbirthdate: %v\n\tmovies: %v\n\n", url, actorName, actorBirthdate, actorMovies)
	s.g.AddActor(url, actorName, actorBirthdate, actorMovies...)
}

// analyzeFilmography will scan through the page and look for a <span> with an id
// of "Films" or "Filmography" and then scan through the next <table> and collect all the movie urls
func (s *Scraper) findFilmographySection(z *html.Tokenizer) (*html.Tokenizer, error) {
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken: // end of doc reached
			return z, errors.New("Could not find filmography section")
		case tt == html.StartTagToken:
			t := z.Token()

			if t.Data == "span" && t.Attr != nil {
				for _, attr := range t.Attr {
					// we found it
					if attr.Key == "id" && (attr.Val == "Film" || attr.Val == "Films" || attr.Val == "Filmography") {
						return z, nil
					}
				}
			}
		}
	}
}

// analyzeFilmography will use the passed in Tokenizer and just scan through the first
// <table> it finds and look for <i> <a href> Tokens which will be movie urls
func (s *Scraper) analyzeFilmography(z *html.Tokenizer) []string {
	movies := make([]string, 0)

	for {
		tt := z.Next()

		switch {
		case tt == html.EndTagToken:
			t := z.Token()
			if t.Data == "table" {
				// we're done scanning through the filmography <table>
				return movies
			}
		case tt == html.StartTagToken:
			t := z.Token()

			if t.Data == "i" {
				tt = z.Next()
				t = z.Token()
				if t.Data == "a" {
					movies = append(movies, wikipedia+t.Attr[0].Val) // Might break - we are assuming 'href' is the first attribute
				}
			}
		}
	}
}

// analyzeActorInfobox will go through the Tokenizer's tokens until it hits
// an ending /table tag, which means we are done processing the infobox
func (s *Scraper) analyzeActorInfobox(z *html.Tokenizer) (string, time.Time) {
	var name string
	var birthDate time.Time

	for {
		tt := z.Next()

		switch {
		case tt == html.EndTagToken:
			t := z.Token()

			if t.Data == "table" {
				// Then we are done with this infobox
				fmt.Println("Finished with infobox")
				return name, birthDate
			}
		case tt == html.StartTagToken:
			t := z.Token()

			// if we've found a span, it may be something we're looking for
			// i.e. Actor's birthdate, or name
			if t.Data == "span" && t.Attr != nil && t.Attr[0].Key == "class" {
				if t.Attr[0].Val == "fn" { // Actor's name
					tt = z.Next()
					t = z.Token()
					name = t.Data
					fmt.Println("actor's name:", t.Data)
				} else if t.Attr[0].Val == "bday" { // Actor's birthdate
					tt = z.Next()
					t = z.Token()
					birthDate, _ = time.Parse(timeFormat, t.Data)
					fmt.Println("actor's birthdate:", t.Data)
				}
			}
		}
	}
}

// findFilmography will analyze the actor's page and look for a filmography link
func (s *Scraper) findFilmographyPage(body io.Reader) (string, error) {
	z := html.NewTokenizer(body)
	href := ""
	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			return href, errors.New("could not find filmography page")
		case tt == html.StartTagToken:
			t := z.Token()

			// if we've found an anchor, check its attributes
			if t.Data == "a" && t.Attr != nil {
				for _, attr := range t.Attr {
					switch {
					case attr.Key == "href":
						href = attr.Val
					case attr.Key == "title" && strings.Contains(attr.Val, "filmography"):
						return href, nil
					}
				}
			}
		}
	}
}
