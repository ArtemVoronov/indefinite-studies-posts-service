package services

import (
	"net/http"
	"sync"
	"time"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/auth"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/services/db"
	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/utils"
)

type Services struct {
	auth *auth.AuthService
	db   *db.PostgreSQLService
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
	client := &http.Client{
		Timeout: utils.EnvVarDurationDefault("HTTP_CLIENT_REQUEST_TIMEOUT_IN_SECONDS", time.Second, 30*time.Second),
	}

	return &Services{
		auth: auth.CreateAuthService(client, utils.EnvVar("AUTH_SERVICE_BASE_URL")),
		db:   db.CreatePostgreSQLService(),
	}
}

func (s *Services) Shutdown() {
	s.auth.Shutdown()
	s.db.Shutdown()
}

func (s *Services) DB() *db.PostgreSQLService {
	return s.db
}

func (s *Services) Auth() *auth.AuthService {
	return s.auth
}
