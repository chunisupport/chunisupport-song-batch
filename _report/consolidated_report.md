# プロジェクト監査レポート

本レポートは、`chunisupport-song-batch` のコードベースに対するセキュリティ、パフォーマンス、信頼性、アーキテクチャ、保守性の観点から監査を実施した結果をまとめたものです。

## 優先度の定義

| 優先度 | 説明 |
|--------|------|
| **Critical** | サービスの停止、著しいパフォーマンス劣化、またはセキュリティリスクに直結するため、即時対応が必要。 |
| **High** | 保守性や拡張性を著しく損なっており、技術的負債として大きなリスクがある。 |
| **Medium** | 一般的なベストプラクティスからの逸脱。リファクタリング推奨。 |
| **Low** | 軽微な修正や好みの問題。 |

---

## 1. セキュリティ (Security)

### SEC-002: SSRF (Server-Side Request Forgery) のリスク (Severity: Medium)

**内容**: `internal/infra/datasource/downloader.go` の `downloadDatasource` メソッドは、設定ファイル (`.config/*.settings.json` や環境変数) から供給される URL に対して HTTP GET リクエストを行います。

**リスク**: 設定ファイルの管理が甘い場合、または環境変数を外部から注入可能な状況下において、攻撃者が内部ネットワーク（例: `http://localhost:8080/admin`, AWS メタデータ `http://169.254.169.254/`）へのアクセスを試みる踏み台として利用される恐れがあります。

**推奨対策**: 許可するプロトコル（httpsのみ）、許可するドメインのホワイトリスト化、およびプライベート IP アドレス範囲への接続ブロックを実装することを強く推奨します。

### SEC-003: 巨大ファイルによる DoS (Memory Exhaustion) (Severity: Medium)

**内容**: `internal/infra/datasource/downloader.go` の `io.ReadAll(resp.Body)` は、レスポンスサイズの上限チェックを行わずに全てのデータをメモリに読み込みます。また、`json.Compact` はメモリ上のバッファをさらに確保します。

**リスク**: 悪意のあるデータソース、あるいは設定ミスにより巨大なファイル（数GB単位）を指定された場合、アプリケーションが OOM (Out of Memory) でクラッシュし、サービス停止（DoS）につながります。

**推奨対策**: `io.LimitReader` を使用して読み込みサイズに上限を設けること。また、`io.ReadAll` ではなく `io.Copy` でファイルにストリーミング書き込みを行う実装に変更すべきです。

### SEC-005: 動的SQL構築におけるインジェクションリスク (Severity: High)

**内容**: `internal/workspace/songchart/workspace.go` において、`CASE` 文を含む複雑な UPDATE/INSERT 文を、`text/template` や文字列連結を用いて構築しています。プレースホルダ (`?`) を使用している箇所もありますが、SQL構文自体を動的に組み立てるアプローチはエスケープ漏れのリスクがあり、デバッグも困難です。

**リスク**: テンプレートへの入力値が適切にサニタイズされていない場合、SQLインジェクション攻撃の対象となる可能性があります。

**推奨対策**: 可能な限り `sqlx` の `NamedExec` や `In` 句を活用し、静的なSQL文とプレースホルダの組み合わせで処理してください。複雑な条件分岐が必要な場合は、アプリケーションロジック側でデータを整形してから単純なクエリを投げる設計を検討してください。

---

## 2. パフォーマンス (Performance)

### PERF-002: `io.ReadAll` と `json.Compact` によるメモリ圧迫 (Severity: High)

**内容**: `internal/infra/datasource/downloader.go` の `downloadDatasource` メソッドにおいて、HTTPレスポンスボディを `io.ReadAll` で全てメモリに読み込んだ後、さらに `json.Compact` で別のバッファにコピーしています。

**リスク**: データソースが巨大化（数百MB単位）した場合、元のデータ `data` と圧縮後のバッファ `minified` の両方がメモリ上に存在するため、メモリ使用量がファイルサイズの数倍に膨れ上がり、OOM (Out Of Memory) キルを誘発する恐れがあります。

**推奨対策**: `io.ReadAll` を廃止し、`os.Create` したファイルに対して `io.Copy(file, resp.Body)` を行うストリーム処理に変更してください。`json.Compact` による空白除去が必須であれば、`json.Decoder` と `json.Encoder` を組み合わせたストリーム変換を検討してください。

### PERF-003: `SELECT *` の使用による不要なカラム読み込み (Severity: Medium)

**内容**: `internal/workspace/songchart/workspace.go` においてワークスペースの取得・ダンプで `SELECT *` が使われており、不要なカラム読み込みが発生します。

**リスク**: パフォーマンスの低下だけでなく、予期せぬスキーマ変更に対する安全性の低下や、機密情報の意図しない取得につながる可能性があります。

