package app

import (
	"fmt"
	"net/http"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/posts"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"
)

func Start() {
	app.LoadEnv()
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

	v1.GET("/ping", ping.Ping)
	authorized := router.Group("/api/v1")
	authorized.Use(app.AuthReqired(authenicate))
	{
		authorized.GET("/debug/vars", expvar.Handler())
		authorized.GET("/safe-ping", ping.SafePing)

		authorized.GET("/posts", posts.GetPosts)
		authorized.GET("/posts/:id", posts.GetPost)
		authorized.POST("/posts", posts.CreatePost)
		authorized.PUT("/posts/:id", posts.UpdatePost)
		authorized.DELETE("/posts/:id", posts.DeletePost)
	}

	return router
}

func authenicate(token string) (*auth.VerificationResult, error) {
	return services.Instance().Auth().VerifyToken(token)
}
