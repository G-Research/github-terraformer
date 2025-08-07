package github

type AppsList struct {
	Apps []App `yaml:"apps,omitempty"`
}

type App struct {
	AppId    int64  `yaml:"app_id"`
	AppSlug  string `yaml:"app_slug"`
	AppOwner string `yaml:"app_owner"`
}
