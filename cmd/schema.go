package main

type schemaRef struct {
	ID string `json:"@id"`
}

type schemaPageRef struct {
	Type string `json:"@type"`
	ID   string `json:"@id"`
}

type schemaPerson struct {
	Type     string   `json:"@type"`
	ID       string   `json:"@id"`
	Name     string   `json:"name"`
	JobTitle string   `json:"jobTitle,omitempty"`
	Email    string   `json:"email,omitempty"`
	URL      string   `json:"url"`
	SameAs   []string `json:"sameAs,omitempty"`
}

type schemaOrganization struct {
	Type string `json:"@type"`
	ID   string `json:"@id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Logo string `json:"logo"`
}

type schemaArticle struct {
	Type             string        `json:"@type"`
	ID               string        `json:"@id"`
	Headline         string        `json:"headline"`
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	InLanguage       string        `json:"inLanguage"`
	DatePublished    string        `json:"datePublished,omitempty"`
	DateModified     string        `json:"dateModified,omitempty"`
	Author           schemaRef     `json:"author"`
	Publisher        schemaRef     `json:"publisher"`
	Image            string        `json:"image"`
	MainEntityOfPage schemaPageRef `json:"mainEntityOfPage"`
	URL              string        `json:"url"`
}

type schemaArticleSummary struct {
	Type     string `json:"@type"`
	Headline string `json:"headline"`
	URL      string `json:"url"`
}

type schemaCollectionPage struct {
	Type        string                 `json:"@type"`
	ID          string                 `json:"@id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	URL         string                 `json:"url"`
	IsPartOf    schemaRef              `json:"isPartOf"`
	HasPart     []schemaArticleSummary `json:"hasPart"`
}

type schemaAboutPage struct {
	Type        string    `json:"@type"`
	ID          string    `json:"@id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	MainEntity  schemaRef `json:"mainEntity"`
	IsPartOf    schemaRef `json:"isPartOf"`
}

type schemaWebSite struct {
	Type          string    `json:"@type"`
	ID            string    `json:"@id"`
	Name          string    `json:"name"`
	AlternateName string    `json:"alternateName"`
	Description   string    `json:"description"`
	URL           string    `json:"url"`
	InLanguage    string    `json:"inLanguage"`
	Publisher     schemaRef `json:"publisher"`
	Author        schemaRef `json:"author"`
}

func (p schemaPerson) ref() schemaRef       { return schemaRef{ID: p.ID} }
func (o schemaOrganization) ref() schemaRef { return schemaRef{ID: o.ID} }
