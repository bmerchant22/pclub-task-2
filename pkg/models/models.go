package models

import (
	"github.com/google/go-github/v37/github"
	"time"
)

type UserDetails struct {
	Username       string     `json:"username"`
	AvatarURL      string     `json:"avatar_url"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Bio            string     `json:"bio"`
	Location       string     `json:"location"`
	Followers      []UserLink `json:"followers"`
	Following      []UserLink `json:"following"`
	PublicRepos    int        `json:"public_repos"`
	Organizations  []string   `json:"organizations"`
	RecentActivity []Event    `json:"recent_activity"`
	MostUsedLang   string     `json:"most_used_lang"`
}

type UserLink struct {
	Username string `json:"username"`
	Link     string `json:"link"`
}

type RepoLink struct {
	RepoName string `json:"repo_name"`
	Link     string `json:"link"`
}

type Event struct {
	Type      string    `json:"type"`
	RepoName  string    `json:"repo_name"`
	CreatedAt time.Time `json:"created_at"`
}

type Repo struct {
	Name          string           `json:"name"`
	Desc          string           `json:"desc"`
	Lang          string           `json:"lang"`
	Clone         string           `json:"clone"`
	CreatedAt     github.Timestamp `json:"created_at"`
	Collaborators []UserLink       `json:"collaborators"`
	Contributors  []UserLink       `json:"contributors"`
	ForkCount     int              `json:"fork_count"`
	WatchersCount int              `json:"watchers_count"`
	Commits       []Commit         `json:"commits"`
}

type Commit struct {
	Author        string    `json:"author"`
	CommitMessage string    `json:"commit_message"`
	CreatedAt     time.Time `json:"created_at"`
}
