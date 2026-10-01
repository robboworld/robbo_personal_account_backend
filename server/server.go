package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/skinnykaen/robbo_student_personal_account.git/app/modules"
	"github.com/skinnykaen/robbo_student_personal_account.git/graph/generated"
	"github.com/skinnykaen/robbo_student_personal_account.git/package/lmsdb"
	"github.com/spf13/viper"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
)

// shutdownTimeout bounds draining in-flight requests on stop; app.StopTimeout and the
// compose stop_grace_period leave headroom above it.
const shutdownTimeout = 25 * time.Second

func NewServer(lifecycle fx.Lifecycle, graphQLModule modules.GraphQLModule, handlers modules.HandlerModule) {
	var server *http.Server
	lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				router := SetupGinRouter(handlers)
				if viper.GetBool("graphql.playground") {
					router.GET("/", playgroundHandler())
				}
				router.POST("/query", graphqlHandler(graphQLModule))
				router.Static("/frontend", "./frontend")
				router.GET("/frontend", func(c *gin.Context) {
					c.File("./frontend/index.html")
				})

				checks := []readinessCheck{{name: "lms_mysql", ping: lmsdb.Ping}}
				if p, ok := handlers.LicensingGateway.(pinger); ok {
					checks = append(checks, readinessCheck{name: "licensing_db", ping: p.Ping})
				}
				server = &http.Server{
					Addr:    viper.GetString("server.address"),
					Handler: healthMux(newCORS().Handler(router), checks),
					// Larger write window so large .sb3 downloads complete (BYTEA payloads).
					ReadTimeout:    120 * time.Second,
					WriteTimeout:   20 * time.Minute,
					MaxHeaderBytes: 1 << 20,
				}
				// Listen synchronously so a busy port fails startup instead of exiting later.
				ln, err := net.Listen("tcp", server.Addr)
				if err != nil {
					return err
				}

				log.Printf("connect to http://localhost:%s/ for GraphQL playground", viper.GetString("graphqlServer.port"))
				go func() {
					if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
						log.Fatalf("Failed to listen and serve: %s", err)
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				if server == nil {
					return nil
				}
				log.Println("http: shutting down, draining requests")
				ctx, cancel := context.WithTimeout(ctx, shutdownTimeout)
				defer cancel()
				if err := server.Shutdown(ctx); err != nil {
					log.Printf("http: shutdown: %v", err)
					return err
				}
				log.Println("http: stopped")
				return nil
			},
		})
}

func SetupGinRouter(handlers modules.HandlerModule) *gin.Engine {
	ginMode()
	// gin.New: gin.Default already adds Logger + Recovery, which were added again below.
	router := gin.New()
	if err := applyTrustedProxies(router); err != nil {
		log.Fatalf("invalid trusted proxies (server.trustedProxies / TRUSTED_PROXIES): %v", err)
	}
	router.Use(
		requestID(),
		accessLog(),
		gin.Recovery(),
		GinContextToContextMiddleware(),
		TokenAuthMiddleware(handlers.LicensingGateway),
		CookieCSRFMiddleware(),
		AuthLoginRateLimit(),
	)
	handlers.AuthHandler.InitAuthRoutes(router)
	if handlers.OIDCHandler != nil {
		handlers.OIDCHandler.InitRoutes(router)
	}
	handlers.PortalNotificationsHandler.InitRoutes(router)
	handlers.NotificationsHandler.InitRoutes(router)
	handlers.UserSearchHandler.InitRoutes(router)
	handlers.ModerationHandler.InitRoutes(router)
	handlers.ProjectsHandler.InitProjectRoutes(router)
	handlers.ProjectPageHandler.InitProjectRoutes(router)
	handlers.CoursesHandler.InitCourseRoutes(router)
	handlers.LicensingHandler.InitLicensingRoutes(router)
	handlers.PaymentsHandler.InitPaymentsRoutes(router)
	handlers.TeacherClassHandler.InitRoutes(router)
	//handlers.CohortsHandler.InitCohortRoutes(router)
	//handlers.UsersHandler.InitUsersRoutes(router)
	//handlers.RobboUnitsHandler.InitRobboUnitsRoutes(router)
	//handlers.RobboGroupHandler.InitRobboGroupRoutes(router)
	//handlers.CoursePacketHandler.InitCoursePacketRoutes(router)
	return router
}

func playgroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL", "/query")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func graphqlHandler(graphQLModule modules.GraphQLModule) gin.HandlerFunc {
	h := newGraphQLServer(generated.NewExecutableSchema(
		generated.Config{
			Resolvers: &graphQLModule.UsersResolver,
		},
	))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// newGraphQLServer mirrors handler.NewDefaultServer, except schema introspection is opt-in
// (graphql.introspection / GRAPHQL_INTROSPECTION): in production it maps the whole API.
func newGraphQLServer(es graphql.ExecutableSchema) *handler.Server {
	srv := handler.New(es)
	srv.AddTransport(transport.Websocket{KeepAlivePingInterval: 10 * time.Second})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})
	// gqlgen's default recover answers "internal server error" and logs nothing.
	srv.SetRecoverFunc(func(ctx context.Context, p interface{}) error {
		rid := ""
		if gc, ok := ctx.Value("GinContextKey").(*gin.Context); ok {
			rid = gc.GetString("request_id")
		}
		log.Printf("graphql panic rid=%s path=%v: %v\n%s", rid, graphql.GetPath(ctx), p, debug.Stack())
		return gqlerror.Errorf("internal server error")
	})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	// Bounds the work one request can ask for (aliases, nesting). The heaviest frontend
	// operation (GetUser) scores ~106; graphql.complexity_limit / GRAPHQL_COMPLEXITY_LIMIT override.
	srv.Use(extension.FixedComplexityLimit(graphQLComplexityLimit()))
	if viper.GetBool("graphql.introspection") {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New[string](100)})
	return srv
}

const defaultGraphQLComplexityLimit = 500

func graphQLComplexityLimit() int {
	if v := viper.GetInt("graphql.complexity_limit"); v > 0 {
		return v
	}
	return defaultGraphQLComplexityLimit
}

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
