package cmd

import (
	"devflow/internal/bitbucket"
	"testing"
)

func makeComment(id int, parentID *int, resolved bool) bitbucket.Comment {
	c := bitbucket.Comment{
		ID:       id,
		Resolved: resolved,
		User: struct {
			DisplayName string `json:"display_name"`
			UUID        string `json:"uuid"`
		}{DisplayName: "user", UUID: "uuid"},
		CreatedOn: "now",
		Content: struct {
			Raw string `json:"raw"`
		}{Raw: "content"},
	}
	if parentID != nil {
		c.Parent = &struct {
			ID int `json:"id"`
		}{ID: *parentID}
	}
	return c
}

func TestOrganizeThreads_Basic(t *testing.T) {
	c1 := makeComment(1, nil, false)
	pid := 1
	c2 := makeComment(2, &pid, false)
	c3 := makeComment(3, &pid, true)
	comments := []bitbucket.Comment{c1, c2, c3}
	threads := organizeThreads(comments)
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread got %d", len(threads))
	}
	thread := threads[0]
	if thread.RootComment.ID != 1 {
		t.Errorf("thread root should be 1 got %d", thread.RootComment.ID)
	}
	if len(thread.Replies) != 2 {
		t.Errorf("expected 2 replies got %d", len(thread.Replies))
	}
}

func TestOrganizeThreads_MultipleRoots(t *testing.T) {
	c1 := makeComment(1, nil, false)
	pid1 := 1
	c2 := makeComment(2, &pid1, false)
	c3 := makeComment(3, nil, true)
	pid3 := 3
	c4 := makeComment(4, &pid3, false)
	comments := []bitbucket.Comment{c1, c2, c3, c4}
	threads := organizeThreads(comments)
	if len(threads) != 2 {
		t.Fatalf("expected 2 threads got %d", len(threads))
	}
	if threads[0].RootComment.ID != 1 || threads[1].RootComment.ID != 3 {
		t.Errorf("unexpected root IDs: %d, %d", threads[0].RootComment.ID, threads[1].RootComment.ID)
	}
	if len(threads[0].Replies) != 1 || len(threads[1].Replies) != 1 {
		t.Errorf("unexpected replies distribution")
	}
}

func TestOrganizeThreads_Empty(t *testing.T) {
	comments := []bitbucket.Comment{}
	threads := organizeThreads(comments)
	if len(threads) != 0 {
		t.Errorf("expected 0 threads got %d", len(threads))
	}
}
