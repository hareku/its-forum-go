# rc016v2 — RC-016 2.0

ITS FORUM RC-016 **2.0** の自転車・歩行者送信（B2I/P2I/B2V/P2V）を読み書きするGoモジュールです。Go 1.27.0以降を使用します。module pathは `github.com/hareku/its-forum-go/rc016v2`、package名は `rc016` です。RC-013 **1.1** の共通領域・型・codecを `rc013v1` から再利用し、第三者ライブラリへの依存はありません。

`v2` は規格majorです。対応する完全な規格版は2.0のみで、Go API SemVer、wireの `Header.Version` とは独立です。共通ヘッダのServiceID/MessageID/Versionは **1/1/1** です。路側機送信はRC-019へ移管されており、本モジュールには含みません。

## 導入と検証

**初回予定のバージョン付きリリース `v0.1.0` は未公開です。** 以下はRC-013と本モジュールのタグ公開後に、利用側のGoモジュールで実行します。

```sh
go get github.com/hareku/its-forum-go/rc016v2@v0.1.0
```

現在はリポジトリ全体をcheckoutし、ルートで `GOWORK=off go run ./scripts/verify-modules.go` を実行してください。空cache・一時file proxyによる独立配布検証であり、公開proxyからの取得の証明ではありません。未公開のRC-013に依存するため、単独ディレクトリでの通常の `GOWORK=off go test ./...` は依存を取得できません。

## 構築・読み取り・編集

一つの個別アプリに、自転車は `PersonalCommon` 5 + `BicycleBasic` 3 + `BicycleExtended` 14 = **22バイト**、歩行者は `PersonalCommon` 5 + `Pedestrian` 5 = **10バイト**を格納します。DFごとに別の個別アプリへ分割しません。単体DFおよび `BicycleData` / `PedestrianData` にもexact-sizeの `MarshalBinary` / `UnmarshalBinary` を提供します。

```go
package main

import (
    "fmt"
    rc013 "github.com/hareku/its-forum-go/rc013v1"
    rc016 "github.com/hareku/its-forum-go/rc016v2"
)

func main() {
    // Service IDs are selected by this example's experiment.
    profile := rc016.Profile{Applications: map[uint8]rc016.ApplicationKind{
        0x90: rc016.ApplicationBicycle,
        0x91: rc016.ApplicationPedestrian,
    }}
    message, err := rc016.NewMessage(rc016.Fields{
        Header: rc013.Header{ServiceID: 1, MessageID: 1, Version: 1},
        Applications: []rc016.Application{{
            ServiceID: 0x91,
            Pedestrian: &rc016.PedestrianData{
                PersonalCommon: rc016.PersonalCommon{Level: rc016.LevelUnavailable},
                Pedestrian: rc016.Pedestrian{ItemInfo: 1, Steps: 100},
            },
        }},
    }, profile)
    if err != nil { panic(err) }
    wire, err := message.MarshalBinary()
    if err != nil { panic(err) }
    decoded, err := rc016.DecodeWithProfile(wire, profile)
    if err != nil { panic(err) }
    decoded.Applications[0].Pedestrian.Pedestrian.Steps = 60000
    wire, err = decoded.MarshalBinary()
    if err != nil { panic(err) }
    fmt.Println(len(wire), decoded.Applications[0].Pedestrian.Pedestrian.Steps) // 50 60000
}
```

ID `0x90` / `0x91` は例の実験値です。規格の既定IDでも、自動判別の識別子でもありません。`Decode` は全個別アプリをrawで保持し、`DecodeWithProfile` は指定IDを完全な組のtypedデータへ復号します。自転車例を含む実行可能な例は [Examples](example_test.go) を参照してください。省略した値はGoのゼロ値となり、不定値へ自動変換しません。

## 保持・エラー・編集の契約

