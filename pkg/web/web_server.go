package web

import (
	"github.com/bmerchant22/pkg/store"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func CreateWebServer(store *store.MongoStore) *Server {

	srv := new(Server)
	srv.store = store
	srv.r = gin.Default()

	srv.r.GET(kHome, srv.handleHome)
	srv.r.GET(kLogin, srv.handleLogin)
	srv.r.GET(kUsers, srv.handleUsers)
	srv.r.GET(kUser, setAuthHeaderMiddleware(), authMiddleware(), srv.handleGetUserDetails)
	srv.r.GET(kCallback, setAuthHeaderMiddleware(), srv.handleCallback)
	srv.r.GET(kRepos, setAuthHeaderMiddleware(), authMiddleware(), srv.handleUserRepos)
	srv.r.GET(kIndividualRepos, srv.handleGetRepository)
	srv.r.GET(kPrivateReposList, setAuthHeaderMiddleware(), authMiddleware(), srv.handleListPrivateRepos)
	srv.r.GET(kUserRepos, setAuthHeaderMiddleware(), authMiddleware(), srv.handleGetPrivateRepo)

	if err := srv.r.Run("localhost:8080"); err != nil {
		zap.S().Errorf("Error while running the server !")
	}

	zap.S().Infof("Web server created successfully !!")

	return srv
}
