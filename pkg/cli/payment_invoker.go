package cli

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/oauth2"
)

func StartServer(config Config) {
	serverAddress := fmt.Sprintf("localhost:%v", config.EmbeddedServerConfig.Port)

	provider, err := oidc.NewProvider(context.TODO(), config.KeycloakConfig.KeycloakURL+"/realms/"+config.KeycloakConfig.Realm)
	if err != nil {
		log.Fatalf("Unable to connect to Keycloak: %v\n", err)
		CloseApp.Done()
	}

	oauth2Config := oauth2.Config{
		ClientID:    config.KeycloakConfig.ClientID,
		RedirectURL: config.EmbeddedServerConfig.GetCallbackURL(),
		Endpoint:    provider.Endpoint(),
		Scopes:      []string{oidc.ScopeOpenID, "profile", "email"},
	}

	http.HandleFunc("/sso-callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		fmt.Printf("Convert code=%v to token? [y/n]\n", code)
		reader := bufio.NewReader(os.Stdin)
		shouldConvert, _ := reader.ReadString('\n')
		if strings.Contains(shouldConvert, "y") {
			oauth2Token, err := oauth2Config.Exchange(context.TODO(), code)
			if err != nil {
				fmt.Errorf("couldn't obtain a token %v", err)
				return
			}

			t, _ := jwt.Parse(oauth2Token.AccessToken, nil)
			m := t.Claims.(jwt.MapClaims)
			fmt.Printf("Obtaining payment information for user %v\n", m["preferred_username"])

		} else {
			fmt.Printf("Just in case you wanted to replay this:\n")
			fmt.Printf("curl \"http://localhost:%v/%v?code=%v\"\n", config.EmbeddedServerConfig.Port, config.EmbeddedServerConfig.CallbackPath, code)
		}
	})

	go func() {
		log.Print("Booting up the server")
		if err := http.ListenAndServe(serverAddress, nil); err != nil {
			log.Fatalf("Unable to start server: %v\n", err)
			CloseApp.Done()
		}
	}()
}
