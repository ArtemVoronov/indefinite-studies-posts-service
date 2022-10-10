package services

import (
	"fmt"
	"sync"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/posts"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/feed"
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

	dbParamsShard1 := &db.DBParams{
		Host:         utils.EnvVar("DATABASE_HOST_SHARD_1"),
		Port:         utils.EnvVar("DATABASE_PORT_SHARD_1"),
		Username:     utils.EnvVar("DATABASE_USER_SHARD_1"),
		Password:     utils.EnvVar("DATABASE_PASSWORD_SHARD_1"),
		DatabaseName: utils.EnvVar("DATABASE_NAME_SHARD_1"),
		SslMode:      utils.EnvVar("DATABASE_SSL_MODE_SHARD_1"),
	}

	dbParamsShard2 := &db.DBParams{
		Host:         utils.EnvVar("DATABASE_HOST_SHARD_2"),
		Port:         utils.EnvVar("DATABASE_PORT_SHARD_2"),
		Username:     utils.EnvVar("DATABASE_USER_SHARD_2"),
		Password:     utils.EnvVar("DATABASE_PASSWORD_SHARD_2"),
		DatabaseName: utils.EnvVar("DATABASE_NAME_SHARD_2"),
		SslMode:      utils.EnvVar("DATABASE_SSL_MODE_SHARD_2"),
	}

	dbShard1 := db.CreatePostgreSQLService(dbParamsShard1)
	dbShard2 := db.CreatePostgreSQLService(dbParamsShard2)

	return &Services{
		auth:  auth.CreateAuthGRPCService(utils.EnvVar("AUTH_SERVICE_GRPC_HOST")+":"+utils.EnvVar("AUTH_SERVICE_GRPC_PORT"), &authcreds),
		feed:  feed.CreateFeedBuilderGRPCService(utils.EnvVar("FEED_SERVICE_GRPC_HOST")+":"+utils.EnvVar("FEED_SERVICE_GRPC_PORT"), &feedcreds),
		posts: posts.CreatePostsService(dbShard1, dbShard2),
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
