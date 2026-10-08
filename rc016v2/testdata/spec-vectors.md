# RC-016 2.0 独立ベクトルと DE matrix

一次資料: https://itsforum.gr.jp/Public/J7Database/p73/ITS_FORUM_RC-016_v20.pdf
印刷ページ番号を使用。MSB-first、DF 内 bit 0 基準。29 DE と予約 2 領域を網羅。
規範は表4-1〜4-4と各 DE 節。以下の固定 hex は production codec/bitio を使わず、各整数を指定幅の2進文字列にして連結し16進へ変換したもの。

## 全DE field matrix（DF先頭bit0、MSB-first）

| DF / planned Go field | offset:width | 通常/飽和/不定/予約・意味 | 規格節 |
|---|---|---|---|
| PersonalCommon.Level | 0:3 | 1..5能力level、7不定、0/6未割当 | 5.1.1/p14 |
| PersonalCommon.SystemDelay | 3:5 | 0..29×10ms、30=300ms以上、31不定。取得から送信まで、最長の遅延 | 5.1.2/p14 |
| PersonalCommon.WatchData | 8:32 | 実験独自内容、未使用0。内容を意味付けしない | 5.1.3/p15 |
| BicycleBasic.AssistType | 0:4 | 0不定、1/2定義、3..15予約 | 5.2.1/p15 |
| BicycleBasic.BicycleType | 4:4 | 0不定、1City/2Cross/3Road/4MTB/5子乗せ/6幼児/7三輪、8..15予約 | 5.2.2/p16 |
| BicycleBasic.AssistState | 8:2 | 0不定/1OFF/2ON/3自走ON | 5.2.3/p16 |
| BicycleBasic.Pedaling | 10:2 | 0不定/1なし/2中/3予約 | 5.2.4/p16 |
| BicycleBasic.DrivePower | 12:8 | 0..253×10W、254=2540W以上、255不定 | 5.2.5/p17 |
| BicycleBasic.Collision | 20:4 | 0不定、1..15TBD。独自意味を割り当てない | 5.2.6/p17 |
| BicycleExtended.MainGear | 0:5 | 0不定、1..31。31を超えた実値の飽和指示なし | 5.3.1/p17 |
| BicycleExtended.MainMaxGear | 5:5 | 0不定、1..31 | 5.3.2/p18 |
| BicycleExtended.SubGear | 10:5 | 0不定、1..31。単一シフトでは0 | 5.3.3/p18 |
| BicycleExtended.SubMaxGear | 15:5 | 0不定、1..31。単一シフトでは0 | 5.3.4/p18 |
| BicycleExtended.TireCircumference | 20:8 | 0不定、1..254×10mm、255=2550mm以上 | 5.3.5/p18 |
| BicycleExtended.Cadence | 28:8 | 0..253rpm、254=254rpm以上、255不定、分解能1rpm | 5.3.6/p19 |
| BicycleExtended.GearRatio | 36:10 | 0不定、1..1022%、1023=1023%以上。後輪/クランク回転数 | 5.3.7/p19 |
| BicycleExtended.RiderTorque | 46:8 | 0..253Nm、254=254Nm以上、255不定 | 5.3.8/p19 |
| BicycleExtended.MotorTorque | 54:8 | 0..253Nm、254=254Nm以上、255不定 | 5.3.9/p19 |
| BicycleExtended.MaxAssistPower | 62:8 | 0..253×10W、254=2540W以上、255不定 | 5.3.10/p20 |
| BicycleExtended.AssistPower | 70:8 | 0..253×10W、254=2540W以上、255不定 | 5.3.11/p20 |
| BicycleExtended.HumanPower | 78:8 | 0..253×5W、254=1270W以上、255不定 | 5.3.12/p20 |
| BicycleExtended.MaxBattery | 86:8 | 0..253×10Wh、254=2540Wh以上、255不定 | 5.3.13/p20 |
| BicycleExtended.Battery | 94:8 | 0..253×10Wh、254=2540Wh以上、255不定 | 5.3.14/p21 |
| BicycleExtended.RearLight | 102:2 | 0不定/1OFF/2ON/3予約 | 5.3.15/p21 |
| BicycleExtended.DriveUnitStatus | 104:2 | 0不定/1正常/2故障/3予約 | 5.3.16/p21 |
| BicycleExtended.Maintenance | 106:2 | 0不定/1正常/2故障/3予約（原文も故障） | 5.3.17/p22 |
| BicycleExtended.Reserved | 108:4 | 送信0指定。raw受信保持、非zeroIssue | 4.3/p12 |
| Pedestrian.ItemInfo | 0:6 | 1子供靴/2高齢者靴、63不定、0と3..62予約 | 5.4.1/p22 |
| Pedestrian.Steps | 6:16 | 検知歩数0..65533、65534以上は65534、65535不定。片側装置でも2倍にしない | 5.4.2/p22 |
| Pedestrian.Motion | 22:2 | 1分あたり0=<20、1=20..139、2=>=140、3不定 | 5.4.3/p23 |
| Pedestrian.Reserved | 24:16 | 表で予約16bit。明文の送信0指定を発見せず | 表4-4/p13 |

