# ChuniSupport Song Batch

## プロジェクト概要
ChuniSupport Song Batch は、アーケードゲーム「チュウニズム」の譜面データを定期的に取得し、MySQL データベースへ統合するための Go 製バッチアプリケーションです。複数の外部データソースから JSON をダウンロードし、インポートした内容を SQLite ワークスペースで統合したあと MySQL に同期します。アプリケーションのエントリーポイントは `main.go` に実装されています。

## 主な処理フロー
1. **データソース解決** – サポート対象の全データソースを環境変数から解決します。
2. **データダウンロード** – `internal/infra/datasource.Downloader` が `.datasources/` 配下に JSON ファイルを保存します。mainframe は Google Sheets から専用ロジックで取得します。
3. **インポート** – データソースごとのインポーターが JSON を読み取り、共通 DTO に変換します。
4. **ワークスペース統合** – `service.ConsolidationService` が SQLite ワークスペースを構築し、全ソースのデータを統合します。
5. **MySQL 同期** – トランザクション内で最終テーブルに upsert し、必要に応じてワークスペースダンプを出力します。

## リポジトリ構成
- `main.go`: バッチアプリケーションのエントリーポイント
- `internal/config`: 環境変数・フラグの読み込み
- `internal/datasource`: データソース定義とレジストリ
- `internal/importer`: JSON 取り込みと DTO 定義
- `internal/infra`: ダウンローダー、DB 接続、リポジトリ実装
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
- 初回実行時に `.datasources/` ディレクトリが生成され、各種 JSON が保存されます。
- `--skip-download` フラグを指定すると既存ファイルを利用します。社内で共有されているサンプル JSON がある場合は `.datasources/<type>.json` として配置してください。

## 実行方法
```bash
go run . --skip-download=false
```
主なフラグは次のとおりです。

| フラグ | 説明 |
| --- | --- |
| `--skip-download` | true の場合、ダウンロードをスキップして既存 JSON を使用します |
| `--major-update` | 大型アップデート用のモード。公式データと追加楽曲のみを使用し、定数更新ルールを適用します |
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
- **データソース解決に失敗する**: いずれかのデータソースに必要な環境変数が未設定の可能性があります。ログを確認し、該当 URL や API キーを設定してください。
- **MySQL 接続に失敗する**: 接続情報（ホスト、ポート、ユーザー、パスワード）と MySQL が起動しているかを確認してください。
- **mainframe ダウンロードが失敗する**: `.datasources/mainframe.json` を削除し、Google API キーとシート ID が正しいかを確認したうえで再実行してください。

ライセンスに関する情報は `LICENSE` を参照してください。
