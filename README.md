Go Bandcamp Search
========================

#### Search

Search fetches the majority of information on the search results
page including genre, url, artwork etc. for each entry.

Fetch exact name matches for "some band" in "berlin":

```
bcs := bcamp.Bandcamp{HTTP: http.DefaultClient}
results, err := bcs.Search("some band", "berlin", 0)
```

Fetch matches with somewhat similar names to "some band":

```
bcs := bcamp.Bandcamp{HTTP: http.DefaultClient}
results, err := bcs.Search("some band", "berlin", 2)
```

#### Artist Info

You can also fetch the bio and links from an artist info page:

```
bcs := Bandcamp{HTTP: http.DefaultClient}
info, err := bcs.GetArtistPageInfo("http://whatever.bandcamp.com")
```
