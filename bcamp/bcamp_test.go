package bcamp

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"flag"

	"bytes"
	"encoding/json"
	"strings"
)

// supply -update flag to update all golden files
var updateFlag = flag.Bool("update", false, "Update the golden files.")

func init() {
	flag.Parse()
}

func writeGoldenFile(name string, data interface{}) error {
	f, err := os.OpenFile(fmt.Sprintf("%s.golden", name), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0775)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.Encode(data)
	return nil
}

func compareGoldenFile(name string, data interface{}, t *testing.T) error {
	f, err := os.OpenFile(fmt.Sprintf("%s.golden", name), os.O_RDONLY, 0775)
	if err != nil {
		return err
	}
	defer f.Close()

	fileContent, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Failed to read file content: %s", err.Error())
	}

	buff := bytes.NewBufferString("")
	enc := json.NewEncoder(buff)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("Failed to encode data: %s", err.Error())
	}

	if dataString := buff.String(); dataString != string(fileContent) {
		t.Errorf("golden file did not match. Expected:\n %s \n\n Actually:\n %s", string(fileContent), dataString)
	}
	return nil
}

func getFixture(name string) string {
	file, err := os.Open(name)
	if err != nil {
		log.Fatalf("Failed to open test fixture (%s): %s", name, err.Error())
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("Failed to read test fixture (%s): %s", name, err.Error())
	}
	return string(bytes)
}

func getServer(content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, content)
	}))
}

func TestBandcampGetArtistPageInfo(t *testing.T) {

	tests := []struct {
		Fixture string
	}{
		{Fixture: "fixture/turboinferno-artist-page.html"},
		{Fixture: "fixture/1981-artist-page.html"},
	}

	for _, test := range tests {
		func() {
			srv := getServer(getFixture(test.Fixture))
			defer srv.Close()

			bc := Bandcamp{HTTP: http.DefaultClient}
			info, err := bc.GetArtistPageInfo(srv.URL)
			if err != nil {
				t.Errorf("Unexpected error: %s", err.Error())
			}
			if *updateFlag {
				writeGoldenFile(test.Fixture, info)
			}

			if err = compareGoldenFile(test.Fixture, info, t); err != nil {
				t.Errorf("Failed to compare result with golden file due to error: %s", err.Error())
			}
		}()
	}
}

func TestBandcampSearch(t *testing.T) {

	tests := []struct {
		Query string
		Fixture string
	}{
		{Query: "Turbo Inferno", Fixture: "fixture/turboinferno-search-page.html"},
		{Query: "1981", Fixture: "fixture/1981-search-page.html"},
	}

	for _, test := range tests {
		func() {
			srv := getServer(getFixture(test.Fixture))
			defer srv.Close()

			//override search URI module var
			SearchURI = srv.URL

			bc := Bandcamp{HTTP: http.DefaultClient}
			results, err := bc.Search(test.Query, "na", 10)
			if err != nil {
				t.Errorf("Unexpected error: %s", err.Error())
			}
			if *updateFlag {
				writeGoldenFile(test.Fixture, results)
			}

			if err = compareGoldenFile(test.Fixture, results, t); err != nil {
				t.Errorf("Failed to compare result with golden file due to error: %s", err.Error())
			}
		}()
	}
}

func TestTransformEmbed(t *testing.T) {

	tests := []struct {
		Original string
		UpdateAttrs map[string]string
	}{
		{
			//update attr
			Original: "http://bandcamp.com/EmbeddedPlayer/album=905056075/size=small/bgcol=ffffff/linkcol=0687f5/transparent=true/",
			UpdateAttrs: map[string]string{"size": "large"},
		},
		{
			//add attr
			Original: "http://bandcamp.com/EmbeddedPlayer/album=905056075/size=large/bgcol=ffffff/linkcol=0687f5/tracklist=false/transparent=true/",
			UpdateAttrs: map[string]string{"artwork": "small"},
		},
		{
			//misc
			Original: "http://bandcamp.com/EmbeddedPlayer/album=905056075/size=large/bgcol=ffffff/linkcol=0687f5/tracklist=false/transparent=true/",
			UpdateAttrs: map[string]string{"artwork": "small", "bgcol": "000000", "tracklist": "false"},
		},
	}

	for _, test := range tests {
		new := TransformEmbed(test.Original, test.UpdateAttrs)
		//check for hostname
		if ! strings.Contains(new, EmbedPrefix) {
			t.Errorf("Failed to find prefix in embed URL %s", new)
		}
		//check for updated attrs
		for k, v := range test.UpdateAttrs {
			expectedKV :=  fmt.Sprintf("%s=%s/", k, v)
			if ! strings.Contains(new,expectedKV) {
				t.Errorf("Failed to find updated attr %s in embed URL %s", expectedKV, new)
			}
		}
	}
}