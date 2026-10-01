package server

import (
	"context"
	"log"
	"net/http"
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
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

func NewServer(lifecycle fx.Lifecycle, graphQLModule modules.GraphQLModule, handlers modules.HandlerModule) {
	lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) (err error) {
				router := SetupGinRouter(handlers)
				if viper.GetBool("graphql.playground") {
					router.GET("/", playgroundHandler())
				}
				router.POST("/query", graphqlHandler(graphQLModule))
				router.Static("/frontend", "./frontend")
				router.GET("/frontend", func(c *gin.Context) {
					c.File("./frontend/index.html")
				})

				server := &http.Server{
					Addr:    viper.GetString("server.address"),
					Handler: newCORS().Handler(router),
					// Larger write window so large .sb3 downloads complete (BYTEA payloads).
					ReadTimeout:    120 * time.Second,
					WriteTimeout:   20 * time.Minute,
					MaxHeaderBytes: 1 << 20,
				}

				log.Printf("connect to http://localhost:%s/ for GraphQL playground", viper.GetString("graphqlServer.port"))
				go func() {
					if err = server.ListenAndServe(); err != nil {
						log.Fatalf("Failed to listen and serve: %s", err)
					}
				}()
				return
			},
			OnStop: func(context.Context) error {
				return nil
			},
		})
}

func SetupGinRouter(handlers modules.HandlerModule) *gin.Engine {
	// gin.New: gin.Default already adds Logger + Recovery, which were added again below.
	router := gin.New()
	if err := applyTrustedProxies(router); err != nil {
		log.Fatalf("invalid trusted proxies (server.trustedProxies / TRUSTED_PROXIES): %v", err)
	}
	router.Use(
		gin.Logger(),
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
	srv.SetQueryCache(lru.New(1000))
	if viper.GetBool("graphql.introspection") {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.AutomaticPersistedQuery{Cache: lru.New(100)})
	return srv
}

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
