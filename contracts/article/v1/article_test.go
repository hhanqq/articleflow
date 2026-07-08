package articlev1

import "testing"

func TestArticleValidate(t *testing.T) {
	article := Article{
		SourceName: "habr",
		URL:        "https://habr.com/ru/articles/123/",
		Title:      "Go microservices",
	}

	if err := article.Validate(); err != nil {
		t.Fatalf("expected valid article, got error: %v", err)
	}
}

func TestArticleValidateRequiresTitle(t *testing.T) {
	article := Article{
		SourceName: "habr",
		URL:        "https://habr.com/ru/articles/123/",
	}

	if err := article.Validate(); err == nil {
		t.Fatal("expected validation error for empty title")
	}
}

func TestSearchQueryNormalizeAndValidate(t *testing.T) {
	query := SearchQuery{Text: "  путешествие в китай  "}

	normalized := query.Normalize()

	if normalized.Text != "путешествие в китай" {
		t.Fatalf("unexpected text: %s", normalized.Text)
	}
	if normalized.Limit != 20 {
		t.Fatalf("expected default limit 20, got %d", normalized.Limit)
	}
	if err := normalized.Validate(); err != nil {
		t.Fatalf("expected valid query, got %v", err)
	}
}

func TestSearchQueryValidateRequiresText(t *testing.T) {
	if err := (SearchQuery{}).Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
