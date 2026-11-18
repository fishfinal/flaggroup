<div align="center">

# flaggroup

**为 `spf13/pflag` 提供分组的 `--help` 输出** —— 告别按字母排序、糊成一堆冗长参数列表。

[![Go Reference](https://pkg.go.dev/badge/github.com/fishfinal/flaggroup.svg)](https://pkg.go.dev/github.com/fishfinal/flaggroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/fishfinal/flaggroup)](https://goreportcard.com/report/github.com/fishfinal/flaggroup)
[![License](https://img.shields.io/github/license/go-defender/flaggroup)](https://github.com/fishfinal/flaggroup/blob/master/LICENSE)
[![CI](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml/badge.svg)](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml)

[English](README.md) | **简体中文** | [日本語](README_ja.md) | [Español](README_es.md) | [Deutsch](README_de.md)

</div>

给 `github.com/spf13/pflag` 补上**选项分组**能力：让单个命令的 `--help` 按功能分区展示，而不是一面按字母序排列的选项墙。

pflag 在 `Flag` 结构体里预留了 `Annotations map[string][]string` 字段，但自身从不消费它。`flaggroup` 就是那个消费方——把分组信息写在 annotation 里，再用一个可配置的渲染器接管 `FlagSet.Usage`。

## 安装

```bash
go get github.com/fishfinal/flaggroup
```

## 快速开始

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

`Apply()` 会把渲染函数绑定到 `fs.Usage`，此后 `--help` 与解析报错时的输出都会自动分组：

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

## 两种分组写法

### 写法一：渲染处显式声明成员

选项集中在一个文件里时最直观，`Add` 的传入顺序就是展示顺序（可覆盖 pflag 默认的字典序）：

```go
r := flaggroup.New(fs)
r.Group("yara", "YARA Options").Add("yara-dir", "yara-name")
r.Group("output", "Output Options").Add("output", "workers", "verbose")
```

### 写法二：定义处打 annotation

选项分散在多个包、由不同模块各自注册时，让每个模块自己声明归属，渲染处只列分组：

```go
// 各模块的 flag 注册函数里
flaggroup.MustSetGroup(fs, "yara", "yara-dir", "yara-name")

// 渲染处
r.Group("yara", "YARA Options")   // 不带成员，自动按 annotation 收集
```

> 不要直接用 `fs.SetAnnotation`：pflag 的实现在选项名拼错时会静默返回，错误被完全吞掉。`SetGroup` 会先校验存在性并返回 error。

## 与 Cobra 集成

Cobra 的 `Group` 分的是**子命令**，不是选项。要让子命令的 `--help` 也分组，把渲染器挂到 `SetUsageFunc`：

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

需要保留 Cobra 原生的命令列表、Example 等区块时，改为只接管 flags 部分：

```go
rootCmd.SetUsageTemplate(rootCmd.UsageTemplate() + "")  // 保持模板
rootCmd.SetFlagErrorFunc(...)                            // 错误路径单独处理
```

更稳妥的做法是不动 Cobra 模板，只在需要分组的叶子命令上覆盖 `SetUsageFunc`。

## API 一览

| 函数 / 方法 | 说明 |
| --- | --- |
| `New(fs, ...Option)` | 创建渲染器，默认输出到 `os.Stderr`、列宽自适应 |
| `(*Renderer) Group(name, title, members...)` | 声明一个分区，返回 `*Group` 以便 `Add` |
| `(*Group) Add(names...)` | 追加分组成员，顺序即展示顺序 |
| `(*Renderer) Ungrouped(title)` | 声明兜底分区，收留未分组选项 |
| `(*Renderer) Render() error` | 渲染并返回写入错误 |
| `(*Renderer) Apply()` | 绑定到 `fs.Usage` |
| `(*Renderer) UsageFunc() func()` | 返回可直接赋给 `fs.Usage` 的函数 |
| `SetGroup / MustSetGroup` | 安全地写 annotation，选项名拼错会报错 |
| `GroupsOf(f)` | 读取某选项所属的全部分组 |

| Option | 说明 |
| --- | --- |
| `WithWriter(w)` | 指定输出目标（默认 `os.Stderr`） |
| `WithColumnWidth(n)` | 固定选项名列宽，关闭自适应 |
| `WithHeading(text)` | 自定义顶部用法行 |
| `WithFooter(text)` | 追加页脚说明 |
| `WithStrictValidation()` | 渲染前校验选项名存在且不重复归组 |

## 渲染规则

- **短选项**：`Shorthand` 存在时渲染为 `-v, --verbose`
- **类型提示**：非 `bool` 类型在名称后追加类型，如 `--output string`
- **默认值**：字符串用 `%q` 加引号；`bool` 的 `false`、空字符串、空切片不展示
- **Hidden**：`MarkHidden` 的选项不出现在任何分组
- **Deprecated**：行尾追加 `(DEPRECATED: <提示>)`
- **空分组**：整组无可见选项时连标题一起跳过
- **列宽**：默认取所有可见选项名的最大长度（下限 24），跨分组对齐

## 测试

```bash
go test ./...
go run ./example --help
```
