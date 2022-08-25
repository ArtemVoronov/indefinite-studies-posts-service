package app

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/ping"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/api/rest/v1/posts"
	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
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
	authorized.Use(authReqired())
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

// TODO: unify
func authReqired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		// fmt.Println("---------------AuthReqired---------------")
		// fmt.Printf("header: %v\n", header)
		// fmt.Println("---------------AuthReqired---------------")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer") {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		token := authHeader[len("Bearer "):]
		verificationResult, err := services.Instance().Auth().VerifyToken(token)

		if err != nil {
			c.JSON(http.StatusInternalServerError, "Internal Server Error")
			log.Printf("error during verifying access token: %v\n", err)
			c.Abort()
			return
		}

		if (*verificationResult).IsExpired {
			c.JSON(http.StatusUnauthorized, "Unauthorized")
			c.Abort()
			return
		}

		c.Next()
	}
}
