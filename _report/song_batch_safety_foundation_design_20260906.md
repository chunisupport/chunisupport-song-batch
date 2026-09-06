# song-batch 安全性基盤 設計・実装計画書

## 0. 文書の位置付け

本書は `chunisupport-song-batch` の実行安全性を先に確立するための設計・実装計画書である。管理画面からのジョブ実行、DBジョブキュー、dispatcher は対象外とする。

本書は当初 `chunisupport-api/_report` に置く。実装リポジトリは `chunisupport-song-batch` であり、実装開始時に同リポジトリの `_report/` または `docs/` へ移す。

関連文書:

- 後続の管理画面・ジョブ基盤: `song_batch_admin_operation_design_20260906.md`
- 現行CLIとデータソース: `chunisupport-song-batch/README.md`

本書を完了するまで、管理画面から大型更新や Mainframe 更新を実行できるようにしてはならない。古い取得ファイルの再利用と cron 多重実行の危険を、より簡単に発生させられる状態になるためである。

---

## 1. 背景と課題

### 1.1 古い取得ファイルの再利用

現行の song-batch は、データソースのダウンロードが一部失敗しても処理を続行する。出力先は固定の `.datasources/` である。今回の取得に失敗したデータソースについて、前回実行時の JSON が残っていると、その古いファイルを読み込む。

実際の処理は次のとおりである。

1. `Downloader.DownloadAll` はデータソース単位で失敗しても全体としては続行する。全件失敗のときだけ error を返す。
2. `main.go` の `executeDataImportBatch` は、ダウンロードが error でも「利用可能なデータソースで継続する」としてインポートへ進む。全件失敗でも同じ経路を通る。
3. `importDataByDatasources` は解決済みデータソースすべてについて `.datasources/{type}.json` を読む。今回書けなかったファイルでも、前回ファイルがあればインポートが成功する。

大型アップデート (`--major-update`) で古い公式データを使うと、意図しない譜面定数の不明化や更新漏れが起きる。通常実行でも、一時障害のたびに古い定数・古い楽曲情報が混入する。

### 1.2 大型更新でも不要データソースを解決・取得している

`--major-update` は統合時に official と additional_songs 以外をスキップする。しかし解決とダウンロードは全データソースを対象にする。

そのため次が起きる。

- mainframe / st1027 / otoge-db の一時障害が、本来不要な取得失敗としてログに残る
- 対象外データソースの前回 JSON がインポート対象になり得る
- 必須データソースの今回取得失敗を、前回ファイルが隠す

必須データの「今回取得できたこと」を成功条件にできない。

### 1.3 多重起動の防止がない

通常 cron、木曜の `--fill-missing-release-date`、手動の `--major-update` は同じバイナリを別プロセスとして起動できる。MySQL 同期はトランザクション内だが、プロセス同士の実行枠は共有していない。

同時実行すると次が起きる。

- 同じ譜面への競合更新
- 長時間ロックやデッドロック
- 後続の管理画面ジョブを足したときに、排他を後付けせざるを得ない

他バッチ（静的データ出力、プレイヤーデータ再計算、stat-batch）はすでに MySQL アドバイザリロックを使っている。song-batch だけが未導入である。

---

## 2. 目的

- 通常のダウンロード実行が、前回の `.datasources/` を暗黙に再利用しないこと
- 今回取得に成功したデータソースだけをインポート・統合すること
- 大型更新は最初から official と additional_songs だけを解決・取得すること
- 必須データソースの今回取得または解析に失敗したら、MySQL の楽曲・譜面を変更しないこと
- すべての song-batch 起動経路が、同時に更新処理を行わないこと
- 既存 CLI フラグと既存 cron の意味を維持すること
- 後続の dispatcher が同じ取得・統合処理を呼べること

---

## 3. 対象範囲

### 3.1 対象

- 実行専用一時ディレクトリ
- データソース単位の取得結果
- モード別の必須データソース判定
- 大型更新時の解決対象の縮小
- 全モード共通の MySQL アドバイザリロック
- `main.go` から再利用可能な実行ユースケースへの分離
- 上記を保証するテスト
- README の `.datasources/` 説明の更新

### 3.2 対象外

