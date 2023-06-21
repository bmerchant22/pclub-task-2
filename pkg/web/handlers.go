package web

import (
	"context"
	"fmt"
	"github.com/bmerchant22/pkg/models"
	"github.com/bmerchant22/pkg/store"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v37/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	githubOauth "golang.org/x/oauth2/github"
	"net/http"
	"strings"
	"time"
)

var globalTokenString string

const jwtKey = "752bd8773707d3843230a47397b707ca8c94a30f4c3b0e2467ac3806f0d49d2c"

var oauthConf = &oauth2.Config{
	ClientID:     "b14cc1fc1064eb52292b",
	ClientSecret: "2409e04f646feac1d19026f618d879bec893a88f",
	Scopes:       []string{"repo", "user", "offline_access"},
	Endpoint:     githubOauth.Endpoint,
}

type Token struct {
	UserID       int64
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

type Server struct {
	r     *gin.Engine
	store *store.MongoStore
}

func (srv *Server) handleHome(c *gin.Context) {
	c.JSON(http.StatusOK, "This is the home route")
}

func (srv *Server) handleLogin(c *gin.Context) {

	username := c.GetString("username")
	zap.S().Infof("Username retrieved while handling login : %v", username)

	accessToken, err := srv.store.GetAccessToken(username)
	zap.S().Infof("Accesstoken : %v and validation status of the token : %v", accessToken, srv.store.IsValidAccessToken(accessToken))
	if err == nil && srv.store.IsValidAccessToken(accessToken) {
		c.Redirect(http.StatusFound, "/callback")
		return
	}

	callbackURL := "http://localhost:8080/callback"
	url := oauthConf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("redirect_uri", callbackURL))
	zap.S().Infof(url)

	c.Redirect(http.StatusFound, url)
}

func (srv *Server) handleCallback(c *gin.Context) {
	code := c.Query("code")
	token, err := oauthConf.Exchange(context.Background(), code)
	if err != nil {
		zap.S().Errorf("OAuth exchange error: %v", err)
		c.Redirect(http.StatusFound, "/")
		return
	}

	client := github.NewClient(oauthConf.Client(context.Background(), token))
	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		zap.S().Errorf("Failed to get user information: %v", err)
		c.Redirect(http.StatusFound, "/")
		return
	}

	accessToken := token.AccessToken
	refreshToken := token.RefreshToken
	username := user.GetLogin()

	err = srv.store.InsertGithubTokens(username, accessToken, refreshToken)
	if err != nil {
		zap.S().Errorf("Failed to insert tokens: %v", err)
		c.Redirect(http.StatusFound, "/")
		return
	}

	expirationTime := time.Now().Add(time.Hour)
	claims := jwt.MapClaims{
		"username": username,
		"exp":      expirationTime.Unix(),
	}

	tokenString, err := store.GenerateToken(jwtKey, claims)
	if err != nil {
		zap.S().Errorf("Failed to generate JWT token: %v", err)
		c.Redirect(http.StatusFound, "http://localhost:8080/")
		return
	}
	c.Header("Authorization", "Bearer "+tokenString)

	globalTokenString = tokenString

	zap.S().Infof("Tokenstring : %v", tokenString)

	c.Redirect(http.StatusFound, "/user")
}

