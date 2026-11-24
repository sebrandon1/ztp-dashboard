package config

type Config struct {
	KubeConfig     string
	Port           int
	LogFormat      string
	OllamaEndpoint string
	OllamaModel    string
	DBPath         string
	AllowedOrigins []string
	AuthToken      string
	AuthTokenFile  string
}
