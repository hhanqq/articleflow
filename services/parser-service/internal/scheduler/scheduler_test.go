package scheduler

import (
	"context"
	"testing"
	"time"

	parserv1 "github.com/hanq/articleflow/contracts/parser/v1"
)

type fakeJobStarter struct {
	started []parserv1.SearchQuery
}

func (starter *fakeJobStarter) StartAsync(_ context.Context, query parserv1.SearchQuery) (parserv1.ParserJob, error) {
	starter.started = append(starter.started, query)
	return parserv1.ParserJob{ID: "parser-job-test", Query: query, Status: parserv1.ParserJobStatusQueued}, nil
}

func TestSchedulerStartsConfiguredQueriesOnce(t *testing.T) {
	starter := &fakeJobStarter{}
	scheduler := New(starter, Config{
		Enabled: true,
		Queries: []Query{
			{Text: "go kafka", Sources: []string{"habr", "dzen"}, Limit: 50},
			{Text: "путешествие", Sources: []string{"vc"}, Limit: 30},
		},
		Interval: time.Hour,
	})

	err := scheduler.RunOnce(context.Background())

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(starter.started) != 2 {
		t.Fatalf("expected 2 started jobs, got %d", len(starter.started))
	}
	if starter.started[0].Text != "go kafka" || starter.started[0].Limit != 50 {
		t.Fatalf("unexpected first query: %#v", starter.started[0])
	}
	if len(starter.started[0].Sources) != 2 || starter.started[0].Sources[0] != "habr" || starter.started[0].Sources[1] != "dzen" {
		t.Fatalf("unexpected first sources: %#v", starter.started[0].Sources)
	}
}

func TestSchedulerSkipsWhenDisabled(t *testing.T) {
	starter := &fakeJobStarter{}
	scheduler := New(starter, Config{
		Enabled: false,
		Queries: []Query{
			{Text: "go kafka", Limit: 50},
		},
	})

	err := scheduler.RunOnce(context.Background())

	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if len(starter.started) != 0 {
		t.Fatalf("expected no started jobs, got %d", len(starter.started))
	}
}
