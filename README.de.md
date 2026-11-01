# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

[English](README.md) · [Français](README.fr.md) · **Deutsch** · [Italiano](README.it.md) · [ไทย](README.th.md) · [tlhIngan Hol](README.tlh.md) · [Ch'ti](README.pcd.md)

Terminplanung für Vorstellungsgespräche zwischen Bewerbern und Recruitern,
in Go, als Kommandozeilenprogramm oder als Bibliothek.

Jeder Bewerber trifft alle seine Recruiter am selben Tag, in Gesprächen, die
direkt aufeinander folgen. Der Plan maximiert zuerst die Zahl der Bewerber,
die alle ihre Recruiter treffen (nach Priorität), dann die Zahl der Treffen,
und schont danach die Zeit der Recruiter. Ein Durchlauf plant einen Tag; was
nicht hineinpasst, landet in der Problemdatei des nächsten Tages.

## Der erste Plan, Schritt für Schritt

Ohne Programmier- oder Kommandozeilenkenntnisse: Folgen Sie genau diesen
Schritten, der Reihe nach.

**1. Das Programm holen.** Öffnen Sie die Seite [Releases](https://github.com/FrederickFrance/igiveyouarendezvous/releases/latest) und laden
Sie die ZIP-Datei für Ihren Computer herunter:

| Ihr Computer | Die Datei |
|---|---|
| Windows | `igiveyouarendezvous-windows.zip` |
| Mac mit Apple-Chip (M1, M2 …: Apple-Menü > Über diesen Mac) | `igiveyouarendezvous-mac-apple.zip` |
| Mac mit Intel-Prozessor | `igiveyouarendezvous-mac-intel.zip` |
| Linux | `igiveyouarendezvous-linux.zip` |

Entpacken Sie sie (am Mac per Doppelklick, unter Windows Rechtsklick > Alle
extrahieren). Sie erhalten einen Ordner mit dem Programm und der Datei
`problem.yaml`.

**2. Den Tag beschreiben.** Öffnen Sie `problem.yaml` mit einem einfachen
Texteditor: unter Windows Rechtsklick > Öffnen mit > Editor, am Mac
Rechtsklick > Öffnen mit > TextEdit. Setzen Sie die Namen Ihrer Recruiter und
Bewerber an die Stelle der Beispielnamen; unter jedem Bewerber listet
`meet:` die Recruiter, die er treffen muss, in eckigen Klammern und durch
Kommas getrennt. Drei Regeln: die Leerzeichen am Zeilenanfang genau so lassen
(zwei Leerzeichen, nie die Tabulatortaste), jeden Namen überall gleich
schreiben, die Doppelpunkte stehen lassen. Speichern.

```yaml
day: 09:00-17:00
breaks: 12:00-13:30
duration: 1h

recruiters:
  Alice:
  Bob:

candidates:
  David:
    meet: [Alice, Bob]
  Emma:
    meet: [Bob]
```

**3. Ein Terminal im Ordner öffnen.**
- Windows: den Ordner öffnen, oben in die Adressleiste klicken, `cmd`
  eingeben, Eingabetaste drücken. Ein schwarzes Fenster öffnet sich.
- Mac: die App Terminal öffnen (Programme > Dienstprogramme), `cd ` eingeben
  (mit Leerzeichen), den Ordner ins Fenster ziehen, Eingabetaste drücken.
- Linux: Rechtsklick im Ordner > Im Terminal öffnen.

**4. Starten.** Diese Zeile eingeben, dann Eingabetaste:
- Windows: `igiveyouarendezvous problem.yaml`
- Mac und Linux: `./igiveyouarendezvous problem.yaml`

Nach höchstens einer Minute erscheint der Plan: wer wen um wie viel Uhr
trifft. Am Mac blockiert das System das unbekannte Programm beim ersten Mal:
Systemeinstellungen > Datenschutz & Sicherheit öffnen, unten auf „Dennoch
öffnen“ klicken und die Zeile erneut eingeben. (Auch Windows kann warnen:
Weitere Informationen > Trotzdem ausführen.)

**5. Das Ergebnis behalten.** Um es in einer Datei statt im Fenster zu
haben: `igiveyouarendezvous problem.yaml > plan.txt` eingeben (Mac und Linux:
mit `./` davor) und `plan.txt` wie in Schritt 2 öffnen.

**6. Der nächste Tag.** Für alle, die heute nicht hineingepasst haben:
`igiveyouarendezvous -rest morgen.yaml problem.yaml` eingeben (Mac und Linux:
mit `./` davor). Das Programm schreibt dann auch `morgen.yaml`: dieselbe
Datei, nur mit den übrigen Treffen. Bei Bedarf ergänzen und am nächsten Tag
anstelle von `problem.yaml` verwenden.

**Bei einem Fehler** nennt das Programm die fehlerhafte Zeile der Datei:
Leerzeichen am Zeilenanfang, Doppelpunkt und die Schreibweise der Namen (wie
unter `recruiters`) prüfen. Das vollständige kommentierte Beispiel
`example-full.yaml` zeigt alles Weitere: Verfügbarkeiten pro Person und
Datum, Prioritäten, gemeinsame Gespräche …

## Installation

Go 1.27.3 oder neuer, ohne C-Abhängigkeit:

```sh
go install github.com/FrederickFrance/igiveyouarendezvous/cmd/igiveyouarendezvous@latest
```

## Verwendung

```sh
igiveyouarendezvous -date 2026-11-03 -rest tag2.yaml example/problem.yaml
```

Die Problemdatei (YAML oder JSON) beschreibt die Recruiter, die Bewerber mit
den Recruitern, die sie treffen müssen, die Verfügbarkeiten und die
Einstellungen: siehe das [kommentierte Beispiel](example/problem.yaml) und
das [JSON-Schema](schema/problem.schema.json). Strg+C beendet die Suche und
gibt den besten bisher gefundenen Plan aus.

Die vollständige Dokumentation gibt es auf [Englisch](README.md) und auf
[Französisch](README.fr.md).

## Lizenz

Nach Wahl [0BSD](LICENSE) oder [WTFPL](WTFPL.txt), Version 2.
