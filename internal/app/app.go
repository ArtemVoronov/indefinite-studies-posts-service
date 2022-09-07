package app

import (
	"fmt"
	"net/http"

	commentsGrpcApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/grpc/v1/comments"
	postsGrpcApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/grpc/v1/posts"
	commentsRestApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/comments"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/ping"
	postsRestApi "github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/posts"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func Start() {
	app.LoadEnv()
	creds := app.TLSCredentials()
	go func() {
		app.StartGRPC(setup, shutdown, app.HostGRPC(), createGrpcApi, &creds)
	}()
	app.StartHTTP(setup, shutdown, app.HostHTTP(), createRestApi())
}

func setup() {
	services.Instance()
}

func shutdown() {
	services.Instance().Shutdown()
}

func createRestApi() *gin.Engine {
	router := gin.Default()
	gin.SetMode(app.Mode())
	router.Use(app.Cors())
	router.Use(gin.Logger())
	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.String(http.StatusInternalServerError, fmt.Sprintf("error: %s", err))
		}
		c.AbortWithStatus(http.StatusInternalServerError)
	}))

	// TODO: add permission controller by user role and user state
	v1 := router.Group("/api/v1")

	v1.GET("/posts/ping", ping.Ping)
	v1.GET("/posts", postsRestApi.GetPosts)
	v1.GET("/posts/:id", postsRestApi.GetPost)
	v1.GET("/posts/:id/comments", commentsRestApi.GetComments)

	authorized := router.Group("/api/v1")
	authorized.Use(app.AuthReqired(authenicate))
	{
		authorized.GET("/posts/debug/vars", expvar.Handler())
		authorized.GET("/posts/safe-ping", ping.SafePing)

		authorized.POST("/posts/", postsRestApi.CreatePost)
		authorized.PUT("/posts/", postsRestApi.UpdatePost)
		authorized.DELETE("/posts/", postsRestApi.DeletePost)

		authorized.POST("/posts/:id/comments", commentsRestApi.CreateComment)
		authorized.PUT("/posts/:id/comments", commentsRestApi.UpdateComment)
		authorized.DELETE("/posts/:id/comments", commentsRestApi.DeleteComment)
	}

	return router
}

func createGrpcApi(s *grpc.Server) {
	postsGrpcApi.RegisterServiceServer(s)
	commentsGrpcApi.RegisterServiceServer(s)
}

func authenicate(token string) (*auth.VerificationResult, error) {
	return services.Instance().Auth().VerifyToken(token)
}