合計29 DEと予約2領域。DF lengths=40/24/112/40bit。連結offsetは自転車PersonalCommon0/Basic40/Extended64、歩行者PersonalCommon0/Pedestrian40。


## 手計算の固定値

各行は matrix の field 順。連結式 `N = (((v0 << w1) | v1) << w2 | v2) ...` を最終 big endian byte 列へ変換する。先頭のゼロを DF 長まで埋める。

| DF | field 順の値 (decimal unless 0x) | 独立 hex |
|---|---|---|
| PersonalCommon | 5,29,0x12345678 | bd12345678 |
| BicycleBasic | 2,7,3,2,173,11 | 27eadb |
| BicycleExtended | 17,23,9,13,201,87,683,42,91,123,145,167,189,211,2,1,2,10 | 8dd2dc957aaca96dee469ef74e6a |
| Pedestrian | 2,0x9abc,2,0x1357 | 0a6af21357 |

- PersonalCommon: `(5<<5)|29 = 0xbd` のあと WatchData 12 34 56 78。
- Basic: 最初の 2/7 は 0x27。続く `11 10 10101101 1011` は ea db。
- Extended: 幅 `5,5,5,5,8,8,10,8,8,8,8,8,8,8,2,2,2,4` の整数列連結。最初の20bitは `10001 10111 01001 01101` (8dd2d)。末尾16bitは Battery の残り6bit `010011` + RearLight `10` + DU `01` + Maintenance `10` + Reserved `1010` という境界で、最後の2byteは 4e 6a。
- Pedestrian: `000010 1001101010111100 10 0001001101010111` = 0a 6a f2 13 57。Steps は byte0 の下2bitから byte2 の上6bitまで、Motion は byte2 下2bit、Reserved は byte3/4。
- 自転車 complete app: PersonalCommon bit0 + Basic bit40 + Extended bit64、`bd1234567827eadb8dd2dc957aaca96dee469ef74e6a` (22byte)。
- 歩行者 complete app: PersonalCommon bit0 + Pedestrian bit40、`bd123456780a6af21357` (10byte)。

`TestFrameAndPayloadIndependentGoldens` は上記 literal と型を双方向比較する。packet fixture は同 directory の bicycle.hex/pedestrian.hex と message tests を参照。

## field coverage と診断

`TestPersonalFixedFramesEveryField` は matrix の全31行についてゼロ/1/最大の独立 `math/big` 整数配置、decode、範囲超過、全bits1保持、0xa5充填フレームの各field編集mask照合、全切詰め、余剰byteを検証する。wire幅とGo型幅が等しい WatchData/DrivePower/Steps/歩行者Reserved と8bit数値群は Go型上で overflow 値を構築できないため範囲超過caseの対象外。

`TestFrameSemanticBoundaries` は level0/1/5/6/7、delay29/30/31、AssistType2/3、BicycleType7/8、AssistState3、Pedaling2/3、Collision15のTBD、gear0/1/最大前/最大、数値8bit0/1/253/254/255、RearLight/DU/Maintenance2/3、Item0/1/2/3/62/63、Steps65533/65534/65535、Motion3、両reservedを検証する。意味検証は値を書き換えない。WatchData は実験独自で全32bitを受理し、固有の意味判定をしない。

Motion0/1/2は毎分<20/20..139/>=140を表すが、歩数からの自動算出はしない。Cadence の1rpmは整数単位の記述で、変換helperは追加しない。歩行者reserved非zeroのIssueは library policy で、明文の送信0要求とは扱わない。

`TestPayloadErrorOffsets` は Basic bit10 (payload byte6)、Extended bit36 (byte12)、Pedestrian bit22 (byte7) の range error を確認する。`TestFrameNilReceivers` は全6型の nil receiver を確認する。

`TestCategoricalCodes` は Level、AssistType、BicycleType、AssistState/Pedaling、RearLight/DU/Maintenance、ItemInfo、Motion の wire範囲内全コードを分類する。`TestPedestrianPayloadUnavailableCodes` は Level7/Delay31/Item63/Steps65535/Motion3 が complete app `ff00000000ffffff0000` へ伝播し Issue がないことを検証する。
