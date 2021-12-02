package conf

type Config struct {
	Serves []Serve
}
type Serve struct {
	Path        string
	Port        int
	AllowOrigin bool
	Log         bool
}
