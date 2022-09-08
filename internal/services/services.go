package services

import (
	"log"
	"sync"

	"github.com/ArtemVoronov/indefinite-studies-posts-service/internal/services/posts"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/app"
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
		log.Fatalf("unable to load TLS credentials")
	}
	feedcreds, err := app.LoadTLSCredentialsForClient(utils.EnvVar("FEED_SERVICE_CLIENT_TLS_CERT_PATH"))
	if err != nil {
		log.Fatalf("unable to load TLS credentials")
	}

	db := db.CreatePostgreSQLService()

	return &Services{
		auth:  auth.CreateAuthGRPCService(utils.EnvVar("AUTH_SERVICE_GRPC_HOST")+":"+utils.EnvVar("AUTH_SERVICE_GRPC_PORT"), &authcreds),
		feed:  feed.CreateFeedBuilderGRPCService(utils.EnvVar("FEED_SERVICE_GRPC_HOST")+":"+utils.EnvVar("FEED_SERVICE_GRPC_PORT"), &feedcreds),
		db:    db,
		posts: posts.CreatePostsService(db),
	}
}

func (s *Services) Shutdown() {
	s.feed.Shutdown()
	s.auth.Shutdown()
	s.db.Shutdown()
	s.posts.Shutdown()
}

func (s *Services) DB() *db.PostgreSQLService {
	return s.db
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