- 管理画面、管理者 API、ジョブテーブル
- `--run-pending-job` と 1 分 dispatcher
- Mainframe スプレッドシート ID の DB 管理
- Mainframe 解析の重複キー変更（タイトル+難易度）
- 環境変数以外からの設定解決
- 実行中ジョブの強制終了
- `--skip-download` 以外のローカルキャッシュ戦略
- プレイヤーデータ再計算、静的データ出力、image-batch、stat-batch との排他

対象外は後続の管理画面設計で扱う。本書の完了後に着手する。

---

## 4. 現行実装の要点

実装時の変更箇所を誤らないための現状である。

| 箇所 | 現状 |
|---|---|
| `internal/infra/datasource/downloader.go` | 固定 `outputDir` へ保存。一部失敗は error にしない。データソース単位の結果を返さない |
| `main.go` `downloadDatasources` | 出力先は常に `.datasources` |
| `main.go` `executeDataImportBatch` | ダウンロード error でもインポートへ進む |
| `main.go` `importDataByDatasources` | 解決済み全件について `.datasources/{type}.json` を読む |
| `main.go` `resolveAllDatasources` | サポート対象をすべて解決する |
| `main.go` `validateMajorUpdateDatasources` | 解決済み一覧に official と additional_songs が含まれることだけを見る |
| `main.go` `validateMajorUpdateSources` | インポート結果の nil チェックだけ。今回取得かどうかは見ない |
| `internal/service/consolidation_service.go` | 大型更新時、統合対象から extra ソースを除外する。除外は取得後 |
| `internal/config/flags.go` | `--skip-download` / `--major-update` / `--fill-missing-release-date` |
| ロック | 未実装 |

`--skip-download` は「既存 JSON を明示利用する」ためのフラグとして残す。この経路だけが `.datasources/` を使ってよい。

---

## 5. 採用方式

### 5.1 実行専用ディレクトリ

ダウンロードする実行では、実行ごとに一意な一時ディレクトリを作る。

```text
os.MkdirTemp("", "chunisupport-song-batch-*")
```

規則:

- 今回成功したファイルだけをそのディレクトリへ書く
- Importer はそのディレクトリ内のファイルだけを読む
- 処理終了後にディレクトリを削除する。成功・失敗を問わない
- プロセスが強制終了して残っても、次回実行の入力には使わない
- 固定 `.datasources/` へ書き込まない

`--skip-download` を明示したときだけ `.datasources/` を読む。この場合はダウンロードせず、既存ファイルを入力とする。通常実行が前回ファイルを暗黙再利用する経路を残さない。

一時ディレクトリのパスはログに出してよい。中身の JSON 本文は出さない。

### 5.2 データソース単位の取得結果

`DownloadAll` は全体の error だけでなく、データソースごとの結果を返す。

保持する値:

- データソース種別
- 成功 / 失敗
- 保存先パス（成功時）
- 取得日時（成功時）
- バイト数（成功時）
- 失敗理由（失敗時。外部レスポンス本文は含めない）

Importer へ渡す対象は、今回成功したデータソースだけとする。解決には成功したが取得に失敗したデータソースは、存在しないものとして扱う。前回ファイルを探さない。

全件失敗の扱いはモードの必須条件に従う。通常モードで 1 件も成功しなければ MySQL を変更せず終了する。

### 5.3 モード別の必須条件

| モード | 解決・取得対象 | 必須データソース | 失敗時 |
|---|---|---|---|
| 通常 | 解決できたすべてのデータソース | なし | 成功したデータソースだけを統合する。0 件なら DB 変更なし |
| 大型更新 | official、additional_songs のみ | official、additional_songs | どちらかの解決・取得・解析失敗で全体失敗。DB 変更なし |
| `--skip-download` | CLI が指定する既存ファイル | そのモードの必須条件を同じファイル集合へ適用 | 必須ファイルが無い、または解析不能なら DB 変更なし |

大型更新では、対象外データソースを解決・ダウンロードしてから除外しない。最初から official と additional_songs だけを対象にする。

`--fill-missing-release-date` は取得対象を変えない。通常モードと同じデータソース解決に、リリース日補完オプションを付けるフラグとして扱う。

