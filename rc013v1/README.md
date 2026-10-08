# rc013v1 — RC-013 1.1

ITS FORUM RC-013 **1.1** の単独メッセージ、共通DF、任意・自由領域を読み書きするGoモジュールです。Go 1.27.0以降を使用し、他のGoモジュールへの依存はありません。モジュールパスは `github.com/hareku/its-forum-go/rc013v1`、package名は `rc013` です。

名前の `v1` は規格majorで、Go API SemVerやwire上の `Header.Version` とは独立です。対応する完全な規格版は1.1で、同じmajorの全minor版対応を意味しません。

## 導入

初回バージョンは `v0.1.0` です。利用側のGoモジュールで次を実行します。

```sh
go get github.com/hareku/its-forum-go/rc013v1@v0.1.0
```

このモジュールのディレクトリ内で単独の検証ができます。

```sh
GOWORK=off GOPROXY=off go test -count=1 ./...
GOWORK=off GOPROXY=off go vet ./...
```

リポジトリ全体をcheckoutした場合は、**リポジトリのルート**から `GOWORK=off go run ./scripts/verify-modules.go` で配布ZIP・外部consumerも検査できます。ローカル配布検証は実公開取得の証明ではありません。

## 構築・読み取り・編集・書き込み

`Message` を構築し、`MarshalBinary` → `Decode` → 速度編集 → 再符号化・再復号を行う例です。`Speed` の単位は0.01m/sです。

```go
package main

import (
	"fmt"
	rc013 "github.com/hareku/its-forum-go/rc013v1"
)

func main() {
	message := rc013.Message{
		Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1, VehicleID: 42},
		Common: rc013.CommonData{
			Time:              rc013.Time{Hour: 12, Minute: 34, Millisecond: 56789},
			Position:          rc013.Position{Latitude: 350000000, Longitude: 1390000000, Elevation: rc013.ElevationUnavailable},
			VehicleState:      rc013.VehicleState{Speed: 1234, Heading: 7200, SteeringWheelAngle: rc013.SteeringWheelAngleUnavailable},
			VehicleAttributes: rc013.VehicleAttributes{SizeClass: 2, RoleClass: 0, Width: 180, Length: 450},
		},
	}
	data, err := message.MarshalBinary()
	if err != nil {
		panic(err)
	}
	decoded, err := rc013.Decode(data)
	if err != nil {
		panic(err)
	}
	decoded.Common.VehicleState.Speed = 1500
	updated, err := decoded.MarshalBinary()
	if err != nil {
		panic(err)
	}
	decoded, err = rc013.Decode(updated)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(updated), decoded.Header.VehicleID, decoded.Common.VehicleState.Speed)
	// Output: 36 42 1500
}
```

出力は `36 42 1500` です。詳細は[Examples](example_test.go)を参照してください。値はAPI操作の例なので、実運用では利用する実験に合わせて設定します。省略したGoフィールドはゼロになり、不定値へ自動変換されません。

## メッセージとDF

| API・型 | 用途 |
|---|---|
| `Decode` / `Message.MarshalBinary` | 単独基本メッセージを読み書き。全体36〜100バイト、対応識別値は `ServiceID=1`, `MessageID=1`, `Version=1` |
| `DecodeCommonData` / `CommonData.MarshalBinary` | 管理ヘッダ・自由領域を除いた共通DF列。任意DFの有無はポインタで表現 |
| `DecodeFreeArea` / `FreeArea.MarshalBinary` | 1〜7件の自由領域。`FreeEntry` にサービスID、未知データ、直前のギャップを保持 |
| 固定長DFの `MarshalBinary` / `UnmarshalBinary` | 個々のDFの読み書き。入力長は固定長と完全一致が必要で、復号失敗時にreceiverを変更しない |

固定長DFは `Header`（8バイト）、`Time`（4）、`Position`（11）、`VehicleState`（9）、`VehicleAttributes`（4）、`PositionOptional`（2）、`GPSStatusOptional`（4）、`PositionAcquisitionOptional`（2）、`VehicleStateOptional`（7）、`Intersection`（10）、`Extension`（1）です。共通DFの必須部分は28バイトです。

メッセージの長さ・option flags、自由領域の件数・アドレス・長さは内容から算出します。`Message.Free == nil` は自由領域なしを表し、0件の自由領域ヘッダとは区別します。任意DFのnilと、存在する全ゼロDFも別です。共通DF・自由領域の単体helperには単独メッセージ全体の100バイト上限を適用しません。

## 保持・編集・検証の契約

整数フィールドは規格上の符号化値です。未知値、予約ビット、不定値、未知ペイロード、受け入れたギャップを保持します。構造が正しい対応形式は、無変更で復号・再符号化すると同じバイト列になります。復号時に保持する入力バイト列をコピーし、符号化時は新しいバイト列を返します。公開スライス・ポインタは編集できますが、Goの構造体コピーは通常どおり浅いコピーです。

未知データの開始位置を変える編集は `ErrLayout` になる場合があります。`CommonData.OpaqueTail` は既知DF後の位置に固定され、先行DFのサイズ変更による移動を拒否します。復号した自由領域も未知ペイロードの位置を保持します。新しい `FreeArea{Entries: ...}` などとして明示的に再構築する操作は、新しい配置を指定する境界です。

`Validate()` は意味上の範囲・予約値を報告し、値を変更しません。長さ不整合や表現ビット幅の超過は復号・符号化自体が拒否します。エラーは `errors.Is` と `errors.As` に対応し、`Error` に位置・フィールド情報、`Issue` に意味上の指摘を保持します。

## 規格上の留保

- ビット列はMSB先頭・big-endianで扱い、ASN.1符号化ではありません。
- 未定義拡張オプションは、既知DF後から宣言長末尾までを不透明な領域として扱う配置に対応します。任意の将来版レイアウトを推測しません。
- 実機通信、暗号化、物標の補完・統合処理、CLI/UIは含みません。検証は規格から独立計算した固定データとテストによるもので、実機相互接続の認証ではありません。

仕様: [RC-013 1.1](https://itsforum.gr.jp/Public/J7Database/p60/ITS_FORUM_RC-013_v11.pdf)。


## 開発・ライセンス

リポジトリ全体をcheckoutした場合の開発・CI手順はルートの `README.md`、版管理・公開順序は `docs/releasing.md` を参照してください。[リポジトリ](https://github.com/hareku/its-forum-go)からも参照できます。これらの全体文書は個別モジュールの配布ZIPには含みません。

[MIT License](LICENSE)。
