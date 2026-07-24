package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
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
				router.GET("/", playgroundHandler())
				router.POST("/query", graphqlHandler(graphQLModule))
				router.Static("/frontend", "./frontend")
				router.GET("/frontend", func(c *gin.Context) {
					c.File("./frontend/index.html")
				})

				server := &http.Server{
					Addr: viper.GetString("server.address"),
					Handler: cors.New(
						cors.Options{
							AllowOriginFunc:  corsOriginAllowed,
							AllowCredentials: true,
							AllowedMethods: []string{
								http.MethodGet,
								http.MethodPost,
								http.MethodPut,
								http.MethodDelete,
								http.MethodOptions,
								http.MethodOptions,
							},
							AllowedHeaders: []string{"*"},
						},
					).Handler(router),
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
	router := gin.Default()
	router.Use(
		gin.Recovery(),
		gin.Logger(),
		GinContextToContextMiddleware(),
		TokenAuthMiddleware(),
	)
	handlers.AuthHandler.InitAuthRoutes(router)
	if handlers.OIDCHandler != nil {
		handlers.OIDCHandler.InitRoutes(router)
	}
	handlers.PortalNotificationsHandler.InitRoutes(router)
	handlers.NotificationsHandler.InitRoutes(router)
	handlers.UserSearchHandler.InitRoutes(router)
	handlers.ProjectsHandler.InitProjectRoutes(router)
	handlers.ProjectPageHandler.InitProjectRoutes(router)
	handlers.CoursesHandler.InitCourseRoutes(router)
	handlers.LicensingHandler.InitLicensingRoutes(router)
	handlers.PaymentsHandler.InitPaymentsRoutes(router)
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
	h := handler.NewDefaultServer(
		generated.NewExecutableSchema(
			generated.Config{
				Resolvers: &graphQLModule.UsersResolver,
			},
		))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func GinContextToContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), "GinContextKey", c)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