`--skip-download` と `--major-update` を同時指定した場合、`.datasources/official.json` と `.datasources/additional_songs.json` の両方が必要である。無い、または解析不能なら DB を変更しない。

### 5.4 全 song-batch 共通ロック

通常実行、`--major-update`、`--fill-missing-release-date` のすべてで MySQL アドバイザリロックを使う。後続の dispatcher も同じロック名を使う。

```text
chunisupport:song-batch
```

定数は `internal/info` に置く。モード別ロックには分けない。

ロックは専用 DB 接続で取得し、プロセス終了まで接続を保持する。`GET_LOCK(name, 0)` で待機しない。既存の `chunisupport-api` / `chunisupport-stat-batch` と同じ接続単位ロックである。

取得できない場合の終了動作:

| 起動経路 | 動作 | 終了コード |
|---|---|---:|
| フラグなし（通常 cron） | スキップ理由を記録して終了 | 0 |
| `--fill-missing-release-date`（木曜 cron） | スキップ理由を記録して終了 | 0 |
| `--major-update`（明示実行） | 実行されなかったことが分かるエラー | 非 0 |
| `--skip-download` を含む明示実行 | 実行されなかったことが分かるエラー | 非 0 |

通常 cron の重複回避と、後続の管理画面ジョブ排他に同じロックを使う。プレイヤーデータ再計算など他バッチのロック名とは共有しない。

ロック取得はデータソース解決より前でよい。取得できないなら外部通信を始めない。

プロセスが異常終了すると、接続切断により MySQL がロックを解放する。明示的な `RELEASE_LOCK` は正常終了時に行う。

### 5.5 CLI とユースケースの分離

`main.go` の取得・検証・統合を、CLI フラグ解析から独立したユースケースへ移す。後続の dispatcher が同じユースケースを呼ぶため、処理を複製しない。

概念上の入力:

```go
type RunMode string

const (
    RunModeNormal      RunMode = "NORMAL"
    RunModeMajorUpdate RunMode = "MAJOR_UPDATE"
)

type RunRequest struct {
    Mode                   RunMode
    SkipDownload           bool
    FillMissingReleaseDate bool
}
```

`MAINFRAME_SHEET_UPDATE` と `JobID` は後続設計で足す。本書のユースケースは、それらが来ても取得・ロック・必須判定の骨格を変えずに拡張できる形にする。

`main` の責務は次に限る。

1. ログ初期化
2. フラグ解析
3. 設定と DB 接続
4. ロック取得と解放
5. `RunRequest` を組み立ててユースケースを呼ぶ
6. 終了コードの決定

フラグからモードへの変換:

- `--major-update` が真なら `RunModeMajorUpdate`
- それ以外は `RunModeNormal`
- `--fill-missing-release-date` は `FillMissingReleaseDate` へ渡す
- `--skip-download` は `SkipDownload` へ渡す

ユースケースは `flag` パッケージも環境変数名も知らない。Downloader / Importer / Consolidation のインターフェースにだけ依存する。

---

## 6. 処理フロー

### 6.1 通常実行（ダウンロードあり）

```text
起動
  ↓
共通ロックを試行
  ├─ 失敗: スキップして終了コード 0
  └─ 成功
       ↓
     解決できたデータソースを列挙
       ↓
     実行専用一時ディレクトリを作成
       ↓
     対象をダウンロード
       ↓
     成功したデータソースだけをインポート
       ├─ 成功 0 件: 一時ディレクトリ削除、DB 変更なし、失敗終了
       └─ 1 件以上
            ↓
          ワークスペース構築
            ↓
          MySQL 単一トランザクションで同期
            ↓
          一時ディレクトリ削除
            ↓
          ロック解放
```

### 6.2 大型更新

```text
起動 --major-update
  ↓
共通ロックを試行
  ├─ 失敗: エラー終了
  └─ 成功
       ↓
     official と additional_songs だけを解決
       ├─ どちらか解決失敗: DB 変更なし、失敗終了
       └─ 両方解決
            ↓
          実行専用一時ディレクトリへ両方を取得
            ├─ どちらか取得失敗: DB 変更なし、失敗終了
            └─ 両方成功
                 ↓
               両方の解析成功を確認
                 ↓
               ワークスペース構築
                 ↓
               MySQL 単一トランザクションで同期
                 ↓
               一時ディレクトリ削除
```

