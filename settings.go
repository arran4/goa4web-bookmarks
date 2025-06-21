package a4webbm

import "time"

var (
	UseCssColumns bool
	Namespace     string
	SiteTitle     string
	NoFooter      bool

	Oauth2ClientID     string
	Oauth2ClientSecret string
	Oauth2AuthURL      string
	Oauth2TokenURL     string
	OauthRedirectURL   string
	FaviconCacheDir    string
	FaviconCacheSize   int64
)

const (
	DefaultFaviconCacheSize   int64         = 20 * 1024 * 1024 // 20MB
	DefaultFaviconCacheMaxAge time.Duration = 24 * time.Hour
)
