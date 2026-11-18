<div align="center">

# flaggroup

**Secciones de `--help` agrupadas para `spf13/pflag`** — despídete de las interminables listas de flags ordenadas alfabéticamente.

[![Go Reference](https://pkg.go.dev/badge/github.com/fishfinal/flaggroup.svg)](https://pkg.go.dev/github.com/fishfinal/flaggroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/fishfinal/flaggroup)](https://goreportcard.com/report/github.com/fishfinal/flaggroup)
[![License](https://img.shields.io/github/license/go-defender/flaggroup)](https://github.com/fishfinal/flaggroup/blob/master/LICENSE)
[![CI](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml/badge.svg)](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml)

[English](README.md) | [简体中文](README_zh.md) | [日本語](README_ja.md) | **Español** | [Deutsch](README_de.md)

</div>

Añade **flags agrupados** a `github.com/spf13/pflag`: la salida de `--help` de un comando se renderiza en secciones funcionales en lugar de un muro de flags ordenados alfabéticamente.

pflag reserva un campo `Annotations map[string][]string` en la estructura `Flag`, pero nunca lo utiliza por sí mismo. `flaggroup` es ese consumidor: almacena la información de agrupación en las anotaciones y cede `FlagSet.Usage` a un renderer configurable.

## Instalación

```bash
go get github.com/fishfinal/flaggroup
```

## Inicio rápido

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

`Apply()` vincula la función de renderizado a `fs.Usage`, de modo que a partir de ese momento tanto `--help` como la salida impresa ante errores de parseo se agrupan automáticamente:

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

## Dos estilos de agrupación

### Estilo uno: declarar los miembros explícitamente al renderizar

Es la opción más directa cuando todos los flags están en un único archivo. El orden pasado a `Add` es el orden de visualización, y prevalece sobre el orden lexicográfico por defecto de pflag:

```go
r := flaggroup.New(fs)
r.Group("yara", "YARA Options").Add("yara-dir", "yara-name")
r.Group("output", "Output Options").Add("output", "workers", "verbose")
```

### Estilo dos: anotar los flags en el punto de definición

Cuando los flags están repartidos entre varios paquetes y cada módulo registra los suyos, deja que cada módulo declare a qué grupo pertenecen y limítate a enumerar los grupos al renderizar:

```go
// Dentro de la función de registro de flags de cada módulo
flaggroup.MustSetGroup(fs, "yara", "yara-dir", "yara-name")

// Al renderizar
r.Group("yara", "YARA Options")   // sin miembros: se recopilan automáticamente desde las anotaciones
```

> No llames directamente a `fs.SetAnnotation`: la implementación de pflag retorna en silencio cuando el nombre de un flag está mal escrito, tragándose el error por completo. `SetGroup` valida primero que cada flag exista y devuelve un error.

## Integración con Cobra

El `Group` de Cobra agrupa **subcomandos**, no flags. Para agrupar también el `--help` de un subcomando, engancha el renderer mediante `SetUsageFunc`:

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

Si necesitas conservar la lista nativa de comandos de Cobra, la sección de ejemplos y otros bloques, hazte cargo solo de la parte de flags:

```go
rootCmd.SetUsageTemplate(rootCmd.UsageTemplate() + "")  // conserva la plantilla
rootCmd.SetFlagErrorFunc(...)                            // gestiona el flujo de error por separado
```

El enfoque más seguro es dejar intactas las plantillas de Cobra y sobrescribir `SetUsageFunc` únicamente en los comandos hoja que necesiten agrupación.

## Resumen de la API

| Función / método | Descripción |
| --- | --- |
| `New(fs, ...Option)` | Crea un renderer; escribe en `os.Stderr` con ajuste automático del ancho de columna por defecto |
| `(*Renderer) Group(name, title, members...)` | Declara una sección y devuelve un `*Group` para encadenar `Add` |
| `(*Group) Add(names...)` | Añade miembros al grupo; el orden es el de visualización |
| `(*Renderer) Ungrouped(title)` | Declara la sección de reserva que recoge los flags sin grupo |
| `(*Renderer) Render() error` | Renderiza y devuelve cualquier error de escritura |
| `(*Renderer) Apply()` | Vincula la salida a `fs.Usage` |
| `(*Renderer) UsageFunc() func()` | Devuelve una función que puede asignarse directamente a `fs.Usage` |
| `SetGroup / MustSetGroup` | Escribe anotaciones de forma segura y reporta nombres de flag mal escritos |
| `GroupsOf(f)` | Devuelve todos los grupos a los que pertenece un flag |

| Opción | Descripción |
| --- | --- |
| `WithWriter(w)` | Establece el destino de salida (por defecto `os.Stderr`) |
| `WithColumnWidth(n)` | Fija el ancho de la columna de nombres de flag y desactiva el ajuste automático |
| `WithHeading(text)` | Personaliza la línea de uso superior |
| `WithFooter(text)` | Añade una nota al pie |
| `WithStrictValidation()` | Valida antes de renderizar que los nombres de flag existan y no estén reclamados dos veces |

## Reglas de renderizado

- **Forma corta**: cuando `Shorthand` está definido, el flag se renderiza como `-v, --verbose`
- **Indicador de tipo**: los tipos distintos de `bool` añaden su tipo al nombre, como en `--output string`
- **Valores por defecto**: las cadenas se entrecomillan con `%q`; un `bool` en `false`, una cadena vacía y un slice vacío no se muestran
- **Ocultos**: los flags pasados a `MarkHidden` no aparecen en ningún grupo
- **Obsoletos**: se añade `(DEPRECATED: <message>)` al final de la línea
- **Grupos vacíos**: un grupo sin flags visibles se omite junto con su título
- **Ancho de columna**: por defecto, la longitud máxima de todos los nombres de flag visibles, con un mínimo de 24, alineado entre grupos

## Pruebas

```bash
go test ./...
go run ./example --help
```
