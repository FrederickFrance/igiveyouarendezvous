# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

[English](README.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · **Italiano** · [ไทย](README.th.md) · [tlhIngan Hol](README.tlh.md) · [Ch'ti](README.pcd.md)

Pianificazione di colloqui tra candidati e selezionatori, in Go, come comando
o come libreria.

Ogni candidato incontra tutti i suoi selezionatori lo stesso giorno, in
colloqui che si susseguono. Il calendario massimizza prima i candidati che
incontrano tutti i loro selezionatori (per priorità), poi gli incontri, e
infine risparmia il tempo dei selezionatori. Un'esecuzione pianifica una
giornata; ciò che non ci sta finisce nel file del problema del giorno dopo.

## Il primo calendario, passo dopo passo

Senza saper programmare né usare la riga di comando: seguite esattamente
questi passi, nell'ordine.

**1. Prendere il programma.** Aprite la pagina [Releases](https://github.com/FrederickFrance/igiveyouarendezvous/releases/latest) e scaricate
lo zip del vostro computer:

| Il vostro computer | Il file |
|---|---|
| Windows | `igiveyouarendezvous-windows.zip` |
| Mac con chip Apple (M1, M2…: menu Apple > Informazioni su questo Mac) | `igiveyouarendezvous-mac-apple.zip` |
| Mac con processore Intel | `igiveyouarendezvous-mac-intel.zip` |
| Linux | `igiveyouarendezvous-linux.zip` |

Estraetelo (sul Mac con un doppio clic, su Windows clic destro > Estrai
tutto). Otterrete una cartella con il programma e il file `problem.yaml`.

**2. Descrivere la giornata.** Aprite `problem.yaml` con un semplice editor
di testo: su Windows clic destro > Apri con > Blocco note; sul Mac clic
destro > Apri con > TextEdit. Mettete i nomi dei vostri selezionatori e
candidati al posto di quelli dell'esempio; sotto ogni candidato, `meet:`
elenca i selezionatori da incontrare, tra parentesi quadre e separati da
virgole. Tre regole: lasciate gli spazi a inizio riga esattamente come sono
(due spazi, mai il tasto Tab); scrivete ogni nome sempre allo stesso modo;
non togliete i due punti. Salvate.

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

**3. Aprire un terminale nella cartella.**
- Windows: aprite la cartella, cliccate nella barra degli indirizzi in alto,
  scrivete `cmd`, premete Invio. Si apre una finestra nera.
- Mac: aprite l'app Terminale (Applicazioni > Utility), scrivete `cd ` (con
  uno spazio), trascinate la cartella nella finestra, premete Invio.
- Linux: clic destro nella cartella > Apri nel terminale.

**4. Avviare.** Scrivete questa riga, poi premete Invio:
- Windows: `igiveyouarendezvous problem.yaml`
- Mac e Linux: `./igiveyouarendezvous problem.yaml`

Entro un minuto compare il calendario: chi incontra chi, e a che ora. Sul
Mac, la prima volta, il sistema blocca il programma che non conosce: aprite
Impostazioni di Sistema > Privacy e sicurezza, cliccate «Apri comunque» in
basso e riscrivete la riga. (Anche Windows può avvisare: Ulteriori
informazioni > Esegui comunque.)

**5. Conservare il risultato.** Per averlo in un file invece che nella
finestra: `igiveyouarendezvous problem.yaml > calendario.txt` (Mac e Linux:
con `./` davanti), poi aprite `calendario.txt` come al passo 2.

**6. Il giorno dopo.** Per chi non ha trovato posto oggi:
`igiveyouarendezvous -rest domani.yaml problem.yaml` (Mac e Linux: con `./`
davanti). Il programma scrive anche `domani.yaml`: lo stesso file, con solo
gli incontri rimasti. Completatelo se serve e usatelo il giorno dopo al posto
di `problem.yaml`.

**In caso di errore**, il programma indica la riga sbagliata del file:
controllate gli spazi a inizio riga, i due punti e che i nomi siano scritti
come sotto `recruiters`. L'esempio completo e commentato, `example-full.yaml`,
mostra tutto il resto: disponibilità per persona e per data, priorità,
colloqui comuni…

## Installazione

Go 1.27.3 o successivo, senza dipendenze C:

```sh
go install github.com/FrederickFrance/igiveyouarendezvous/cmd/igiveyouarendezvous@latest
```

## Uso

```sh
igiveyouarendezvous -date 2026-11-03 -rest giorno2.yaml example/problem.yaml
```

Il file del problema (YAML o JSON) descrive i selezionatori, i candidati con i
selezionatori da incontrare, le disponibilità e le impostazioni: vedere
l'[esempio commentato](example/problem.yaml) e lo
[schema JSON](schema/problem.schema.json). Ctrl+C ferma la ricerca e stampa
il miglior calendario trovato finora.

La documentazione completa è in [inglese](README.md) e in
[francese](README.fr.md).

## Licenza

A scelta, [0BSD](LICENSE) o [WTFPL](WTFPL.txt), versione 2.
