package registry

import (
	"fmt"
	"os"
	"strings"

	"github.com/chunisupport/chunisupport-song-batch/internal/info"
)

// init registers available datasource providers.
func init() {
	Register("official", officialProvider)
	Register("additional_songs", additionalSongsProvider)
	Register("st1027", st1027Provider)
	Register("mainframe", mainframeProvider)
	Register("otoge_db", otogeDbProvider)
}

func officialProvider() (Definition, error) {
	url, err := requireNonEmptyEnv(info.ENV_OFFICIAL_URL)
	if err != nil {
		return Definition{}, err
	}
	return Definition{
		Type: "official",
		URL:  url,
	}, nil
}

func st1027Provider() (Definition, error) {
	url, err := requireNonEmptyEnv(info.ENV_ST1027_URL)
	if err != nil {
		return Definition{}, err
	}
	return Definition{
		Type: "st1027",
		URL:  url,
	}, nil
}

func mainframeProvider() (Definition, error) {
	// mainframeはGoogleスプレッドシートを使用するため、
	// APIキーとシートIDを確認
	apiKey, err := requireNonEmptyEnv(info.ENV_GOOGLE_CLOUD_API_KEY)
	if err != nil {
		return Definition{}, fmt.Errorf("mainframe requires Google Cloud API key: %w", err)
	}
	sheetID, err := requireNonEmptyEnv(info.ENV_GOOGLE_SHEET_ID)
	if err != nil {
		return Definition{}, fmt.Errorf("mainframe requires Google Sheet ID: %w", err)
	}
	baseURL, err := requireNonEmptyEnv(info.ENV_GOOGLE_SPREADSHEET_BASE_URL)
	if err != nil {
		return Definition{}, fmt.Errorf("mainframe requires Google Spreadsheet base URL: %w", err)
	}

	// URLフィールドは使用しないが、識別用にシートIDを設定
	return Definition{
		Type: "mainframe",
		URL:  fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", sheetID),
		Params: map[string]string{
			"apiKey":  apiKey,
			"sheetID": sheetID,
			"baseURL": baseURL,
		},
	}, nil
}

func otogeDbProvider() (Definition, error) {
	// otoge-dbデータソースはURLのみで認証不要
	url, err := requireNonEmptyEnv(info.ENV_OTOGE_DB_URL)
	if err != nil {
		return Definition{}, err
	}

	return Definition{
		Type: "otoge_db",
		URL:  url,
	}, nil
}

func additionalSongsProvider() (Definition, error) {
	// additional_songsはGoogleスプレッドシートを使用するため、
	// APIキーとシートIDを確認
	apiKey, err := requireNonEmptyEnv(info.ENV_GOOGLE_CLOUD_API_KEY)
	if err != nil {
		return Definition{}, fmt.Errorf("additional_songs requires Google Cloud API key: %w", err)
	}
	sheetID, err := requireNonEmptyEnv(info.ENV_ADDITIONAL_SONGS_SHEET_ID)
	if err != nil {
		return Definition{}, fmt.Errorf("additional_songs requires Google Sheet ID: %w", err)
	}
	baseURL, err := requireNonEmptyEnv(info.ENV_GOOGLE_SPREADSHEET_BASE_URL)
	if err != nil {
		return Definition{}, fmt.Errorf("additional_songs requires Google Spreadsheet base URL: %w", err)
	}

	return Definition{
		Type: "additional_songs",
		URL:  fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", sheetID),
		Params: map[string]string{
			"apiKey":  apiKey,
			"sheetID": sheetID,
			"baseURL": baseURL,
		},
	}, nil
}

func requireNonEmptyEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("%s environment variable not set", key)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s environment variable must not be empty", key)
	}
	return value, nil
}
