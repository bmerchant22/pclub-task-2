package store

import (
	"context"
	"fmt"
	"github.com/bmerchant22/pkg/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/go-github/v37/github"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type PostgresStore struct {
	db *gorm.DB
}

type MongoStore struct {
	Collection *mongo.Collection
}

type GithubToken struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Username     string             `bson:"username"`
	AccessToken  string             `bson:"access_token"`
	RefreshToken string             `bson:"refresh_token"`
	Expiry       time.Time          `bson:"expiry"`
}

const uri = "mongodb+srv://merchantburhanuddin484:GTpqeNUJbGMIIM1u@cluster0.h7cvikv.mongodb.net/"

func (m *MongoStore) ConnectToDatabase() {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(uri).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(context.TODO(), opts)

	if err != nil {
		panic(err)
	}

	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
		return
	}

	fmt.Println("Pinged the primary node of the cluster. You successfully connected to MongoDB! Collection : tokens")

	m.Collection = client.Database("auth-tokens").Collection("tokens")

}

func (m *MongoStore) InsertGithubTokens(username string, accessToken, refreshToken string) error {

	token := GithubToken{
		Username:     username,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       time.Now().Add(time.Hour),
	}

	_, err := m.Collection.InsertOne(context.Background(), token)
	if err != nil {
		zap.S().Errorf("Error while inserting auth token for user : %v", username)
		return err
	}

	return nil
}

func (m *MongoStore) GetAccessToken(username string) (string, error) {
	var token GithubToken
	err := m.Collection.FindOne(context.Background(), bson.M{"username": username}).Decode(&token)
	if err != nil {
		zap.S().Errorf("Error while fetching access tokens for user with username %v : %v", username, err)
		return "", err
	}
	zap.S().Infof("Access token : %v", token.AccessToken)
	return token.AccessToken, nil
}

func (m *MongoStore) IsValidAccessToken(accessToken string) bool {
	var token GithubToken
	err := m.Collection.FindOne(context.Background(), bson.M{"access_token": accessToken}).Decode(&token)
	if err != nil {
		// Handle error
	}

	// Check if the access token is expired
	if token.Expiry.Before(time.Now()) {
		return false
	}

	return true
}

func ValidateToken(tokenString, secretKey string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		zap.S().Errorf("failed to parse token: %v", err)
		return nil, err
	}

	if !token.Valid {
		zap.S().Errorf("invalid token")
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !claims.VerifyExpiresAt(time.Now().Unix(), true) {
		zap.S().Errorf("token has expired")
		return nil, err
	}

	return token, nil
}

func GenerateToken(secretKey string, claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func UserToLinkModel(users []*github.User) []models.UserLink {
	userLinks := make([]models.UserLink, len(users))
	for i, user := range users {
		userLinks[i] = models.UserLink{
			Username: user.GetLogin(),
			Link:     fmt.Sprintf("http://localhost:8080/users/%s", user.GetLogin()),
		}
	}

	return userLinks
}

func CalculateMostUsedLanguage(ctx context.Context, client *github.Client, username string) (string, error) {
	repos, _, err := client.Repositories.List(ctx, username, nil)
	if err != nil {
		return "", err
	}

	languageCount := make(map[string]int)

	for _, repo := range repos {
		owner := *repo.Owner.Login
		repoName := *repo.Name

		repoInfo, _, err := client.Repositories.Get(ctx, owner, repoName)
		if err != nil {
			return "", err
		}

		language := repoInfo.Language
		if language != nil {
			languageCount[*language]++
		}
	}

	mostUsedLanguage := ""
	maxCount := 0

	for language, count := range languageCount {
		if count > maxCount {
			maxCount = count
			mostUsedLanguage = language
		}
	}

	return mostUsedLanguage, nil
}

func GetMostPopularRepo(ctx context.Context, client *github.Client, username string) (*github.Repository, error) {
	repos, _, err := client.Repositories.List(ctx, username, nil)
	if err != nil {
		return nil, err
	}

	var mostPopularRepo *github.Repository
	maxStars := 0

	for _, repo := range repos {
		if repo.StargazersCount != nil && *repo.StargazersCount > maxStars {
			maxStars = *repo.StargazersCount
			mostPopularRepo = repo
		}
	}

	return mostPopularRepo, nil
}
