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

