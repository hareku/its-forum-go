# rc016v1 — RC-016 1.0

ITS FORUM RC-016 **1.0** の自転車・歩行者、通常路側機、CSMA型路側機の全3系統を読み書きするGoモジュールです。Go 1.27.0以降を使用します。モジュールパスは `github.com/hareku/its-forum-go/rc016v1`、package名は `rc016` です。

`rc013v1` のRC-013 1.1の型・共通DFを再利用します。第三者Goライブラリへの依存はありません。名前の `v1` は規格majorで、Go API SemVerやwire上の `Header.Version` とは独立です。同じ規格majorの全minor版対応や、RC-016規格major 2の実装を意味しません。

## 導入

初回バージョンは `v0.1.0` です。利用側のGoモジュールで次を実行します。

```sh
go get github.com/hareku/its-forum-go/rc016v1@v0.1.0
```

このモジュールのディレクトリ内で、公開版のRC-013を使った単独の検証ができます。

```sh
GOWORK=off go test -mod=readonly -count=1 ./...
GOWORK=off go vet -mod=readonly ./...
GOWORK=off go mod verify
```

リポジトリ全体をcheckoutした場合は、**リポジトリのルート**から `GOWORK=off go run ./scripts/verify-modules.go` で配布ZIP・外部consumerも検査できます。runnerのローカル配布検証は実公開取得の証明ではありません。

## 読み取り・編集・書き込み

CSMAメッセージを作成し、読み戻して速度を編集する例です。`Speed`の単位は0.01m/sです。

```go
package main

import (
    "fmt"
    rc016 "github.com/hareku/its-forum-go/rc016v1"
)

func main() {
    message := rc016.CSMAMessage{
        Header: rc016.CSMAHeader{Version: 1},
        Objects: []rc016.CSMAObject{{ID: 7, Type: 4, Size: 2}},
    }
    data, err := message.MarshalBinary()
    if err != nil {
        panic(err)
    }
    decoded, err := rc016.DecodeCSMA(data)
    if err != nil {
        panic(err)
    }
    decoded.Objects[0].Speed = 1234
    data, err = decoded.MarshalBinary()
    if err != nil {
        panic(err)
    }
    fmt.Println(len(data), decoded.Objects[0].Speed) // 36 1234
}
```

上の値はAPI操作の例です。実運用のIDや各項目は利用する実験に合わせて設定してください。省略したGoフィールドはゼロになり、不定値へ自動変換されません。

| 系統 | 復号API | 構築・プロファイル |
|---|---|---|
| 自転車・歩行者 | `DecodeBicycle` / `DecodeBicycleWithProfile` | `NewBicycle`、`BicycleProfile`で個別アプリIDを4種類のDFへ対応付け |
| 通常路側機 | `DecodeRoadside` / `DecodeRoadsideWithProfile` | `NewRoadside`、`RoadsideProfile`でセンサ属性・グループ順序・個別拡張DEを指定 |
| CSMA型路側機 | `DecodeCSMA` | `CSMAMessage`を構築。物標0〜5件 |

プロファイル中のIDは利用者が指定する実験値です。未指定の個別データは生バイトとして保持します。通常路側機でセンサとの対応を確定できない場合は、判明したヘッダ・属性と未解釈の残りを返します。`ObjectsDecoded()`で物標まで解釈できたか確認できます。センサの`SensorOpaque`プロファイルで状態セレクタを省略すると、そのセンサのグループが常に存在する指定になります。

各モデルの`MarshalBinary`で書き込みます。長さ・件数・既知領域の有無・アドレスは内容から算出します。使用例は[Examples](example_test.go)を参照してください。

## 保持・検証の契約

整数フィールドは規格上の符号化値です。未知値、予約ビット、不定値、未知ペイロード、受け入れたギャップを保持します。構造が正しい対応形式は、無変更で復号・再符号化すると同じバイト列になります。未知データの開始位置を変える編集は`ErrLayout`になる場合があります。明示的な再構築は新しい配置を指定する操作です。

復号時に入力バイト列とプロファイルをコピーし、符号化時は新しいバイト列を返します。公開スライス・ポインタは編集できますが、Goの構造体コピーは通常どおり浅いコピーです。`Validate()`は意味上の範囲・予約値を報告し、値を変更しません。長さ不整合や表現ビット幅の超過は復号・符号化自体が拒否します。エラーは`errors.Is`と`errors.As`に対応します。

## 規格上の留保

- 通常路側機ヘッダは表4-3の4ビット版数・16バイト構造を採用しています。§4.4.2.1.1の「1ビット」との矛盾は解消済みとは扱いません。
- 自転車のケイデンスは定義と分解能欄の単位が矛盾するため、生整数で扱います。個人共通DFは付録の遅延省略記述にかかわらず本文の固定5バイト構造を採用します。
- 実機通信、暗号化、物標の補完・統合処理、CLI/UIは含みません。検証は規格から独立計算した固定データとテストによるもので、実機相互接続の認証ではありません。

仕様: [RC-016 1.0](https://itsforum.gr.jp/Public/J7Database/p68/ITS_FORUM_RC-016_v10.pdf)。

## RC-013との型・エラーの共有

公開モデルの `Speed` などにはRC-013の型を使用します。両方を利用するときは `rc013 "github.com/hareku/its-forum-go/rc013v1"` と `rc016 "github.com/hareku/its-forum-go/rc016v1"` のようにimportできます。

`rc016.Error` と `rc016.Issue` はRC-013の型aliasです。`ErrTruncated`、`ErrMalformed`、`ErrRange`、`ErrUnsupported`、`ErrLayout` もRC-013と同じエラー値で、例えば `rc016.ErrLayout == rc013.ErrLayout` が成立します。


## 開発・ライセンス

リポジトリ全体をcheckoutした場合の開発・CI手順はルートの `README.md`、版管理・公開順序は `docs/releasing.md` を参照してください。[リポジトリ](https://github.com/hareku/its-forum-go)からも参照できます。これらの全体文書は個別モジュールの配布ZIPには含みません。

[MIT License](LICENSE)。
