package cli

import "testing"

func TestPostYAMLParse(t *testing.T) {
	t.Parallel()

	y := `
slug: one-week-off-grid
title: One Week Off-Grid
published-at: 2018-06-12
description: I turned off my computer and phone for a week to reboot my technology habits.
draft: true`

	var p Post
	if err := p.ParseYAML([]byte(y)); err != nil {
		t.Fatal("error parsing yaml:", err)
	}
	if p.Slug != "one-week-off-grid" {
		t.Errorf("slug expected %q, got %q", "one-week-off-grid", p.Slug)
	}
	if p.Title != "One Week Off-Grid" {
		t.Errorf("title expected %q, got %q", "One Week Off-Grid", p.Title)
	}
	ed := "I turned off my computer and phone for a week to reboot my technology habits."
	if p.Description != ed {
		t.Errorf("description expected %q, got %q", ed, p.Description)
	}
	if p.PublishedAt.Year() != 2018 || p.PublishedAt.Month() != 6 ||
		p.PublishedAt.Day() != 12 {
		t.Errorf("publishedAt expected 2018-06-12 got %s", p.PublishedAt.Format("2006-01-02"))
	}
	if p.Draft != true {
		t.Errorf("draft expected %v got %v", true, p.Draft)
	}
}