**推奨対策**: 取得カラムを明示し、必要最低限の列のみ取得するようにしてください。

### PERF-004: MySQL全件読み込みによるメモリとI/Oコストの増大 (Severity: Medium)

**内容**: `internal/workspace/songchart/workspace.go` において MySQL 全件を読み込んで差分比較しており、テーブル規模が大きい場合にメモリとI/Oコストが増大します。

**リスク**: テーブルのレコード数が増加した場合、処理時間とメモリ使用量が線形に増加し、スケーラビリティの問題が発生します。

**推奨対策**: ワークスペース側のID集合で絞り込み、必要なレコードのみ取得する（IN句や一時テーブル/JOINを活用）。

---

## 3. 信頼性 (Reliability)

### REL-002: 部分的なダウンロード失敗の許容とデータ不整合 (Severity: Medium)

**内容**: `main.go` の `executeDataImportBatch` では、`downloadDatasources` がエラーを返しても「一部のデータソースで続行」します。しかし、`Downloader.DownloadAll` は「全てのダウンロードが失敗した場合」のみエラーを返す仕様にはなっていません（実装上は `successCount == 0 && len(downloadErrors) > 0` の判定）。

**リスク**: クリティカルなマスタデータ（例: 公式データ）のダウンロードに失敗し、補完的なデータのみ成功した場合でも処理が進んでしまう可能性があります。これにより、データベースが不完全な状態で更新されるリスクがあります。

**推奨対策**: データソースごとに「必須（Critical）」フラグを設け、必須データソースのダウンロードに失敗した場合は即座にバッチ処理を中断（Abend）させるべきです。

---

## 4. アーキテクチャ (Architecture)

### ARCH-001: OCP (開放閉鎖原則) 違反による拡張性の欠如 (Severity: High)

**内容**: `main.go` (`importDataByDatasources`), `internal/importer` パッケージにおいて、新しいデータソースを追加する際、`main.go` 内の `switch` 文を手動で修正し、具体的な型（`*importer.OfficialData` 等）へのキャストを追加する必要があります。

**リスク**: メインロジックが具体的な実装詳細に依存していることを意味し、拡張のたびに既存コードの破壊リスクが生じます。

**推奨対策**: `Importer` インターフェース、またはそこから生成されるデータモデルを統一（例: 全てのインポーターが共通の `[]SongDTO` を返す）し、`main.go` はポリモーフィズムを用いて透過的に処理できるようにしてください。

---

## 5. 保守性 (Maintainability)

### MAINT-002: 命名規則の不統一とマジックストリング (Severity: Low)

**内容**: `internal/info/info.go`, `main.go` 他において、定数名において `Name` (CamelCase) と `ENV_OFFICIAL_URL` (Screaming Snake Case) が混在しています。また、ファイルパス（`.datasources`）や難易度IDなどがコード中にハードコードされています。

**リスク**: コードの可読性が低下し、設定値の変更時に修正漏れが発生しやすくなります。

**推奨対策**: Go の命名規約（定数は CamelCase）に統一してください。設定値は設定ファイルや環境変数から読み込むか、一箇所の定数定義に集約してください。

### MAINT-003: `panic` を前提としたエラーハンドリング (Severity: Low)

**内容**: `internal/datasource/registry/registry.go`, `internal/workspace/songchart/workspace.go` において、`panic` を前提としたエラーハンドリング（`Register` の重複登録、`template.Must`）があり、実行時停止のリスクがあります。

**リスク**: 予期せぬ入力や設定ミスによりアプリケーションが突然停止し、運用に支障をきたす可能性があります。

**推奨対策**: `error` を返す設計に変更し、呼び出し側で適切に処理できるようにしてください。

---

## 指摘事項サマリー

| ID | カテゴリ | 優先度 | 概要 |
|----|----------|--------|------|
| SEC-002 | セキュリティ | Medium | SSRF のリスク |
| SEC-003 | セキュリティ | Medium | 巨大ファイルによる DoS |
| SEC-005 | セキュリティ | High | 動的SQL構築におけるインジェクションリスク |
| PERF-002 | パフォーマンス | High | `io.ReadAll` と `json.Compact` によるメモリ圧迫 |
| PERF-003 | パフォーマンス | Medium | `SELECT *` の使用 |
| PERF-004 | パフォーマンス | Medium | MySQL全件読み込み |
| REL-001 | 信頼性 | Medium | `mainframe` パーサーのサイレントなデータ欠損 |
| REL-002 | 信頼性 | Medium | 部分的なダウンロード失敗の許容 |
| ARCH-001 | アーキテクチャ | High | OCP違反による拡張性の欠如 |
| MAINT-002 | 保守性 | Low | 命名規則の不統一 |
| MAINT-003 | 保守性 | Low | `panic` を前提としたエラーハンドリング |
