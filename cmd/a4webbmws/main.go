package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	. "github.com/arran4/goa4web-bookmarks"
	"github.com/arran4/gorillamuxlogic"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	clientID       string
	clientSecret   string
	externalUrl    string
	redirectUrl    string
	oauth2AuthURL  string
	oauth2TokenURL string
	version        = "dev"
	commit         = "none"
	date           = "unknown"
)

func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	SessionName = "a4webbookmarks"
	SessionStore = sessions.NewCookieStore([]byte("random-key")) // TODO random key
}

func main() {
	envPath := os.Getenv("GOBM_ENV_FILE")
	if envPath == "" {
		envPath = "/etc/goa4web-bookmarks/goa4web-bookmarks.env"
	}
	if err := LoadEnvFile(envPath); err != nil {
		log.Printf("unable to load env file %s: %v", envPath, err)
	}

	clientID = os.Getenv("OAUTH2_CLIENT_ID")
	clientSecret = os.Getenv("OAUTH2_SECRET")

	cfg := Config{
		Oauth2ClientID:  os.Getenv("OAUTH2_CLIENT_ID"),
		Oauth2Secret:    os.Getenv("OAUTH2_SECRET"),
		Oauth2AuthURL:   os.Getenv("OAUTH2_AUTH_URL"),
		Oauth2TokenURL:  os.Getenv("OAUTH2_TOKEN_URL"),
		ExternalURL:     os.Getenv("EXTERNAL_URL"),
		CssColumns:      os.Getenv("GBM_CSS_COLUMNS") != "",
		Namespace:       os.Getenv("GBM_NAMESPACE"),
		Title:           os.Getenv("GBM_TITLE"),
		FaviconCacheDir: os.Getenv("FAVICON_CACHE_DIR"),
		NoFooter:        os.Getenv("GBM_NO_FOOTER") != "",
		SessionKey:      os.Getenv("SESSION_KEY"),
	}
	if v := os.Getenv("FAVICON_CACHE_SIZE"); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.FaviconCacheSize = i
		}
	}

	configPath := DefaultConfigPath()
	var cfgFlag string
	var versionFlag bool
	flag.StringVar(&cfgFlag, "config", configPath, "path to config file")
	flag.BoolVar(&versionFlag, "version", false, "show version")
	flag.Parse()
	if versionFlag {
		fmt.Printf("a4webbmws %s commit %s built %s\n", version, commit, date)
		return
	}
	if cfgFlag != "" {
		configPath = cfgFlag
	}
	cfgSpecified := cfgFlag != "" || os.Getenv("GOBM_CONFIG_FILE") != ""
	if fileCfg, found, err := LoadConfigFile(configPath); err == nil {
		if found {
			MergeConfig(&cfg, fileCfg)
		}
	} else {
		if os.IsNotExist(err) && !cfgSpecified {
			log.Printf("config file %s not found", configPath)
		} else {
			log.Fatalf("unable to load config file %s: %v", configPath, err)
		}
	}

	UseCssColumns = cfg.CssColumns
	Namespace = cfg.Namespace
	SiteTitle = cfg.Title
	NoFooter = cfg.NoFooter
	Oauth2ClientID = cfg.Oauth2ClientID
	Oauth2ClientSecret = cfg.Oauth2Secret
	oauth2AuthURL = cfg.Oauth2AuthURL
	oauth2TokenURL = cfg.Oauth2TokenURL
	if Oauth2ClientID != "" {
		clientID = Oauth2ClientID
	}
	if Oauth2ClientSecret != "" {
		clientSecret = Oauth2ClientSecret
	}
	if cfg.FaviconCacheDir != "" {
		FaviconCacheDir = cfg.FaviconCacheDir
	}
	if cfg.FaviconCacheSize != 0 {
		FaviconCacheSize = cfg.FaviconCacheSize
	} else {
		FaviconCacheSize = DefaultFaviconCacheSize
	}

	externalUrl = strings.TrimRight(cfg.ExternalURL, "/")
	redirectUrl = fmt.Sprintf("%s/oauth2Callback", externalUrl)
	OauthRedirectURL = redirectUrl

	endpoint := oauth2.Endpoint{AuthURL: cfg.Oauth2AuthURL, TokenURL: cfg.Oauth2TokenURL}
	if endpoint.AuthURL == "" && endpoint.TokenURL == "" {
		endpoint = endpoints.Google
	}
	Oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectUrl,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     endpoint,
	}

	r := mux.NewRouter()

	r.Use(DBAdderMiddleware)
	r.Use(UserAdderMiddleware)
	r.Use(CoreAdderMiddleware)

	r.HandleFunc("/main.css", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(GetMainCSSData())
	}).Methods("GET")
	r.HandleFunc("/favicon.ico", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write(GetFavicon())
	}).Methods("GET")

	// News
	r.Handle("/", http.HandlerFunc(runTemplate("indexPage.gohtml"))).Methods("GET")
	r.Handle("/", http.HandlerFunc(TaskDoneAutoRefreshPage)).Methods("POST")

	bmr := r.PathPrefix("/bookmarks").Subrouter()
	bmr.HandleFunc("", runTemplate("bookmarksPage.gohtml")).Methods("GET")
	bmr.HandleFunc("/mine", runTemplate("bookmarksMinePage.gohtml")).Methods("GET", "POST")
	bmr.HandleFunc("/edit", runTemplate("loginPage.gohtml")).Methods("GET").MatcherFunc(gorillamuxlogic.Not(RequiresAnAccount()))
	bmr.HandleFunc("/edit", runTemplate("bookmarksEditPage.gohtml")).Methods("GET").MatcherFunc(RequiresAnAccount())
	bmr.HandleFunc("/edit", runHandlerChain(BookmarksEditSaveAction, redirectToHandler("/bookmarks/mine"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Save"))
	bmr.HandleFunc("/edit", runHandlerChain(BookmarksEditCreateAction, redirectToHandler("/bookmarks/mine"))).Methods("POST").MatcherFunc(RequiresAnAccount()).MatcherFunc(TaskMatcher("Create"))
	bmr.HandleFunc("/edit", TaskDoneAutoRefreshPage).Methods("POST")

	r.HandleFunc("/logout", runHandlerChain(UserLogoutAction, runTemplate("userLogoutPage.gohtml"))).Methods("GET")
	r.HandleFunc("/oauth2Callback", runHandlerChain(Oauth2CallbackPage, redirectToHandler("/bookmarks/mine"))).Methods("GET")
	r.HandleFunc("/proxy/favicon", FaviconProxyHandler).Methods("GET")

	http.Handle("/", r)

	if !fileExists("cert.pem") || !fileExists("key.pem") {
		CreatePEMFiles()
	}

	log.Printf("A4webbmws: %s, commit %s, built at %s", version, commit, date)
	log.Printf("Redirect URL configured to: %s", redirectUrl)
	log.Println("Server started on http://localhost:8080")
	log.Println("Server started on https://localhost:8443")

	// Create a context with a cancel function
	_, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cancellation when main exits

	// Create an HTTP server with a handler
	httpServer := &http.Server{
		Addr: ":8080",
	}

	// Create an HTTPS server with a handler
	httpsServer := &http.Server{
		Addr: ":8443",
	}

	var sigCh chan os.Signal
	// Handle ^C signal (SIGINT) to gracefully shut down the servers
	go func() {
		sigCh = make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh

		fmt.Println("Shutting down gracefully...")

		// Cancel the context to signal shutdown to both servers
		cancel()

		// Give some time for active connections to finish
		timeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("HTTP server error during shutdown: %v", err)
		}

		if err := httpsServer.Shutdown(ctx); err != nil {
			log.Printf("HTTPS server error during shutdown: %v", err)
		}

		fmt.Println("Servers gracefully shut down.")
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)
	// Start the HTTP server
	go func() {
		defer wg.Done()
		fmt.Println("HTTP server listening on :8080...")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Start the HTTPS server (TLS/SSL)
	go func() {
		defer wg.Done()
		fmt.Println("HTTPS server listening on :8443...")
		if err := httpsServer.ListenAndServeTLS("cert.pem", "key.pem"); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTPS server error: %v", err)
		}
	}()

	wg.Wait()

}

func CreatePEMFiles() {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		log.Fatalf("Failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Your Organization"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	certFile, err := os.Create("cert.pem")
	if err != nil {
		log.Fatalf("Failed to create cert.pem file: %v", err)
	}
	defer certFile.Close()
	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %v", err)
	}

	keyFile, err := os.Create("key.pem")
	if err != nil {
		log.Fatalf("Failed to create key.pem file: %v", err)
	}
	defer keyFile.Close()
	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		log.Fatalf("Failed to marshal private key: %v", err)
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %v", err)
	}
}

