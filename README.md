# its-forum-go

ITS FORUM の公開RC規格をGoの型として読み書きするライブラリ群です。Go 1.27.0以降を使用します。第三者Goライブラリへの依存はなく、RC-016モジュールは同じリポジトリのRC-013モジュールに依存します。

リポジトリは `github.com/hareku/its-forum-go` です。**各モジュールの初回予定のバージョン付きリリース `v0.1.0` は未公開です。** 公開proxyからのバージョン指定取得は未検証です。

| モジュールパス・利用ガイド | package名 | 対応する規格と範囲 |
|---|---|---|
| [`github.com/hareku/its-forum-go/rc013v1`](rc013v1/README.md) | `rc013` | RC-013 **1.1**：単独メッセージ、共通DF、任意・自由領域 |
| [`github.com/hareku/its-forum-go/rc016v1`](rc016v1/README.md) | `rc016` | RC-016 **1.0**：自転車・歩行者、通常路側機、CSMA型路側機の全3系統 |
| [`github.com/hareku/its-forum-go/rc016v2`](rc016v2/README.md) | `rc016` | RC-016 **2.0**：自転車・歩行者の完全な個別アプリpayload |

名前の `v1` / `v2` は規格majorを表します。対応は上記の完全な規格版に限り、同じmajorの全minor版への対応を意味しません。Go APIのリリース番号、wire上の `Header.Version` とは独立です。RC-019や他のRCの実装は含みません。

利用例、API、未知データの保持・編集契約、規格上の留保は、上表の各モジュールの利用ガイドを参照してください。

## 開発と公開前の検証

ルートはGoモジュールではなく、[go.work](go.work)で3つのモジュールを同時開発します。リポジトリのルートで、Go 1.27.0を用意して次を実行してください。

```sh
set -eu
go version
format_output=$(gofmt -l rc013v1 rc016v1 rc016v2 scripts)
if [ -n "$format_output" ]; then
    printf '%s\n' "$format_output"
    exit 1
fi
GOPROXY=off go test -count=1 ./rc013v1/... ./rc016v1/... ./rc016v2/...
GOPROXY=off go vet ./rc013v1/... ./rc016v1/... ./rc016v2/...
GOWORK=off go run ./scripts/verify-modules.go
```

workspaceのtest／vetにはコマンド単位で `GOPROXY=off` を指定し、未公開の兄弟モジュールのメタデータ取得を避けます。独立検証runnerは別途、自身の一時file proxyを設定します。

[検証runner](scripts/verify-modules.go)は候補ソースの配布ZIP、workspace外の各モジュール・外部consumer、依存graph、LICENSE、期待した失敗を空cacheと一時file proxyで検査します。公開サービスへの問い合わせは不要です。RC-013未公開時のRC-016独立検証にもこのrunnerを使います。

これは実公開取得の証明ではありません。一時proxy・cache・checksumは隔離し、製品の `go.sum` へ転用しません。実公開版からのchecksum生成は[リリース手順](docs/releasing.md)に従います。

[CI](.github/workflows/ci.yml)はpush・pull request時に同じformat・workspace test/vet・独立検証を実行します。新しいモジュールの追加手順と規格版／Go API版の運用も[リリース手順](docs/releasing.md)を参照してください。

## ライセンス

[MIT License](LICENSE)。各モジュールにもルートと完全に同じLICENSEを配置し、個別の配布ZIPに含めます。
