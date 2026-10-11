package gen

import (
	"slices"
	"strings"
	"testing"
)

func TestJSONLDCannotCloseTheScriptTag(t *testing.T) {
	got := string(jsonLD(schemaArticle{Headline: `</script><script>alert(1)</script>`}))

	if strings.Count(got, "</script>") != 1 {
		t.Errorf("payload closed the tag early: %s", got)
	}
	if strings.Contains(got, "<script>alert") {
		t.Errorf("angle brackets were not escaped: %s", got)
	}
}

func TestPersonAndOrganizationAreTheNodesOtherNodesReference(t *testing.T) {
	s := &site{URL: "https://x.test"}
	if personGraph(s).ID != authorRef(s).ID {
		t.Errorf("person %q is not the author %q", personGraph(s).ID, authorRef(s).ID)
	}
	if organizationGraph(s).ID != brandRef(s).ID {
		t.Errorf("organization %q is not the brand %q", organizationGraph(s).ID, brandRef(s).ID)
	}
	if websiteGraph(s).ID != websiteRef(s).ID {
		t.Errorf("website %q is not the website reference %q", websiteGraph(s).ID, websiteRef(s).ID)
	}
}

func TestArticleGraphUsesTheArticleSchemaType(t *testing.T) {
	if got := articleGraph(&site{URL: "https://x.test"}, article{SchemaType: "BlogPosting"}).Type; got != "BlogPosting" {
		t.Errorf("articleGraph().Type = %q, want BlogPosting", got)
	}
}

func TestWebPageGraphIsPartOfTheWebsite(t *testing.T) {
	s := &site{URL: "https://x.test"}
	got := webPageGraph(s, "ContactPage", page{OGTitle: "Contact", Path: contactFile})
	want := schemaWebPage{Type: "ContactPage", ID: "https://x.test/contact.html#page", Name: "Contact", URL: "https://x.test/contact.html", IsPartOf: websiteRef(s)}
	if got != want {
		t.Errorf("webPageGraph() = %+v, want %+v", got, want)
	}
}

func TestCollectionGraphListsTheArticles(t *testing.T) {
	s := &site{URL: "https://x.test"}
	got := collectionGraph(s, page{Path: "blog/"}, []article{{Title: "A", Path: "blog/a.html", SchemaType: "TechArticle"}})
	want := []schemaArticleSummary{{Type: "TechArticle", Headline: "A", URL: "https://x.test/blog/a.html"}}
	if !slices.Equal(got.HasPart, want) {
		t.Errorf("collectionGraph().HasPart = %+v, want %+v", got.HasPart, want)
	}
}

func TestBreadcrumbGraphNumbersTheCrumbsFromOne(t *testing.T) {
	s := &site{URL: "https://x.test"}
	got := breadcrumbGraph(s, []crumb{{"Home", ""}, {"Blog", "blog/"}})
	if len(got.Items) != 2 || got.Items[0].Position != 1 || got.Items[1].Position != 2 {
		t.Fatalf("breadcrumbGraph() = %+v, want positions 1 and 2", got.Items)
	}
	if got.Items[1].Item != "https://x.test/blog/" {
		t.Errorf("second crumb links to %q, want https://x.test/blog/", got.Items[1].Item)
	}
}
