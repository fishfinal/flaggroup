<div align="center">

# flaggroup

**`spf13/pflag` に `--help` のグループ表示を追加** — アルファベット順に並んだだけの長大なフラグ一覧とはおさらば。

[![Go Reference](https://pkg.go.dev/badge/github.com/fishfinal/flaggroup.svg)](https://pkg.go.dev/github.com/fishfinal/flaggroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/fishfinal/flaggroup)](https://goreportcard.com/report/github.com/fishfinal/flaggroup)
[![License](https://img.shields.io/github/license/go-defender/flaggroup)](https://github.com/fishfinal/flaggroup/blob/main/LICENSE)
[![CI](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml/badge.svg)](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml)

[English](README.md) | [简体中文](README_zh.md) | **日本語** | [Español](README_es.md) | [Deutsch](README_de.md)

</div>

`github.com/spf13/pflag` に**フラグのグループ化**を追加します。単一コマンドの `--help` が、アルファベット順に並んだフラグの壁ではなく、機能ごとのセクションに分けて表示されるようになります。

pflag は `Flag` 構造体に `Annotations map[string][]string` フィールドを用意していますが、自身では決して利用しません。`flaggroup` はその利用者です。グループ情報をアノテーションに格納し、`FlagSet.Usage` を設定可能なレンダラーに引き渡します。

## インストール

```bash
go get github.com/fishfinal/flaggroup
```

## クイックスタート

```go
package main

import (
  flag "github.com/spf13/pflag"
  "github.com/fishfinal/flaggroup"
)

func main() {
  fs := flag.NewFlagSet("myapp", flag.ExitOnError)
  input := fs.String("input", "", "Input file path")
  output := fs.String("output", "out.txt", "Output file")
  verbose := fs.BoolP("verbose", "v", false, "Enable verbose mode")

  flaggroup.New(fs).
    Group("input", "Input Options", "input").
    Group("output", "Output Options").Add("output", "verbose").
    Ungrouped("Other Options").
    Apply()

  fs.Parse(nil)
  _, _, _ = input, output, verbose
}
```

`Apply()` はレンダー関数を `fs.Usage` にバインドします。以降は `--help` でもパースエラー時に出力される内容でも、自動的にグループ化されます。

```
Usage: myapp [options]

Input Options:
  --input string       Input file path

Output Options:
  --output string      Output file (default: "out.txt")
  -v, --verbose        Enable verbose mode

Other Options:
  --help               help for myapp
```

## 2 つのグループ化スタイル

### スタイル 1: レンダー時にメンバーを明示的に宣言する

すべてのフラグが 1 つのファイルに収まっている場合に最も素直な方法です。`Add` に渡した順序がそのまま表示順となり、pflag のデフォルトの辞書順を上書きします。

```go
r := flaggroup.New(fs)
r.Group("yara", "YARA Options").Add("yara-dir", "yara-name")
r.Group("output", "Output Options").Add("output", "workers", "verbose")
```

### スタイル 2: 定義箇所でフラグにアノテーションを付ける

フラグが複数のパッケージに分散し、各モジュールが自身のフラグを登録する場合は、各モジュールに所属グループを宣言させ、レンダー時にはグループだけを列挙します。

```go
// 各モジュールのフラグ登録関数内
flaggroup.MustSetGroup(fs, "yara", "yara-dir", "yara-name")

// レンダー時
r.Group("yara", "YARA Options")   // メンバー指定なし: アノテーションから自動収集
```

> `fs.SetAnnotation` を直接呼ばないでください。pflag の実装はフラグ名の綴りが誤っていても黙って戻るため、エラーが完全に握りつぶされます。`SetGroup` はまず各フラグの存在を検証し、エラーを返します。

## Cobra との統合

Cobra の `Group` は**サブコマンド**をグループ化するもので、フラグではありません。サブコマンドの `--help` もグループ化するには、`SetUsageFunc` 経由でレンダラーを差し込みます。

```go
rootCmd.SetUsageFunc(func(c *cobra.Command) error {
  return flaggroup.New(c.Flags(),
    flaggroup.WithHeading(fmt.Sprintf("Usage: %s [flags] [args]", c.CommandPath())),
  ).
    Group("main", "Main Options", "yara-dir", "output").
    Ungrouped("Other Options").
    Render()
})
```

Cobra 標準のコマンド一覧、Example セクションなどのブロックを維持したまま、フラグ部分だけを引き受けたい場合:

```go
rootCmd.SetUsageTemplate(rootCmd.UsageTemplate() + "")  // テンプレートを維持
rootCmd.SetFlagErrorFunc(...)                            // エラー経路は別途処理
```

より安全なのは、Cobra のテンプレートには手を付けず、グループ化が必要な末端コマンドに対してのみ `SetUsageFunc` を上書きする方法です。

## API 一覧

| 関数 / メソッド | 説明 |
| --- | --- |
| `New(fs, ...Option)` | レンダラーを生成。デフォルトで `os.Stderr` に書き込み、桁幅を自動調整 |
| `(*Renderer) Group(name, title, members...)` | セクションを宣言し `*Group` を返すため `Add` をチェーン可能 |
| `(*Group) Add(names...)` | グループのメンバーを追加。順序がそのまま表示順 |
| `(*Renderer) Ungrouped(title)` | 未分類フラグを受け止めるフォールバックセクションを宣言 |
| `(*Renderer) Render() error` | レンダリングを行い、書き込みエラーがあれば返す |
| `(*Renderer) Apply()` | 出力を `fs.Usage` にバインド |
| `(*Renderer) UsageFunc() func()` | `fs.Usage` に直接代入できる関数を返す |
| `SetGroup / MustSetGroup` | アノテーションを安全に書き込み、フラグ名の綴り誤りを通知 |
| `GroupsOf(f)` | フラグが属するすべてのグループを返す |

| オプション | 説明 |
| --- | --- |
| `WithWriter(w)` | 出力先を設定（デフォルトは `os.Stderr`） |
| `WithColumnWidth(n)` | フラグ名列の幅を固定し、自動調整を無効化 |
| `WithHeading(text)` | 先頭の usage 行をカスタマイズ |
| `WithFooter(text)` | フッターの注記を追加 |
| `WithStrictValidation()` | レンダリング前にフラグ名の存在と二重指定の有無を検証 |

## レンダリング規則

- **短縮形**: `Shorthand` が設定されている場合、`-v, --verbose` として表示
- **型ヒント**: `bool` 以外の型は名前に付加され、`--output string` のように表示
- **デフォルト値**: 文字列は `%q` でクォート。`false` の bool、空文字列、空スライスは表示しない
- **非表示**: `MarkHidden` に渡したフラグはどのグループにも現れない
- **非推奨**: 行末に `(DEPRECATED: <message>)` を付加
- **空グループ**: 表示可能なフラグを持たないグループは、タイトルごとスキップ
- **桁幅**: デフォルトでは表示可能な全フラグ名の最大長（下限 24）で、グループ間で揃えて整列

## テスト

```bash
go test ./...
go run ./example --help
```
