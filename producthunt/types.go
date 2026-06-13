package producthunt

import "encoding/xml"

// Product is the record emitted for each Product Hunt launch from the feed.
type Product struct {
	Rank      int    `json:"rank"`
	Name      string `json:"name"`
	Tagline   string `json:"tagline"`
	Author    string `json:"author"`
	Published string `json:"published"`
	URL       string `json:"url"`
}

// atomFeed is the wire type for the Atom feed root element.
type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

// atomEntry is the wire type for a single Atom feed entry.
type atomEntry struct {
	ID    string `xml:"id"`
	Title string `xml:"title"`
	Link  struct {
		Href string `xml:"href,attr"`
	} `xml:"link"`
	Author struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Published string `xml:"published"`
	Updated   string `xml:"updated"`
	Content   string `xml:"content"`
}
