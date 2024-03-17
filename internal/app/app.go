package app

import (
	"fmt"
	"net/http"

	postsGrpcApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/grpc/v1/posts"
	commentsRestApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/comments"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/ping"
	postsRestApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/posts"
	tagsRestApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/tags"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func Start() {
	app.LoadEnv()
	file := log.SetUpLogPath(utils.EnvVarDefault("APP_LOGS_PATH", "stdout"))
	if file != nil {
		defer file.Close()
	}
	creds := app.TLSCredentials()
	go func() {
		app.StartGRPC(setup, shutdown, app.HostGRPC(), createGrpcApi, &creds, log.Instance())
	}()
	app.StartHTTP(setup, shutdown, app.HostHTTP(), createRestApi(log.Instance()))
}

func setup() {
	services.Instance()
}

func shutdown() {
	err := services.Instance().Shutdown()
	log.Error("error during app shutdown", err.Error())
}

func createRestApi(logger *logrus.Logger) *gin.Engine {
	router := gin.Default()
	gin.SetMode(app.Mode())
	router.Use(app.Cors())
	router.Use(app.NewLoggerMiddleware(logger))
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	v1 := router.Group("/api/v1")

	v1.GET("/posts/ping", ping.Ping)
	v1.GET("/posts/:uuid", postsRestApi.GetPost)
	v1.GET("/posts/:uuid/comments", commentsRestApi.GetComments)
	v1.GET("/posts/tags", tagsRestApi.GetTags)
	v1.GET("/posts/tags/:id", tagsRestApi.GetTag)

	v1.GET("/posts/preview/:uuid", postsRestApi.GetPostPreview)

	authorized := router.Group("/api/v1")
	authorized.Use(app.AuthReqired(authenicate))
	{
		authorized.GET("/posts/debug/vars", app.RequiredOwnerRole(), expvar.Handler())
		authorized.GET("/posts/safe-ping", app.RequiredOwnerRole(), ping.SafePing)

		// TODO: after allowing to create posts for others need to add rule: ONLY OWNER and MODERATOR could change states from ON_MODERATION -> PUBLISHED
		// TODO: after allowing to create posts for others need to add rule: ONLY OWNER and MODERATOR or author of post could update it
		// TODO: after allowing to create posts for others need to add rule: ONLY OWNER and MODERATOR or author of post could delete it
		authorized.POST("/posts/", app.RequiredOwnerRole(), postsRestApi.CreatePost)
		authorized.PUT("/posts/", app.RequiredOwnerRole(), postsRestApi.UpdatePost)
		authorized.DELETE("/posts/", app.RequiredOwnerRole(), postsRestApi.DeletePost)

		authorized.POST("/posts/comments", commentsRestApi.CreateComment)
		authorized.PUT("/posts/comments", commentsRestApi.UpdateComment)
		authorized.DELETE("/posts/comments", app.RequiredOwnerRole(), commentsRestApi.DeleteComment)

		authorized.POST("/posts/tags/", app.RequiredOwnerRole(), tagsRestApi.CreateTag)
		authorized.PUT("/posts/tags/", app.RequiredOwnerRole(), tagsRestApi.UpdateTag)
		authorized.PUT("/posts/tags/assign", app.RequiredOwnerRole(), tagsRestApi.AssignTags)
		authorized.PUT("/posts/tags/remove", app.RequiredOwnerRole(), tagsRestApi.RemoveTags)
	}

	return router
}

func createGrpcApi(s *grpc.Server) {
	postsGrpcApi.RegisterServiceServer(s)
}

func authenicate(token string) (*auth.VerificationResult, error) {
	return services.Instance().Auth().VerifyToken(token)
}