func runHandlerChain(chain ...any) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, each := range chain {
			switch each := each.(type) {
			case http.Handler:
				each.ServeHTTP(w, r)
			case http.HandlerFunc:
				each(w, r)
			case func(http.ResponseWriter, *http.Request):
				each(w, r)
			default:
				log.Panicf("unknown input: %s", reflect.TypeOf(each))
			}
		}
	}
}

func runTemplate(template string) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type Data struct {
			*CoreData
		}

		data := Data{
			CoreData: r.Context().Value(ContextValues("coreData")).(*CoreData),
		}

		if err := GetCompiledTemplates(NewFuncs(r)).ExecuteTemplate(w, template, data); err != nil {
			log.Printf("Template Error: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}

func redirectToHandler(toUrl string) func(http.ResponseWriter, *http.Request) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, toUrl, http.StatusTemporaryRedirect)
	})
}

func RequiresAnAccount() mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		var session *sessions.Session
		sessioni := request.Context().Value(ContextValues("session"))
		if sessioni == nil {
			var err error
			session, err = SessionStore.Get(request, SessionName)
			if err != nil {
				return false
			}
		} else {
			var ok bool
			session, ok = sessioni.(*sessions.Session)
			if !ok {
				return false
			}
		}
		userRef, _ := session.Values["UserRef"].(string)
		return userRef != ""
	}
}

func TaskMatcher(taskName string) mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		return request.PostFormValue("task") == taskName
	}
}

func NoTask() mux.MatcherFunc {
	return func(request *http.Request, match *mux.RouteMatch) bool {
		return request.PostFormValue("task") == ""
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}