- mapped appは対応する `Bicycle` / `Pedestrian` ポインタが権威を持ち、`Data` は復号時のsnapshotです。typed編集をrawで上書きしません。unmapped appは `Data` が権威を持ち、typedポインタを拒否します。種類不一致・未知profile kind・複数typedポインタは構造エラーです。
- 受け入れた未知値・予約bits・gap・Commonのopaque tail・未知payload・自由領域の不在を保持します。無変更で再符号化すると全バイトが一致します。長さ、count、flag、addressは構築内容から計算し、RC-013の36〜100バイト上限やpresent自由領域1〜7件の構造に従います。
- 復号した未知payloadのpacket絶対開始位置を動かす編集は `ErrLayout` です。共通optionalの追加、appの追加・並べ替え、gap変更も該当し得ます。明示的な `NewMessage` は呼出側が新配置を選択する再構築境界です。
- 復号・構築時は入力bytes、gap、共通optional、typedポインタ、profile mapをコピーします。marshalは新しいbytesを返します。公開structの通常の代入はGoの浅いコピーです。DF/payloadの `UnmarshalBinary` は失敗時にreceiverを変更しません。
- exact-size不一致、bit幅超過、不正な構造はcodecが拒否します。`Validate()` は意味上の懸念を `[]Issue` として報告し、値を補正しません。不定値はそれ自体をIssueにしません。

`Error` / `Issue` はRC-013の型aliasです。`ErrTruncated`、`ErrMalformed`、`ErrRange`、`ErrUnsupported`、`ErrLayout` も同一のエラー値です。`errors.Is` / `errors.As` に対応し、message中のエラーoffsetはpacket先頭基準です。Header/CommonにはRC-013の公開型を直接使います。

## v2の値と診断

整数は符号化値です。物理量変換・丸め・歩数からのMotion自動算出は行いません。

| フィールド | v2の意味 |
|---|---|
| Level | 1〜5は搭載機器能力、`LevelUnavailable` = 7不定。0/6は未割当 |
| SystemDelay | 10ms単位、`SystemDelaySaturated` = 30は300ms以上、`SystemDelayUnavailable` = 31不定 |
| BicycleType | 0不定、1〜7定義済み、8〜15予約 |
| ItemInfo | 1子供靴、2高齢者靴、`ItemInfoUnavailable` = 63不定。他は予約 |
| Steps | **16bit**、`StepsSaturated` = 65534以上、`StepsUnavailable` = 65535不定。片側装置の歩数を2倍にしない |
| Motion | 1分あたり20未満/20〜139/140以上を0/1/2で表す。3不定。bit22から2bit |
| Pedestrian.Reserved | bit24から16bit。生値を保持 |
| Cadence | 分解能1rpm。254飽和、255不定 |

全DFのbit配置・不定・飽和・予約値と独立計算goldenは [仕様ベクトル](testdata/spec-vectors.md) を参照してください。Collisionの1〜15はTBDであり、固定意味や一律のinvalid判定を設けません。

表3-1で不定指定・うるう秒0固定の共通項目と既知level1〜5の整合性は警告型の `Issue` で診断します。「可能」は取得不能時の不定を禁止しません。level7では能力を推定せず、複数appは各appのlevelと共有Commonをそれぞれ検査します。codecによる拒否・自動補正はしません。

## 規格上の留保・範囲

表3-1のCounter・車両ID・重量の構造記載はRC-013と整合しません。§3.2.1の参照先RC-013を構造規範とし、誤記が解決済みとは扱いません。歩行者予約領域の非ゼロIssueはライブラリの診断policyであり、明文の送信0指定を根拠にしていません。

実機通信、暗号化、補完・統合、CLI/UI、RC-019は対象外です。検証は規格から独立計算した固定データとテストに基づき、実機相互接続の認証ではありません。

仕様: [RC-016 2.0](https://itsforum.gr.jp/Public/J7Database/p73/ITS_FORUM_RC-016_v20.pdf)、共通領域: [RC-013 1.1](https://itsforum.gr.jp/Public/J7Database/p60/ITS_FORUM_RC-013_v11.pdf)。

## 開発・ライセンス

全体の開発・CI手順はリポジトリのルート `README.md`、版管理・公開順序は `docs/releasing.md` を参照してください。これらは個別ZIPには含まれないため、[リポジトリ](https://github.com/hareku/its-forum-go)でも参照できます。依存先RC-013公開後に `rc016v2/v0.1.0` を独立して公開する予定です。

[MIT License](LICENSE)。モジュール配布ZIPにルートと同一のLICENSEを含めます。
