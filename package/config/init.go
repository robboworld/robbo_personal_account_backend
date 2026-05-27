package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

func Init() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./package/config")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if os.Getenv("POSTGRES_HOST") != "" {
		viper.Set("postgres.postgresDsn", "host="+os.Getenv("POSTGRES_HOST")+" port=5432 user="+os.Getenv("POSTGRES_USER")+" password="+os.Getenv("POSTGRES_PASSWORD")+" dbname="+os.Getenv("POSTGRES_DB")+" sslmode=disable")
	}
	if os.Getenv("PROJECTS_POSTGRES_DSN") != "" {
		viper.Set("projectsPostgres.postgresDsn", os.Getenv("PROJECTS_POSTGRES_DSN"))
	}
	if os.Getenv("LMS_MYSQL_DSN") != "" {
		viper.Set("lmsMysql.dsn", os.Getenv("LMS_MYSQL_DSN"))
	}
	if os.Getenv("LMS_MYSQL_WRITE_DSN") != "" {
		viper.Set("lmsMysql.writeDsn", os.Getenv("LMS_MYSQL_WRITE_DSN"))
	}
	if os.Getenv("AUTH_MODE") != "" {
		viper.Set("auth.mode", os.Getenv("AUTH_MODE"))
	}
	if os.Getenv("AUTH_LMS_PASSWORD_FALLBACK") == "true" {
		viper.Set("auth.lmsPasswordFallback", true)
	}
	if os.Getenv("LEGACY_POSTGRES_ENABLED") == "true" {
		viper.Set("legacyPostgres.enabled", true)
	} else if os.Getenv("LEGACY_POSTGRES_ENABLED") == "false" {
		viper.Set("legacyPostgres.enabled", false)
	}
	if os.Getenv("LK_SSO_WITH_LMS_ENABLED") == "true" {
		viper.Set("oidc.enabled", true)
	}
	if os.Getenv("OIDC_ISSUER") != "" {
		viper.Set("oidc.issuer", os.Getenv("OIDC_ISSUER"))
	}
	if os.Getenv("OIDC_AUTHORIZATION_ENDPOINT") != "" {
		viper.Set("oidc.authorizationEndpoint", os.Getenv("OIDC_AUTHORIZATION_ENDPOINT"))
	}
	if os.Getenv("OIDC_TOKEN_ENDPOINT") != "" {
		viper.Set("oidc.tokenEndpoint", os.Getenv("OIDC_TOKEN_ENDPOINT"))
	}
	if os.Getenv("OIDC_JWKS_URI") != "" {
		viper.Set("oidc.jwksUri", os.Getenv("OIDC_JWKS_URI"))
	}
	if os.Getenv("OIDC_CLIENT_ID") != "" {
		viper.Set("oidc.clientId", os.Getenv("OIDC_CLIENT_ID"))
	}
	if os.Getenv("OIDC_REDIRECT_URI") != "" {
		viper.Set("oidc.redirectUri", os.Getenv("OIDC_REDIRECT_URI"))
	}
	if os.Getenv("OIDC_LOGOUT_ENDPOINT") != "" {
		viper.Set("oidc.logoutEndpoint", os.Getenv("OIDC_LOGOUT_ENDPOINT"))
	}
	if os.Getenv("OIDC_POST_LOGOUT_REDIRECT_URI") != "" {
		viper.Set("oidc.postLogoutRedirectUri", os.Getenv("OIDC_POST_LOGOUT_REDIRECT_URI"))
	}
	if os.Getenv("OIDC_USERINFO_ENDPOINT") != "" {
		viper.Set("oidc.userinfoEndpoint", os.Getenv("OIDC_USERINFO_ENDPOINT"))
	}
	if os.Getenv("OIDC_FRONTEND_BASE_URL") != "" {
		viper.Set("oidc.frontendBaseUrl", os.Getenv("OIDC_FRONTEND_BASE_URL"))
	}
	if os.Getenv("PORTAL_OUTBOX_ENABLED") == "true" {
		viper.Set("portalOutbox.enabled", true)
	} else if os.Getenv("PORTAL_OUTBOX_ENABLED") == "false" {
		viper.Set("portalOutbox.enabled", false)
	}
	if os.Getenv("LMS_NOTIFICATIONS_ENABLED") == "true" {
		viper.Set("lmsNotifications.enabled", true)
	}
	if os.Getenv("LMS_NOTIFICATIONS_INGEST_TOKEN") != "" {
		viper.Set("lmsNotifications.ingestBearerToken", os.Getenv("LMS_NOTIFICATIONS_INGEST_TOKEN"))
	}

	err := viper.ReadInConfig()
	return err
}

func InitForTests() error {
	viper.SetConfigName("config-test")
	viper.SetConfigType("yml")
	viper.AddConfigPath("../../package/config")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	return err
}
