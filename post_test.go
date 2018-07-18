package cli

import (
	"testing"
)

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

// TODO read this in from files
func TestPostParseMarkdown(t *testing.T) {
	t.Parallel()

	md := `
Today is my last day at The New York Times. Working here for the last two years
has been the most incredible experience of my career up to this point. I have
learned so much, I have grown as a person and as an engineer, and I have worked
with the best damn team that I could have asked for. I will miss you all
greatly.

Tomorrow I get to start an adventure that I have wanted to go on for many years.
I've waited for this, preparing financially and professionally for the day that
I would feel comfortable leaving a steady income and jumping into a big unknown.
I will be traveling for most of the year with few concrete plans, which is
exciting and scary and I can't wait.

If you want to follow along I will be posting regular updated here and on my
[Instagram][ig] and [YouTube][yt] accounts!

[ig]: https://www.instagram.com/brianfoshee/
[yt]: https://www.youtube.com/channel/UCCnxocnbh74guQll8yqWGUA
	`

	expected := `<p>Today is my last day at The New York Times. Working here for the last two years
has been the most incredible experience of my career up to this point. I have
learned so much, I have grown as a person and as an engineer, and I have worked
with the best damn team that I could have asked for. I will miss you all
greatly.</p>

<p>Tomorrow I get to start an adventure that I have wanted to go on for many years.
I&rsquo;ve waited for this, preparing financially and professionally for the day that
I would feel comfortable leaving a steady income and jumping into a big unknown.
I will be traveling for most of the year with few concrete plans, which is
exciting and scary and I can&rsquo;t wait.</p>

<p>If you want to follow along I will be posting regular updated here and on my
<a href="https://www.instagram.com/brianfoshee/">Instagram</a> and <a href="https://www.youtube.com/channel/UCCnxocnbh74guQll8yqWGUA">YouTube</a> accounts!</p>
`

	var p Post
	if err := p.ParseMarkdown([]byte(md)); err != nil {
		t.Fatal("error parsing markdown", err)
	}

	if p.Body != expected {
		t.Fatalf("body expected %q\ngot %q", expected, p.Body)
	}
}

func TestPostParseFile(t *testing.T) {
	var p Post
	if err := p.processFile("fixtures/im-taking-a-year-off.md"); err != nil {
		t.Fatal("error processing file", err)
	}

	if p.Title != "I'm Taking a Year Off" {
		t.Fatalf("title expected %q, got %q", "I'm Taking a Year Off", p.Title)
	}
}