対象外データソースの前回 JSON が `.datasources/` に残っていても読まない。

### 6.3 `--skip-download`

```text
起動 --skip-download
  ↓
共通ロックを試行
  ↓
.datasources/ の既存ファイルを入力にする
  ↓
モードの必須条件を適用
  ↓
ダウンロードも一時ディレクトリも使わない
  ↓
インポートと同期
```

この経路だけが固定ディレクトリを使ってよい。開発時の明示利用であり、cron の通常実行では使わない。

---

## 7. 内部設計

### 7.1 Downloader

`DownloadAll` の戻り値を結果集合へ変える。一部失敗を「全体成功」と見なして呼び出し側がファイル有無を推測する形はやめる。

呼び出し側は結果集合から成功分だけを Importer に渡す。Downloader は前回ディレクトリを見ない。渡された `outputDir` にだけ書く。

既存テスト `TestDownloader_DownloadAll_ErrorHandling` は「一部失敗でも error が nil」を期待している。この期待は新しい結果集合契約へ置き換える。一部失敗は全体 error ではなく、失敗した要素の `Success=false` として表す。テストケース自体が検証している「失敗ファイルを書かない」は残す。

### 7.2 データソース解決

通常モード: 現行どおり、解決失敗したデータソースは警告してスキップする。

大型更新モード: official と additional_songs だけを解決する。どちらかが解決できなければ、その時点で失敗する。st1027 / mainframe / otoge-db の環境変数欠如は大型更新の失敗理由にしない。

### 7.3 インポート

入力は「種別とファイルパス」の成功リストだけとする。解決済み一覧を再走査して固定パスを組み立てない。

ファイルが無い、JSON が空、解析不能は、そのデータソースの失敗とする。通常モードならそのソースを除く。大型更新なら全体失敗とする。

### 7.4 同期の開始条件

MySQL トランザクションは、モードの必須条件を満たしたあとで開始する。ダウンロードや解析の失敗でトランザクションを開かない。

現行どおり、ワークスペース構築はトランザクション開始前に行う。

### 7.5 ロック実装

`internal/infra/db` に接続単位のアドバイザリロックを置く。API 側の `advisory_lock.go` と同じ考え方でよい。song-batch から api パッケージを import しない。

ロック名は `internal/info` の定数にする。ハードコードしない。

テストは、取得成功、未取得時のモード別終了、解放後に再取得できることを確認する。実 MySQL が必要な場合は既存の DB テスト方針に合わせ、ユニットでは provider を差し替えられるようにする。

### 7.6 パッケージ配置

- ユースケース: `internal/usecase` を新設する。現行の `internal/service`（統合）は残し、ユースケースがそれを呼ぶ
- ロック: `internal/infra/db`
- ロック名: `internal/info`
- 一時ディレクトリのプレフィックスなど固定値: `internal/info`

`main.go` に取得・インポート・統合の手続きを残さない。

---

## 8. エラー時の仕様

| 状況 | MySQL の楽曲・譜面 | 終了 |
|---|---|---|
| 通常 cron がロックを取得できない | 変更なし | 終了コード 0 |
| `--major-update` がロックを取得できない | 変更なし | 非 0 |
| 通常実行で一部データソース取得失敗 | 成功分だけ同期 | 成功終了。失敗はログ |
| 通常実行で取得成功 0 件 | 変更なし | 非 0 |
| 大型更新で official 解決・取得・解析失敗 | 変更なし | 非 0 |
| 大型更新で additional_songs 解決・取得・解析失敗 | 変更なし | 非 0 |
| `--skip-download` で必須ファイル欠落 | 変更なし | 非 0 |
| ワークスペース構築失敗 | 変更なし | 非 0 |
| MySQL 同期失敗 | ロールバック | 非 0 |

失敗時に `.datasources/` へ部分ファイルを残して次回の入力にしない。ダウンロード実行の成果物は一時ディレクトリに閉じ、終了時に削除する。

外部サービスのレスポンス本文、API キー、DB 接続情報をログの失敗理由へ含めない。

