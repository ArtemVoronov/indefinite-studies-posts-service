package services

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/posts"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/shard"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

type Services struct {
	auth  *auth.AuthGRPCService
	feed  *feed.FeedBuilderGRPCService
	db    *db.PostgreSQLService
	posts *posts.PostsService
}

var once sync.Once
var instance *Services

func Instance() *Services {
	once.Do(func() {
		if instance == nil {
			instance = createServices()
		}
	})
	return instance
}

func createServices() *Services {
	authcreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("AUTH_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials: %s", err)
	}
	feedcreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("FEED_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials: %s", err)
	}

	dbClients := []*db.PostgreSQLService{}
	for i := 1; i <= shard.DEFAULT_BUCKET_FACTOR; i++ {
		dbConfig := &db.DBParams{
			Host:         utils.EnvVar("DATABASE_HOST"),
			Port:         utils.EnvVar("DATABASE_PORT"),
			Username:     utils.EnvVar("DATABASE_USER"),
			Password:     utils.EnvVar("DATABASE_PASSWORD"),
			DatabaseName: utils.EnvVar("DATABASE_NAME_PREFIX") + "_" + strconv.Itoa(i),
			SslMode:      utils.EnvVar("DATABASE_SSL_MODE"),
		}
		dbClients = append(dbClients, db.CreatePostgreSQLService(dbConfig))
	}

	return &Services{
		auth:  auth.CreateAuthGRPCService(utils.EnvVar("AUTH_SERVICE_GRPC_HOST")+":"+utils.EnvVar("AUTH_SERVICE_GRPC_PORT"), &authcreds),
		feed:  feed.CreateFeedBuilderGRPCService(utils.EnvVar("FEED_SERVICE_GRPC_HOST")+":"+utils.EnvVar("FEED_SERVICE_GRPC_PORT"), &feedcreds),
		posts: posts.CreatePostsService(dbClients),
	}
}

func (s *Services) Shutdown() error {
	result := []error{}
	err := s.feed.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.auth.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.db.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	err = s.posts.Shutdown()
	if err != nil {
		result = append(result, err)
	}
	if len(result) > 0 {
		return fmt.Errorf("errors during shutdown: %v", result)
	}
	return nil
}

func (s *Services) Auth() *auth.AuthGRPCService {
	return s.auth
}

func (s *Services) Feed() *feed.FeedBuilderGRPCService {
	return s.feed
}

func (s *Services) Posts() *posts.PostsService {
	return s.posts
}
