package bcamp

import (
	"net/http"
	"net/url"
	"sort"
	"strings"
	"unicode"

	"fmt"

	"github.com/PuerkitoBio/goquery"
	"github.com/texttheater/golang-levenshtein/levenshtein"
)

var SearchURI = "https://bandcamp.com/search"

const EmbedPrefix = "http://bandcamp.com/EmbeddedPlayer/"

type Results []*Result

func (a Results) Len() int           { return len(a) }
func (a Results) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Results) Less(i, j int) bool { return a[i].Score < a[j].Score }

type Result struct {
	Name     string   `json:"name"`
	Location string   `json:"location"`
	URL      string   `json:"url"`
	Genre    string   `json:"genre"`
	Tags     []string `json:"tags"`
	Art      string   `json:"art_url"`
	Score    int      `json:"match_score"`
}

type ArtistPage struct {
	Bio   string
	Links []*Link
	Embed   string
}

type Link struct {
	URI  string
	Text string
}

type Bandcamp struct {
	HTTP *http.Client
}

func (b *Bandcamp) GetArtistPageInfo(artistURL string) (*ArtistPage, error) {

	a := &ArtistPage{Bio: "", Links: make([]*Link, 0)}
	if artistURL == "" {
		return a, fmt.Errorf("Artist URL cannot be blank")
	}

	searchPage, err := b.HTTP.Get(artistURL)
	if err != nil {
		return a, fmt.Errorf("Failed to fetch artist page: %s", err)
	}
	defer searchPage.Body.Close()
	doc, err := goquery.NewDocumentFromReader(searchPage.Body)
	if err != nil {
		return a, fmt.Errorf("Failed to read artist page: %s", err)
	}

	doc.Find("#bio-container").Each(func(i int, bioContainer *goquery.Selection) {
		a.Bio = strings.TrimSpace(bioContainer.Find(".signed-out-artists-bio-text meta").First().AttrOr("content", ""))
	})
	doc.Find("#band-links li a").Each(func(i int, atag *goquery.Selection) {
		a.Links = append(a.Links, &Link{URI: atag.AttrOr("href", ""), Text: strings.TrimSpace(atag.Text())})
	})

	doc.Find("meta[property=\"og:video\"]").Each(func(i int, metaTag *goquery.Selection) {
		a.Embed = metaTag.AttrOr("content", "")
	})

	return a, nil
}

func (b *Bandcamp) Search(name string, location string, maxScore int) (Results, error) {
	searchPage, err := b.HTTP.Get(fmt.Sprintf("%s?q=%s", SearchURI, url.QueryEscape(name+" "+location)))
	if err != nil {
		return nil, err
	}
	defer searchPage.Body.Close()

	doc, err := goquery.NewDocumentFromReader(searchPage.Body)
	if err != nil {
		return nil, err
	}

	//select the main data column and handle all the sub-tables
	results := make(Results, 0)
	doc.Find("#pgBd > div.search > div.leftcol > div > ul > .band").Each(func(i int, bandDiv *goquery.Selection) {
		result := b.processSearchResult(i, bandDiv)
		b.scoreResult(name, location, result)
		if result.Score <= maxScore {
			results = append(results, result)
		}
	})
	sort.Sort(results)
	return results, nil
}

func (b *Bandcamp) processSearchResult(i int, bandDiv *goquery.Selection) *Result {

	//data from directly on the search results page
	result := &Result{Tags: make([]string, 0), Score: i}
	result.Name = strings.TrimSpace(bandDiv.Find(".heading").First().Text())
	result.Location = strings.TrimSpace(bandDiv.Find(".subhead").First().Text())
	result.URL = strings.TrimSpace(bandDiv.Find(".itemurl").First().Text())
	result.Genre = strings.TrimPrefix(strings.TrimSpace(bandDiv.Find(".genre").First().Text()), "genre:")
	for _, tag := range strings.Split(strings.TrimPrefix(strings.TrimSpace(bandDiv.Find(".tags").First().Text()), "tags:"), ",") {
		result.Tags = append(result.Tags, strings.TrimSpace(tag))
	}
	result.Art = strings.TrimSpace(bandDiv.Find(".artcont .art img").First().AttrOr("src", ""))
	return result
}

func (b *Bandcamp) scoreResult(searchedName, searchedLocation string, result *Result) {
	opts := levenshtein.DefaultOptions
	opts.Matches = func(sourceCharacter rune, targetCharacter rune) bool {
		return unicode.ToLower(sourceCharacter) == unicode.ToLower(targetCharacter)
	}
	result.Score += levenshtein.DistanceForStrings([]rune(searchedName), []rune(result.Name), opts)
}


//http://bandcamp.com/EmbeddedPlayer/album=905056075/size=small/bgcol=ffffff/linkcol=0687f5/transparent=true/
//http://bandcamp.com/EmbeddedPlayer/album=905056075/size=small/bgcol=ffffff/linkcol=0687f5/artwork=none/transparent=true/
//http://bandcamp.com/EmbeddedPlayer/album=905056075/size=large/bgcol=ffffff/linkcol=0687f5/tracklist=false/transparent=true/

func TransformEmbed(orignalEmbed string, updatedAttrs map[string]string) string {


	attrs := map[string]string{}
	for _, attr := range strings.Split(strings.Replace(orignalEmbed, EmbedPrefix, "2", 0), "/") {
		if attrParts := strings.Split(attr, "="); len(attrParts) == 2 {
			attrs[attrParts[0]] = attrParts[1]
		}
	}

	for attrName, attrValue := range updatedAttrs {
		attrs[attrName] = attrValue
	}

	newEmbed := EmbedPrefix + "/"
	for k, v := range attrs {
		newEmbed += fmt.Sprintf("%s=%s/", k, v)
	}
	return newEmbed
}