# ChuniSupport Song Batch

## プロジェクト概要
ChuniSupport Song Batch は、アーケードゲーム「チュウニズム」の譜面データを定期的に取得し、MySQL データベースへ統合するための Go 製バッチアプリケーションです。複数の外部データソースから JSON をダウンロードし、インポートした内容を SQLite ワークスペースで統合したあと MySQL に同期します。アプリケーションのエントリーポイントは `main.go` に実装されています。

## 主な処理フロー
1. **ロック** – 全起動経路で MySQL アドバイザリロック `chunisupport:song-batch` を取得します。通常 cron と `--fill-missing-release-date` は競合時にスキップ（終了コード 0）、`--major-update` と `--skip-download` はエラー終了します。
2. **データソース解決** – 通常実行はサポート対象を解決します。`--major-update` は official と additional_songs だけを対象にします。
3. **データダウンロード** – 実行専用の一時ディレクトリへ取得します。成功分だけをインポートし、終了後に一時ディレクトリを削除します。st1027 / otoge-db は取得失敗時に `.datasources/` の last-known-good を使います。
4. **インポート** – データソースごとのインポーターが JSON を読み取り、共通 DTO に変換します。
5. **ワークスペース統合** – `service.ConsolidationService` が SQLite ワークスペースを構築し、全ソースのデータを統合します。
6. **MySQL 同期** – トランザクション内で最終テーブルに upsert します。必須データソースの今回取得・解析に失敗した場合は同期しません。

## リポジトリ構成
- `main.go`: フラグ解析、DB 接続、ロック、ユースケースの起動
- `internal/usecase`: 取得・必須判定・インポート・統合の実行
- `internal/config`: 環境変数・フラグの読み込み
- `internal/datasource`: データソース定義とレジストリ
- `internal/importer`: JSON 取り込みと DTO 定義
- `internal/infra`: ダウンローダー、DB 接続、アドバイザリロック、リポジトリ実装
- `internal/service`: データ統合とトランザクション管理
- `internal/workspace`: SQLite ワークスペースと MySQL 同期処理

## 動作要件
- Go 1.27.0（`go.mod` を参照）
- MySQL 8 互換データベース（`parseTime=true` オプションで接続）
- 外部データソースにアクセス可能なネットワーク
- mainframe データソースを利用する場合は Google Cloud API キーと対象スプレッドシート ID

## セットアップ手順
### 1. リポジトリと依存関係
```bash
git clone https://github.com/example/chunisupport-song-batch.git
cd chunisupport-song-batch
go mod download
```

### 2. MySQL の初期化
以下はローカル開発で利用できるサンプル設定です。必要に応じて任意の値に読み替えてください。
```sql
CREATE DATABASE chunisupport CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
CREATE USER 'chunisupport'@'localhost' IDENTIFIED BY 'chunisupport';
GRANT ALL PRIVILEGES ON chunisupport.* TO 'chunisupport'@'localhost';
FLUSH PRIVILEGES;
```
> **注意:** 本リポジトリには MySQL のテーブル定義が含まれていません。必要なスキーマはチーム内で管理している情報を適用してください。

### 3. 環境変数の設定
`.env.example` をコピーして `.env` を作成するか、シェル環境に以下の環境変数を設定します。mainframe 用の値が不要な場合、省略可能です。

```bash
cp .env.example .env
```

`--major-update` を利用する場合は、`CHUNISUPPORT_BATCH_OFFICIAL_URL` と `CHUNISUPPORT_BATCH_ADDITIONAL_SONGS_SHEET_ID`（および Google Sheets 関連の環境変数）が必須です。

