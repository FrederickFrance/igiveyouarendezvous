# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

[English](README.md) · [Français](README.fr.md) · [Deutsch](README.de.md) · [Italiano](README.it.md) · [ไทย](README.th.md) · [tlhIngan Hol](README.tlh.md) · **Ch'ti**

Ch'est un programme pour planifier ches intertiens intre ches candidats et
ches recruteus, in Go.

Chaque candidat i vot tous ses recruteus dins l'période (eune journée, eune
semaine…), avec un écart minimum et maximum intre deux intertiens (par
défaut eun après l'eute, ou bin par exempe au moins trois jours), et si o
veut dins un ordre. Ch'programme i fait passer l'pus d'gins possibe, pis i
fait des économies d'temps pour ches recruteus. Eune période à l'fos (d'`from`
à `to`, du lundi au vindredi par défaut, sans ches jours fériés) : cho qui
n'rinte pon, i l'met dins ch'fichier d'la période d'après.

Pas qu'des intertiens : avec `together: true`, un « candidat » i vot tous ses
« recruteus » in même temps, ou bin personne, pour inregistrer eune émission,
pour eune partie d'jeu d'rôle… Et un colloque dins eune salle, ch'est un
« candidat » qu'ses intervintions i s'suivtent. Vos ches exempes
[émission](example/fr/show.yaml), [colloque](example/fr/colloquium.yaml) et [jeu
d'rôle](example/fr/roleplay.yaml).

**[Essayer dins ch'navigateur](https://frederickfrance.github.io/igiveyouarendezvous/)**, sans rin installer : tout i s'calcule
su vot' ordinateur, rin n'est invoyé.

## Ch'premier planning, pas à pas

Pas besoin d'ête informaticien : fais just' cho qu'i est marqué, dins l'ordre.

**1. Prinds ch'programme.** Va su' l'page [Releases](https://github.com/FrederickFrance/igiveyouarendezvous/releases/latest) et
télécharge ch'zip d'tin ordinateur :

| T'ordinateur | Ch'fichier |
|---|---|
| Windows | `igiveyouarendezvous-windows.zip` |
| Mac avec eune puce Apple (M1, M2… : menu Pomme > À propos de ce Mac) | `igiveyouarendezvous-mac-apple.zip` |
| Mac Intel | `igiveyouarendezvous-mac-intel.zip` |
| Linux | `igiveyouarendezvous-linux.zip` |

Dézippe-le (su' Mac, double-clic ; su' Windows, clic droit > Extraire tout).
T'as un dossier avec ch'programme et `problem.yaml`.

**2. Raconte et'journée.** Ouvre `problem.yaml` avec ch'Bloc-notes
(Windows : clic droit > Ouvrir avec > Bloc-notes ; Mac : clic droit > Ouvrir
avec > TextEdit). Mets les noms d'tes recruteus et d'tes candidats à l'plache
d'cheux d'l'exempe ; sous chaque candidat, `meet:` ch'est les recruteus qu'i
doit vir, intre crochets, séparés par des virgules. Trois règles : n'touche
point aux espaces au début des lignes (deux espaces, jamais l'touche Tab),
écris chaque nom pareil partout, laisse les deux-points. Inregistre.

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

**3. Ouvre un terminal dins ch'dossier.**
- Windows : ouvre ch'dossier, clique dins l'barre d'adresse in haut, tape
  `cmd`, Entrée. Eune fenête noire all' s'ouvre.
- Mac : ouvre Terminal (Applications > Utilitaires), tape `cd ` (avec un
  espace), glisse ch'dossier dins l'fenête, Entrée.
- Linux : clic droit dins ch'dossier > Ouvrir dans un terminal.

**4. Lance-le.** Tape cha, pis Entrée :
- Windows : `igiveyouarendezvous problem.yaml`
- Mac et Linux : `./igiveyouarendezvous problem.yaml`

Dins moins d'eune minute, t'as ch'planning : qui qu'i vot qui, et à quelle
heure. Su' Mac, la première fos, ch'système i bloque ch'programme qu'i
n'connot point : Réglages Système > Confidentialité et sécurité, « Ouvrir
quand même », pis tape la ligne eud'nouveau. (Windows i peut aussi rouspéter :
Informations complémentaires > Exécuter quand même.)

**5. Garde ch'résultat.** `igiveyouarendezvous problem.yaml > planning.txt`
(Mac et Linux : avec `./` devant), pis ouvre `planning.txt` comme au 2.

**6. Éch'lindemain.** Pour cheux qui n'ont point pu passer :
`igiveyouarendezvous -rest lindemain.yaml problem.yaml` (Mac et Linux : avec
`./` devant). Ch'programme i écrit aussi `lindemain.yaml`, avec seulement cho
qui reste : prinds-le éch'lindemain à l'plache d'`problem.yaml`. (Pour eune
semaine intière d'un coup, vos `from` et `to` dins `example-full.yaml`.)

**Si cha n'va point**, ch'programme i dit quelle ligne qu'all' est fausse :
r'garde les espaces au début, ch'deux-points, et si les noms is sont écrits
comme sous `recruiters`. `example-full.yaml` i montre tout ch'reste.

## Pour l'installer

```sh
go install github.com/FrederickFrance/igiveyouarendezvous/cmd/igiveyouarendezvous@latest
```

## Pour s'in servir

```sh
igiveyouarendezvous -rest lindemain.yaml example/problem.yaml
```

Ch'fichier problème (YAML ou JSON) i dit qui qu'i doit vir qui, et quand
ch'est qu'ches gins is sont libes : [vlà un exempe](example/problem.yaml).
Ctrl+C : i s'arrête, pis i t'donne ch'meilleur planning qu'il a trouvé.

Tout ch'mode d'emploi, i est in [inglais](README.md) pis in
[français](README.fr.md).

## Licince

[0BSD](LICENSE) ou [WTFPL](WTFPL.txt). Fais-in cho qu'te veux, biloute.
