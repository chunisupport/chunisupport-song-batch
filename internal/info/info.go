package info

const (
	// Name はアプリケーション名です。
	Name = "chunisupport-song-batch"
	// Version はアプリケーションのバージョンです。
	Version = "0.0.1"
	// ConfigDir は設定ファイルのディレクトリです。
	ConfigDir = ".config/"
	// ResourceDir はリソースファイルのディレクトリです。
	ResourceDir = ".resources/"
	// MigrationDir はマイグレーションファイルのディレクトリです。
	MigrationDir = "migration/sql/"
)

const (
	// SongBulkInsertChunkSize は楽曲をバルク挿入する際の1チャンク件数です。
	// ChartBulkInsertChunkSize は譜面をバルク挿入する際の1チャンク件数です。
	// BulkInsertChunkSize is shared by song/chart/worldsend bulk insert operations.
	BulkInsertChunkSize = 500
	// SQLiteCompoundSelectLimit はSQLiteのUNION ALLで結合できるSELECT文の数の制限です。
	// SQLiteのデフォルト制限は500なので、安全のために400を使用します。
	SQLiteCompoundSelectLimit = 400
)

const envPrefix = "CHUNISUPPORT_BATCH_"

const (
	// ENV_OFFICIAL_URL は公式データソースのURLを指す環境変数名です。
	ENV_OFFICIAL_URL = envPrefix + "OFFICIAL_URL"
	// ENV_NATUA_URL はNatuaデータソースのURLを指す環境変数名です。
	ENV_NATUA_URL = envPrefix + "NATUA_URL"
	// ENV_OTOGE_DB_URL はotoge-dbデータソースのURLを指す環境変数名です。
	ENV_OTOGE_DB_URL = envPrefix + "OTOGE_DB_URL"
	// ENV_GOOGLE_CLOUD_API_KEY はGoogle Cloud APIキーを指す環境変数名です。
	ENV_GOOGLE_CLOUD_API_KEY = envPrefix + "GOOGLE_CLOUD_API_KEY"
	// ENV_GOOGLE_SHEET_ID はGoogle Sheet IDを指す環境変数名です。
	ENV_GOOGLE_SHEET_ID = envPrefix + "GOOGLE_SHEET_ID"
	// ENV_GOOGLE_SPREADSHEET_BASE_URL はGoogle Spreadsheet APIのベースURLを指す環境変数名です。
	ENV_GOOGLE_SPREADSHEET_BASE_URL = envPrefix + "GOOGLE_SPREADSHEET_BASE_URL"
	// ENV_ADDITIONAL_SONGS_SHEET_ID は追加楽曲用Google Sheet IDを指す環境変数名です。
	ENV_ADDITIONAL_SONGS_SHEET_ID = envPrefix + "ADDITIONAL_SONGS_SHEET_ID"
)