| 変数名 | 用途 |
| --- | --- |
| `APP_ENV` | ログレベル判定に利用（`production` のとき Info、それ以外は Debug） |
| `PW_PEPPER` | 起動時に必須のペッパー値。現在の `display_id` 生成では使用していません |
| `DB_NAME` | MySQL データベース名 |
| `DB_HOST` | MySQL ホスト名 |
| `DB_PORT` | MySQL ポート番号 |
| `DB_USER` | MySQL ユーザー |
| `DB_PASS` | MySQL パスワード |
| `CHUNISUPPORT_BATCH_OFFICIAL_URL` | 公式データソースのダウンロード URL |
| `CHUNISUPPORT_BATCH_ST1027_URL` | st1027 データソースのダウンロード URL |
| `CHUNISUPPORT_BATCH_OTOGE_DB_URL` | otoge-db データソースのダウンロード URL（リリース日、WORLD'S END の BPM・ノーツ数・譜面製作者補完用） |
| `CHUNISUPPORT_BATCH_GOOGLE_CLOUD_API_KEY` | mainframe データソースの Google API キー |
| `CHUNISUPPORT_BATCH_GOOGLE_SHEET_ID` | mainframe データソースのスプレッドシート ID |
| `CHUNISUPPORT_BATCH_ADDITIONAL_SONGS_SHEET_ID` | additional_songs データソースのスプレッドシート ID |
| `CHUNISUPPORT_BATCH_GOOGLE_SPREADSHEET_BASE_URL` | Google Sheets API のベース URL |

mainframe のデータソースでは API キーとシート ID をもとに Google Sheets API を利用します。
### 4. データソース JSON の扱い
- 通常のダウンロード実行は `.datasources/` を入力にしません。取得先は実行ごとの一時ディレクトリです。
- 同期に成功した取得ファイルは last-known-good として `.datasources/<type>.json` へ保存します。
- `--skip-download` を指定したときだけ `.datasources/` の既存ファイルを入力にします。社内で共有されているサンプル JSON がある場合は `.datasources/<type>.json` として配置してください。
- 通常実行の必須ファイルは `official.json`、`additional_songs.json`、`mainframe.json` です。`--major-update` では `official.json` と `additional_songs.json` だけが必須です。

## 実行方法
```bash
go run . --skip-download=false
```
主なフラグは次のとおりです。

| フラグ | 説明 |
| --- | --- |
| `--skip-download` | true の場合、ダウンロードをスキップして `.datasources/` の既存 JSON を使用します |
| `--major-update` | 大型アップデート用のモード。公式データと追加楽曲のみを解決・取得し、定数更新ルールを適用します |
| `--fill-missing-release-date` | 特定フラグ有効時、いずれのデータソースからも日付が補完されずMySQLに楽曲自体が存在しない（brand new）場合に実行日（JST）をreleased_atへ補完します。otoge-db等で日付が得られない場合の最終フォールバック用 |

## `display_id` の生成
楽曲の `display_id` は、`crypto/rand` で生成した 8 バイトの乱数を16進文字列に変換した16文字のIDです。楽曲名、アーティスト名、公式ID、`PW_PEPPER` などの入力値から決定的に生成しているものではありません。

既存楽曲を MySQL に同期する際は、既存の `display_id` が空でない限り既存値を維持します。新規楽曲を別アプリから作成する場合も、同じDBに対しては一意制約の衝突を考慮して保存してください。

## テスト
ユニットテストは次のコマンドで実行できます。
```bash
go test ./...
```

## トラブルシューティング
- **データソース解決に失敗する**: 必須データソース（通常実行は official / additional_songs / mainframe、大型更新は official / additional_songs）の環境変数が未設定の可能性があります。
- **MySQL 接続に失敗する**: 接続情報（ホスト、ポート、ユーザー、パスワード）と MySQL が起動しているかを確認してください。
- **必須データソースのダウンロードが失敗する**: 前回の `.datasources/` は使いません。429/502/503/504 は数回再試行します。mainframe と additional_songs は同じ API キーを直列に呼びます。それでも失敗する場合は API キー・シート ID・URL を確認してください。
- **別プロセスが実行中**: 通常 cron は終了コード 0 でスキップします。`--major-update` や `--skip-download` はエラー終了します。

ライセンスに関する情報は `LICENSE` を参照してください。