func (srv *Server) handleGetUserDetails(c *gin.Context) {
	username := c.GetString("username")
	zap.S().Infof("Username from c.GetString method is : %v", username)

	accessToken, err := srv.store.GetAccessToken(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve access token"})
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		zap.S().Errorf("Failed to get user details: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user details",
		})
		return
	}

	followers, _, err := client.Users.ListFollowers(ctx, username, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch followers"})
		return
	}

	following, _, err := client.Users.ListFollowing(ctx, username, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch following"})
		return
	}
	followerLinks := store.UserToLinkModel(followers)
	followingLinks := store.UserToLinkModel(following)

	orgs, _, err := client.Organizations.List(context.Background(), *user.Login, nil)
	if err != nil {
		zap.S().Errorf("Failed to get user organizations: %v", err)
		c.Redirect(http.StatusFound, "/")
		return
	}

	var orgNames []string
	for _, org := range orgs {
		orgNames = append(orgNames, *org.Login)
	}

	events, _, err := client.Activity.ListEventsPerformedByUser(ctx, username, false, nil)
	if err != nil {
		zap.S().Errorf("Error while fetching events for user :%v", err)
	}

	eventDetails := make([]models.Event, len(events))
	for i, event := range events {
		eventDetails[i] = models.Event{
			Type:      *event.Type,
			RepoName:  *event.Repo.Name,
			CreatedAt: *event.CreatedAt,
		}
	}
	if err != nil {
		zap.S().Errorf("Error while calculating total hours : %v", err)
	}

	mostUsedLanguage, err := store.CalculateMostUsedLanguage(ctx, client, username)
	if err != nil {
		zap.S().Errorf("Error while calculating most used language : %v", err)
	}

	repo, err := store.GetMostPopularRepo(ctx, client, username)
	if err != nil {
		zap.S().Errorf("Error while getting the most popular repo : %v", err)
	}

	mostPopularRepo := models.RepoLink{
		RepoName: repo.GetName(),
		Link:     fmt.Sprintf("http://localhost:8080/user-repos/%s", repo.GetName()),
	}

	userDetails := models.AuthorizedUserDetails{
		Username:        user.GetLogin(),
		AvatarURL:       user.GetAvatarURL(),
		Name:            user.GetName(),
		Email:           user.GetEmail(),
		Bio:             user.GetBio(),
		Location:        user.GetLocation(),
		Followers:       followerLinks,
		Following:       followingLinks,
		PublicRepos:     user.GetPublicRepos(),
		Organizations:   orgNames,
		RecentActivity:  eventDetails,
		MostUsedLang:    mostUsedLanguage,
		MostPopularRepo: mostPopularRepo,
	}

	c.JSON(http.StatusOK, userDetails)
}

func (srv *Server) handleUsers(c *gin.Context) {
	ctx := context.Background()
	username := c.Param("username")

	client := github.NewClient(nil)
	user, _, err := client.Users.Get(ctx, username)
	if err != nil {
		zap.S().Errorf("Failed to get repository information: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get repository information"})
		return
	}

	followers, _, err := client.Users.ListFollowers(ctx, username, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch followers"})
		return
	}

	following, _, err := client.Users.ListFollowing(ctx, username, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch following"})
		return
	}

	repos, _, err := client.Repositories.List(context.Background(), username, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching private repos from github"})
	}

	repoLinks := make([]models.RepoLink, len(repos))
	for i, repo := range repos {
		repoLinks[i] = models.RepoLink{
			RepoName: repo.GetName(),
			Link:     fmt.Sprintf("http://localhost:8080/repos/%s/%s", username, repo.GetName()),
		}
	}

	followerLinks := store.UserToLinkModel(followers)
	followingLinks := store.UserToLinkModel(following)

	repo, err := store.GetMostPopularRepo(ctx, client, username)
	if err != nil {
		zap.S().Errorf("Error while getting the most popular repo : %v", err)
	}

	mostPopularRepo := models.RepoLink{
		RepoName: repo.GetName(),
		Link:     fmt.Sprintf("http://localhost:8080/user-repos/%s", repo.GetName()),
	}

	userDetails := models.UserDetails{
		Username:        user.GetLogin(),
		AvatarURL:       user.GetAvatarURL(),
		Name:            user.GetName(),
		Email:           user.GetEmail(),
		Bio:             user.GetBio(),
		Location:        user.GetLocation(),
		Followers:       followerLinks,
		Following:       followingLinks,
		PublicRepos:     user.GetPublicRepos(),
		RepoLinks:       repoLinks,
		MostPopularRepo: mostPopularRepo,
	}

	c.JSON(http.StatusOK, userDetails)
}

func (srv *Server) handleUserRepos(c *gin.Context) {
	username := c.GetString("username")
	zap.S().Infof("Username from c.GetString method is : %v", username)

	accessToken, err := srv.store.GetAccessToken(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve access token"})
		return
	}

	client := github.NewClient(oauthConf.Client(context.Background(), &oauth2.Token{AccessToken: accessToken}))

	repos, _, err := client.Repositories.List(context.Background(), "", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching private repos from github"})
	}

	repoLinks := make([]models.RepoLink, len(repos))
	for i, repo := range repos {
		repoLinks[i] = models.RepoLink{
			RepoName: repo.GetName(),
			Link:     fmt.Sprintf("http://localhost:8080/user-repos/%s", repo.GetName()),
		}
	}

	c.JSON(http.StatusOK, repoLinks)
}