---

## 9. 既存 CLI / cron 互換

維持するもの:

- フラグ名 `--skip-download`、`--major-update`、`--fill-missing-release-date`
- 通常実行が「取れたデータソースだけ統合する」こと
- 大型更新が official と additional_songs だけを統合し、定数不明化ルールを使うこと
- 既存 cron の起動コマンドと時刻

変わるもの:

- 通常実行が `.datasources/` を作らなくなる
- ダウンロード失敗時に前回 JSON を読まなくなる
- 大型更新が対象外データソースを解決・取得しなくなる
- 実行中は他の song-batch が開始されない
- 通常 cron がロック競合でスキップすることがある（終了コード 0）

README の「初回実行時に `.datasources/` が生成される」「mainframe 失敗時は `.datasources/mainframe.json` を削除する」は、新しい契約に合わせて直す。

---

## 10. 実装計画

各工程はテストを先に追加し、Red、Green、Refactor の順で進める。既存テストは、契約変更で期待が古くなったもの以外は削除しない。

1. 通常ダウンロードが前回ファイルを再利用しないことを示すテストを追加する
2. 実行専用一時ディレクトリを導入する。終了時削除を含める
3. Downloader がデータソース単位の結果を返すように変える
4. Importer 入力を今回成功分だけに変える
5. 大型更新時は official と additional_songs だけを解決するように変える
6. 必須データソースの今回取得失敗時に DB 更新しないテストを追加し、実装する
7. 全モードへ共通アドバイザリロックを導入する
8. CLI 処理を `internal/usecase` へ分離する
9. README を更新する

工程 1 から 6 をロックより先に完了する。取得安全性が無い状態で実行経路を増やさないためである。

---

## 11. テスト計画

- 固定 `.datasources/` に古い official.json がある状態で、今回の official 取得が失敗しても、その古いファイルをインポートしない
- 通常実行が一時ディレクトリを使い、終了後に削除する
- 通常実行が `.datasources/` へ書き込まない
- `--skip-download` のときだけ `.datasources/` を使う
- `--skip-download` でファイルが無いデータソースは、通常モードなら対象外、大型更新なら全体失敗
- 大型更新が st1027 / mainframe / otoge-db を解決・ダウンロードしない
- 大型更新で official または additional_songs の今回取得失敗時、同期処理を呼ばない
- 大型更新で両方取得・解析できたときだけ同期する
- 通常実行で一部失敗しても、成功分の同期は行う
- 通常実行で成功 0 件なら同期しない
- ロック取得失敗時、通常と `--fill-missing-release-date` は終了コード 0 相当のスキップになる
- ロック取得失敗時、`--major-update` はエラーになる
- ロック保持中は別実行が取得できない
- 解放後は再取得できる
- `--major-update` と `--skip-download` なしの実行が同じユースケースを通る

Mainframe の空データ拒否、矛盾する定数、未解決譜面、ジョブ復旧は後続設計のテストであり、本書では扱わない。

---

## 12. 受け入れ条件

- 通常実行が、今回取得に失敗したデータソースについて前回 JSON を利用しない
- `--skip-download` を付けない限り `.datasources/` を入力にも出力にも使わない
- 大型更新が official と additional_songs 以外を解決・取得しない
- 大型更新で official または additional_songs を今回取得できない場合、DB を変更しない
- 通常 cron と `--major-update` と `--fill-missing-release-date` が同時に song-batch 更新を実行しない
- 通常 cron がロック競合で失敗扱いにならず、スキップして終了コード 0 になる
- 明示的な `--major-update` がロック競合でエラー終了する
- 既存フラグの意味と、取れたソースだけ統合する通常実行の意味が維持される
- `main.go` に取得・インポート・統合の手続きが残っていない

---

## 13. 後続作業との境界

本書が終わったあとに、管理画面設計へ進む。後続で足すものは次である。

- DB の設定行とジョブ行
- Mainframe 候補 ID による取得・検証
- `--run-pending-job`
- ジョブ phase と結果件数
- リース期限切れの復旧

後続は本書の一時ディレクトリ、データソース単位結果、必須条件、共通ロック、ユースケースを前提にする。取得安全性を後続へ先送りしない。
