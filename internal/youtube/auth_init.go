package youtube

import (
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/youtube/v3"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"youtube_tracker/internal/helpers"
)

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var redirectUrl string
	if _, err := fmt.Scan(&redirectUrl); err != nil {
		return nil, fmt.Errorf("Unable to read authorization code %v", err)
	}

	code, err := getAuthCode(redirectUrl)
	if err != nil {
		return nil, fmt.Errorf("Unable to get auth code %v", err)
	}

	token, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve token from web %v", err)
	}

	return token, nil
}

func getAuthCode(urlStr string) (string, error) {
	if urlStr == "" {
		return "", fmt.Errorf("url is empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	codeValues, ok := parsedURL.Query()["code"]
	if !ok {
		return "", fmt.Errorf("no code param: %s", urlStr)
	}

	code := codeValues[0]
	if code == "" {
		return "", fmt.Errorf("no code value: %s", urlStr)
	}

	return codeValues[0], nil
}

func saveToken(file string, token *oauth2.Token) error {
	fmt.Printf("Saving credential file to: %s\n", file)
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0600)
}

func AuthInit() {
	rootPath, err := helpers.FindProjectRoot()
	if err != nil {
		log.Fatalf("failed to get absolute path to the root: %v", err)
	}
	clientSecretFilePath := filepath.Join(rootPath, "client_secret.json")

	secretsConf, err := os.ReadFile(clientSecretFilePath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(secretsConf, youtube.YoutubeReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	cacheFilePath := tokenCacheFile(rootPath)

	_, err = os.Open(cacheFilePath)
	if err == nil {
		log.Printf("auth cache file exists: %s\n", cacheFilePath)
		return
	}

	tok, err := getTokenFromWeb(config)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}

	err = saveToken(cacheFilePath, tok)
	if err != nil {
		log.Fatalf("Unable to save token to file %s: %v", cacheFilePath, err)
	}
}