func (srv *Server) handleGetRepository(c *gin.Context) {
	username := c.Param("username")
	repoName := c.Param("repo")

	client := github.NewClient(nil)
	repo, _, err := client.Repositories.Get(context.Background(), username, repoName)
	if err != nil {
		zap.S().Errorf("Failed to get repository information: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get repository information"})
		return
	}

	var lang string
	if repo.Language != nil {
		lang = *repo.Language
	}

	response := models.Repo{
		Name:         repo.GetName(),
		Desc:         repo.GetDescription(),
		Lang:         lang,
		Clone:        *repo.CloneURL,
		ForkCount:    *repo.ForksCount,
		StarredCount: *repo.StargazersCount,
		CreatedAt:    *repo.CreatedAt,
	}

	c.JSON(http.StatusOK, response)
}

func (srv *Server) handleGetPrivateRepo(c *gin.Context) {
	repoName := c.Param("repo")
	username := c.GetString("username")
	zap.S().Infof("Username from c.GetString method is : %v", username)

	accessToken, err := srv.store.GetAccessToken(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve access token"})
		return
	}

	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	oauthClient := oauth2.NewClient(ctx, tokenSource)
	client := github.NewClient(oauthClient)

	repo, _, err := client.Repositories.Get(ctx, username, repoName)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to get repository")
		return
	}

	collaborators, _, err := client.Repositories.ListCollaborators(ctx, username, repoName, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Failed to get collaborators on the given repo")
	}

	collaboratorLinks := store.UserToLinkModel(collaborators)

	contributors, _, err := client.Repositories.ListContributors(ctx, username, repoName, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Failed to get contributors on the given repo")
	}

	contributorLinks := make([]models.UserLink, len(contributors))
	for i, contributor := range contributors {
		contributorLinks[i] = models.UserLink{
			Username: contributor.GetLogin(),
			Link:     fmt.Sprintf("http://localhost:8080/users/%s", contributor.GetLogin()),
		}
	}
	var lang string
	if repo.Language != nil {
		lang = *repo.Language
	}

	commits, _, err := client.Repositories.ListCommits(ctx, username, repoName, nil)

	commitDetails := make([]models.Commit, len(commits))
	for i, commit := range commits {
		commitDetails[i] = models.Commit{
			Author:        *commit.Author.Login,
			CommitMessage: *commit.Commit.Message,
			CreatedAt:     commit.Commit.Committer.GetDate(),
		}
	}

	response := models.AuthorizedRepo{
		Name:          repo.GetName(),
		Desc:          repo.GetDescription(),
		Lang:          lang,
		Clone:         *repo.CloneURL,
		CreatedAt:     *repo.CreatedAt,
		Collaborators: collaboratorLinks,
		Contributors:  contributorLinks,
		ForkCount:     *repo.ForksCount,
		StarredCount:  *repo.StargazersCount,
		Commits:       commitDetails,
	}

	c.JSON(http.StatusOK, response)
}

func (srv *Server) handleListPrivateRepos(c *gin.Context) {
	username := c.GetString("username")
	zap.S().Infof("Username from c.GetString method is : %v", username)

	accessToken, err := srv.store.GetAccessToken(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve access token"})
		return
	}

	ctx := context.Background()
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	oauthClient := oauth2.NewClient(ctx, tokenSource)
	client := github.NewClient(oauthClient)

	opt := &github.RepositoryListOptions{
		Visibility: "private",
	}

	repos, _, err := client.Repositories.List(context.Background(), "", opt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while fetching private repos from github"})
	}

	repoLinks := make([]models.RepoLink, len(repos))
	for i, repo := range repos {
		repoLinks[i] = models.RepoLink{
			RepoName: repo.GetName(),
			Link:     fmt.Sprintf("http://localhost:8080/user-repos/%s", repo.GetName()),
		}
	}

	c.JSON(http.StatusOK, repoLinks)
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized BM debug"})
			c.Abort()
			return
		}
		zap.S().Infof("Auth Header : %v", authHeader)

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token"})
			return
		}
		tokenString := parts[1]
		zap.S().Infof("Token string in the auth header is given as : %v", tokenString)

		token, err := store.ValidateToken(tokenString, jwtKey)
		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token signature"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			}
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		zap.S().Infof("Claims of the jwt token are : %v", claims)

		username, ok := claims["username"].(string)
		zap.S().Infof("Username retrieved by authMiddleware : %v", username)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		c.Set("username", username)
		c.Next()
	}
}

func setAuthHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Set("Authorization", "Bearer "+globalTokenString)
		c.Next()
	}
}
