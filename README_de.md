<div align="center">

# flaggroup

**Gruppierte `--help`-Abschnitte für `spf13/pflag`** — Schluss mit alphabetisch sortierten, endlosen Flag-Listen.

[![Go Reference](https://pkg.go.dev/badge/github.com/fishfinal/flaggroup.svg)](https://pkg.go.dev/github.com/fishfinal/flaggroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/fishfinal/flaggroup)](https://goreportcard.com/report/github.com/fishfinal/flaggroup)
[![License](https://img.shields.io/github/license/go-defender/flaggroup)](https://github.com/fishfinal/flaggroup/blob/main/LICENSE)
[![CI](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml/badge.svg)](https://github.com/fishfinal/flaggroup/actions/workflows/test.yml)

[English](README.md) | [简体中文](README_zh.md) | [日本語](README_ja.md) | [Español](README_es.md) | **Deutsch**

</div>

Fügt `github.com/spf13/pflag` **gruppierte Flags** hinzu: Die `--help`-Ausgabe eines einzelnen Befehls wird in funktionalen Abschnitten dargestellt statt als alphabetisch sortierte Wand aus Flags.

pflag reserviert im Struct `Flag` ein Feld `Annotations map[string][]string`, nutzt es aber selbst nie. `flaggroup` ist der Konsument dieses Feldes: Die Gruppierungsinformationen werden in den Annotationen gespeichert und `FlagSet.Usage` an einen konfigurierbaren Renderer übergeben.

## Installation

```bash
go get github.com/fishfinal/flaggroup
```

## Schnellstart

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

`Apply()` bindet die Render-Funktion an `fs.Usage`. Ab diesem Zeitpunkt werden sowohl `--help` als auch die Ausgabe bei Parse-Fehlern automatisch gruppiert:

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

## Zwei Gruppierungsstile

### Stil eins: Mitglieder explizit beim Rendern deklarieren

Das ist die geradlinigste Variante, wenn alle Flags in einer einzigen Datei liegen. Die an `Add` übergebene Reihenfolge ist die Anzeigereihenfolge und überschreibt die standardmäßige lexikografische Sortierung von pflag:

```go
r := flaggroup.New(fs)
r.Group("yara", "YARA Options").Add("yara-dir", "yara-name")
r.Group("output", "Output Options").Add("output", "workers", "verbose")
```

### Stil zwei: Flags an der Definitionsstelle annotieren

Wenn die Flags über mehrere Pakete verteilt sind und jedes Modul seine eigenen registriert, lässt man jedes Modul deklarieren, zu welcher Gruppe seine Flags gehören, und listet beim Rendern nur noch die Gruppen auf:

```go
// Innerhalb der Flag-Registrierungsfunktion des jeweiligen Moduls
flaggroup.MustSetGroup(fs, "yara", "yara-dir", "yara-name")

// Beim Rendern
r.Group("yara", "YARA Options")   // keine Mitglieder: werden automatisch aus den Annotationen gesammelt
```

> Rufe `fs.SetAnnotation` nicht direkt auf: Die pflag-Implementierung kehrt bei einem falsch geschriebenen Flag-Namen stillschweigend zurück und verschluckt den Fehler vollständig. `SetGroup` prüft zunächst, ob jedes Flag existiert, und gibt einen Fehler zurück.

## Cobra-Integration

Cobras `Group` gruppiert **Subcommands**, nicht Flags. Um auch die `--help`-Ausgabe eines Subcommands zu gruppieren, hängt man den Renderer über `SetUsageFunc` ein:

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

Wenn die native Befehlsliste von Cobra, der Example-Abschnitt und andere Blöcke erhalten bleiben sollen, übernimmt man stattdessen nur den Flag-Teil:

```go
rootCmd.SetUsageTemplate(rootCmd.UsageTemplate() + "")  // Template beibehalten
rootCmd.SetFlagErrorFunc(...)                            // Fehlerpfad separat behandeln
```

Der sicherere Weg ist, Cobras Templates unangetastet zu lassen und `SetUsageFunc` nur bei den Leaf-Commands zu überschreiben, die eine Gruppierung benötigen.

## API-Übersicht

| Funktion / Methode | Beschreibung |
| --- | --- |
| `New(fs, ...Option)` | Erzeugt einen Renderer; schreibt standardmäßig nach `os.Stderr` mit automatischer Spaltenbreite |
| `(*Renderer) Group(name, title, members...)` | Deklariert einen Abschnitt und gibt einen `*Group` zurück, sodass `Add` verkettet werden kann |
| `(*Group) Add(names...)` | Fügt Gruppenmitglieder hinzu; die Reihenfolge ist die Anzeigereihenfolge |
| `(*Renderer) Ungrouped(title)` | Deklariert den Auffangabschnitt für nicht gruppierte Flags |
| `(*Renderer) Render() error` | Rendert und gibt etwaige Schreibfehler zurück |
| `(*Renderer) Apply()` | Bindet die Ausgabe an `fs.Usage` |
| `(*Renderer) UsageFunc() func()` | Gibt eine Funktion zurück, die direkt `fs.Usage` zugewiesen werden kann |
| `SetGroup / MustSetGroup` | Schreibt Annotationen sicher und meldet falsch geschriebene Flag-Namen |
| `GroupsOf(f)` | Gibt alle Gruppen zurück, zu denen ein Flag gehört |

| Option | Beschreibung |
| --- | --- |
| `WithWriter(w)` | Legt das Ausgabeziel fest (standardmäßig `os.Stderr`) |
| `WithColumnWidth(n)` | Fixiert die Spaltenbreite der Flag-Namen und schaltet die automatische Anpassung ab |
| `WithHeading(text)` | Passt die Usage-Zeile oben an |
| `WithFooter(text)` | Hängt eine Fußnote an |
| `WithStrictValidation()` | Prüft vor dem Rendern, ob Flag-Namen existieren und nicht doppelt beansprucht werden |

## Rendering-Regeln

- **Kurzform**: Ist `Shorthand` gesetzt, wird das Flag als `-v, --verbose` dargestellt
- **Typangabe**: Bei Typen außer `bool` wird der Typ an den Namen angehängt, etwa `--output string`
- **Default-Werte**: Strings werden mit `%q` gequotet; ein `false`-Bool, ein leerer String und ein leeres Slice werden nicht angezeigt
- **Versteckt**: Flags, die an `MarkHidden` übergeben wurden, erscheinen in keiner Gruppe
- **Veraltet**: `(DEPRECATED: <message>)` wird am Zeilenende angehängt
- **Leere Gruppen**: Eine Gruppe ohne sichtbare Flags wird samt Titel übersprungen
- **Spaltenbreite**: Standardmäßig die maximale Länge aller sichtbaren Flag-Namen, mindestens 24, gruppenübergreifend ausgerichtet

## Tests

```bash
go test ./...
go run ./example --help
```
