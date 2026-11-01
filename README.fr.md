# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

[English](README.md) · **Français** · [Deutsch](README.de.md) · [Italiano](README.it.md) · [ไทย](README.th.md) · [tlhIngan Hol](README.tlh.md) · [Ch'ti](README.pcd.md)

Planification d'entretiens entre candidats et recruteurs, en Go, en ligne de
commande ou comme bibliothèque.

Chaque candidat doit rencontrer certains recruteurs. Les règles :

- un recruteur reçoit un candidat à la fois ; un candidat peut rencontrer
  plusieurs recruteurs en un seul entretien commun quand ils font partie d'un
  même **groupe commun** ;
- un candidat rencontre **tous ses recruteurs le même jour**, ou aucun ce
  jour-là ;
- les entretiens d'un candidat **s'enchaînent**, et ne sont jamais de part et
  d'autre d'une **pause** comme le déjeuner ;
- chacun a des disponibilités, les mêmes chaque jour ou **par date**, et
  chaque recruteur une durée d'entretien ;
- les candidats peuvent avoir une **priorité** : un candidat passe avant
  n'importe quel nombre de candidats de priorité inférieure ;
- entre candidats qui se valent, les **plus urgents** passent d'abord : ceux
  à qui il reste le moins de jours.

Tout est décrit dans un seul **fichier problème**, en YAML ou en JSON. Un
lancement planifie **une journée** ; les rencontres qui ne tiennent pas sont
écrites dans le fichier problème du lendemain.

## Premier planning, pas à pas

Sans rien connaître à la programmation ni à la ligne de commande : suivez ces
étapes exactement, dans l'ordre.

**1. Récupérer le programme.** Ouvrez la page [Releases](https://github.com/FrederickFrance/igiveyouarendezvous/releases/latest) et
téléchargez le zip de votre ordinateur :

| Votre ordinateur | Le fichier |
|---|---|
| Windows | `igiveyouarendezvous-windows.zip` |
| Mac avec une puce Apple (M1, M2… : menu Pomme > À propos de ce Mac) | `igiveyouarendezvous-mac-apple.zip` |
| Mac avec un processeur Intel | `igiveyouarendezvous-mac-intel.zip` |
| Linux | `igiveyouarendezvous-linux.zip` |

Décompressez-le (sur Mac, double-clic ; sur Windows, clic droit > Extraire
tout). Vous obtenez un dossier avec le programme et un fichier nommé
`problem.yaml`.

**2. Décrire votre journée.** Ouvrez `problem.yaml` avec un éditeur de texte
simple : sur Windows, clic droit > Ouvrir avec > Bloc-notes ; sur Mac, clic
droit > Ouvrir avec > TextEdit. Mettez les noms de vos recruteurs et de vos
candidats à la place de ceux de l'exemple ; sous chaque candidat, `meet:`
liste les recruteurs qu'il doit rencontrer, entre crochets et séparés par des
virgules. Trois règles : gardez les espaces en début de ligne exactement
comme ils sont (deux espaces, jamais la touche Tab) ; écrivez chaque nom de
la même façon partout ; laissez les deux-points. Enregistrez.

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

**3. Ouvrir un terminal dans le dossier.**
- Windows : ouvrez le dossier, cliquez dans la barre d'adresse en haut, tapez
  `cmd`, appuyez sur Entrée. Une fenêtre noire s'ouvre.
- Mac : ouvrez l'application Terminal (Applications > Utilitaires), tapez
  `cd ` (avec un espace), faites glisser le dossier dans la fenêtre, appuyez
  sur Entrée.
- Linux : clic droit dans le dossier > Ouvrir dans un terminal.

**4. Lancer le calcul.** Tapez cette ligne, puis appuyez sur Entrée :
- Windows : `igiveyouarendezvous problem.yaml`
- Mac et Linux : `./igiveyouarendezvous problem.yaml`

En moins d'une minute, le planning s'affiche : qui rencontre qui, et à quelle
heure. Sur Mac, la première fois, le système bloque le programme qu'il ne
connaît pas : ouvrez Réglages Système > Confidentialité et sécurité, cliquez
sur « Ouvrir quand même » en bas, puis retapez la ligne. (Windows peut aussi
prévenir : Informations complémentaires > Exécuter quand même.)

**5. Garder le résultat.** Pour l'avoir dans un fichier plutôt qu'à l'écran,
tapez `igiveyouarendezvous problem.yaml > planning.txt` (Mac et Linux : avec
`./` devant), puis ouvrez `planning.txt` comme à l'étape 2.

**6. Le lendemain.** Pour les personnes qui n'ont pas pu passer, tapez
`igiveyouarendezvous -rest demain.yaml problem.yaml` (Mac et Linux : avec
`./` devant). Le programme écrit aussi `demain.yaml` : le même fichier, avec
seulement les rencontres restantes. Complétez-le si besoin, et utilisez-le le
lendemain à la place de `problem.yaml`.

**En cas d'erreur**, le programme indique la ligne du fichier en cause :
vérifiez les espaces en début de ligne, les deux-points, et que les noms sont
écrits comme sous `recruiters`. L'exemple complet et commenté,
`example-full.yaml`, montre tout le reste : disponibilités par personne et
par date, priorités, entretiens communs…

## Installation

Go 1.27.3 ou plus récent, sans dépendance C :

```sh
go install github.com/FrederickFrance/igiveyouarendezvous/cmd/igiveyouarendezvous@latest
```

## Fonctionnement

Une recherche locale en Go pur : le planning s'améliore tant qu'elle tourne.
Elle est optimale sur le premier objectif dans tous les cas de test, et
proche sur les autres (voir [Qualité](#qualité-de-la-recherche-locale)).

Elle part d'un planning glouton, construit en une milliseconde par ajouts
successifs : les candidats de plus haute priorité d'abord et, à priorité
égale, les plus urgents, puis ceux qui ont le moins de rencontres (au hasard
entre égaux), chacun à
la première place trouvée. Puis elle enchaîne des mouvements « détruire et
reconstruire » : elle retire quelques candidats (au hasard, ceux qui
partagent un recruteur, ceux d'une plage horaire, ceux qui pourraient mieux
faire…) et les replace au mieux, en essayant les regroupements et les ordres
possibles de leurs entretiens. Un mouvement est accepté quand le nouveau
planning n'est pas pire que le courant, ou que celui d'il y a 1 000
mouvements (*late acceptance hill climbing*) : accepter de temps en temps un
planning un peu moins bon sort la recherche des impasses. La commande lance
une recherche par cœur.

Elle s'arrête après `-timeout`, après `-patience` sans amélioration, ou dès
que tous les objectifs atteignent une borne évidente (tout le monde servi,
par exemple). Au-delà de ces bornes, elle ne prouve pas l'optimalité.

Plus elle tourne, mieux c'est : sur le cas de test le plus dur, 10 minutes
avec une patience de 3 minutes donnent 110 rencontres pour un optimum de 111,
contre 103 avec les réglages par défaut (voir
[Qualité](#qualité-de-la-recherche-locale)).

## Utilisation

```sh
igiveyouarendezvous -date 2026-11-03 -rest jour2.yaml example/problem.yaml
igiveyouarendezvous -date 2026-11-04 -rest jour3.yaml jour2.yaml
```

| Option | Sens |
|---|---|
| `-date 2026-11-03` | jour à planifier, nécessaire quand des disponibilités sont données par date |
| `-rest suite.yaml` | écrit le fichier problème du lendemain : le même, avec les rencontres restantes |
| `-objectives a,b,c` | objectifs par priorité, à la place de ceux du fichier problème |
| `-out plan.yaml` | écrit aussi le planning dans un fichier plan, pour reprendre avec `-start` |
| `-start plan.yaml` | part de ce planning au lieu de partir de zéro (voir plus bas) |
| `-fix-all` | avec `-start` : tous les candidats du fichier sont figés |
| `-timeout 1m` | temps de calcul maximal |
| `-patience 30s` | arrêt après ce temps sans amélioration (`0` : jusqu'au timeout) |
| `-progress` | affiche chaque amélioration |
| `-workers N` | recherches lancées en parallèle (par défaut : le nombre de cœurs) |
| `-seed N` | graine aléatoire |
| `-list-objectives` | liste les objectifs |

Les fichiers nommés `.json` sont écrits en JSON, les autres en YAML ; les deux
formats sont lus quel que soit le nom. Sans nom de fichier (ou avec `-`), le
problème est lu sur l'entrée standard.

La commande sort avec le code 0 quand tout va bien, 2 quand des rendez-vous
figés ne tiennent plus (voir plus bas), et 1 en cas d'erreur.

La commande affiche le planning par heure, l'agenda de chaque recruteur, la
valeur atteinte sur chaque objectif et les rencontres restantes. Elle signale
aussi les rencontres **impossibles** : un candidat et un recruteur jamais
libres ensemble assez longtemps. Elles restent dans le fichier problème du
lendemain et reviendront chaque jour tant que les disponibilités ne changent
pas.

## Le fichier problème

Un [exemple commenté](example/problem.yaml), et
[le même en JSON](example/problem.json) :

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
day: 08:00-17:00
breaks: 12:00-13:30
joint-groups:
  - [recruteur 2, recruteur 3]
recruiters:
  recruteur 1:
    duration: 30m
    availability: 09:00-12:00
  recruteur 2:                 # les valeurs par défaut
  recruteur 3:
candidates:
  candidat A:
    meet: [recruteur 1, recruteur 2, recruteur 3]
    priority: 2
  candidat B:
    meet: [recruteur 3]
    availability:
      2026-11-03: none
      2026-11-04: 08:00-12:00
```

Les clés sont en anglais, comme le reste du programme. `recruiters` et
`candidates` sont obligatoires, le reste est facultatif :

| Clé | Défaut | Sens |
|---|---|---|
| `day` | `08:00-12:00` | disponibilité par défaut de tout le monde ; plusieurs plages possibles : `08:00-12:00, 14:00-17:00` |
| `breaks` | aucune | pauses comme le déjeuner (`12:00-13:30`) : pas d'entretien pendant, et jamais d'entretiens d'un candidat de part et d'autre |
| `duration` | `1h` | durée d'entretien par défaut (`45m`, `1h30`, `90`) |
| `step` | automatique | grille horaire ; par défaut la plus grossière sur laquelle tombent toutes les durées et heures |
| `max-wait` | `0` | attente totale permise à un candidat entre ses entretiens : `0` = ils s'enchaînent, `unlimited` = sans limite |
| `joint-groups` | aucun | groupes de recruteurs qui peuvent recevoir ensemble (voir plus bas) |
| `max-joint-size` | `0` | nombre maximal de recruteurs dans un entretien commun (`0` : sans limite) |
| `objectives` | voir plus bas | objectifs, le plus important d'abord |
| `recruiters.<nom>.duration` | `duration` | durée des entretiens avec ce recruteur ; un entretien commun dure autant que sa plus longue partie |
| `recruiters.<nom>.availability` | `day` | plages du recruteur (`none` : absent), chaque jour ou par date (voir plus bas) |
| `candidates.<nom>.meet` | obligatoire | les recruteurs à rencontrer |
| `candidates.<nom>.availability` | `day` | plages du candidat, chaque jour ou par date |
| `candidates.<nom>.priority` | `1` | plus c'est haut, plus c'est important (objectif `priorities`) |

Une clé inconnue (une faute de frappe), un recruteur inconnu ou un nom donné
deux fois est une erreur, avec son numéro de ligne. Entre accolades,
`{availability: "09:00-12:00, 14:00-16:00"}`, mettez les plages entre
guillemets : YAML prend la virgule pour la fin de la valeur.

Une disponibilité est soit la même chaque jour, soit donnée **par date** :

```yaml
candidates:
  candidat A:
    meet: [recruteur 1]
    availability: 09:00-12:00              # chaque jour
  candidat B:
    meet: [recruteur 1]
    availability:                          # par date
      2026-11-03: 08:00-10:00, 14:00-17:00
      2026-11-05: 09:00-12:00
```

Une personne dont la disponibilité est donnée par date est absente les
autres jours. Le jour à planifier se choisit alors avec `-date` : la commande
refuse de tourner sans, et liste les dates trouvées dans le fichier.

**Groupes communs.** Un candidat qui doit rencontrer des recruteurs d'un même
groupe **peut** les voir en un seul entretien commun, ou séparément. Un
recruteur peut appartenir à plusieurs groupes, et un entretien commun
demande que tous ses recruteurs soient dans un même groupe : avec
`[[R2, R3], [R3, R4]]`, 2+3 et 3+4 sont possibles, 2+4 non. Un recruteur sans
groupe reçoit toujours seul ; sans groupes, tous les entretiens sont
individuels.

**Le lendemain.** `-rest` réécrit le fichier problème, sans les candidats
servis et, pour les autres, avec les rencontres restantes. Tout le reste est
conservé tel quel, les commentaires aussi en YAML (mais pas les lignes
vides), pour que le fichier du lendemain se modifie à la main comme le
premier : nouveaux candidats, nouvelles disponibilités…

### Dans un éditeur

Les schémas JSON du [fichier problème](schema/problem.schema.json) et du
[fichier plan](schema/plan.schema.json) permettent à un éditeur de vérifier
les fichiers pendant la saisie, avec la complétion et la documentation de
chaque clé. Dans VS Code, pour le YAML (avec l'extension YAML de Red Hat),
une première ligne comme :

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
```

et pour le JSON, une clé :

```json
{"$schema": "https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json"}
```

Un chemin relatif vers une copie locale du schéma marche aussi. La clé
`$schema` est ignorée par la commande.

## Objectifs

| Nom | Sens |
|---|---|
| `priorities` | maximise les candidats qui voient tous leurs recruteurs, niveau de priorité par niveau, le plus haut d'abord |
| `urgency` | parmi eux, préfère les candidats à qui il reste le moins de jours (voir plus bas) |
| `complete-candidates` | maximise les candidats qui voient tous leurs recruteurs, quelle que soit leur priorité |
| `meetings` | maximise les rencontres tenues |
| `recruiters` | minimise les recruteurs qui ont au moins un entretien dans la journée |
| `recruiter-half-days` | minimise les demi-journées (matin, après-midi) où des recruteurs ont un entretien, sommées sur les recruteurs |
| `interviews` | minimise les entretiens, donc les rencontres groupées quand c'est permis |
| `recruiter-span` | minimise le temps de présence des recruteurs, du début de leur premier entretien à la fin du dernier |
| `candidate-wait` | minimise l'attente des candidats entre leurs entretiens (utile quand `max-wait` n'est pas `0`) |
| `earliest` | place les entretiens le plus tôt possible |

L'optimisation est **lexicographique** : un planning est meilleur qu'un autre
s'il fait mieux sur le premier objectif, ou aussi bien sur le premier et
mieux sur le deuxième, et ainsi de suite. Aucun poids à régler. L'ordre par
défaut est `priorities, urgency, meetings, recruiters, recruiter-half-days,
interviews, recruiter-span`, et la clé `objectives` le change.

`priorities` vaut un objectif par priorité des candidats : d'abord les
candidats de plus haute priorité qui voient tous leurs recruteurs, puis ceux
de la priorité suivante, et ainsi de suite. Un candidat passe donc avant
n'importe quel nombre de candidats de priorité inférieure, même quand les
servir permettrait de voir plus de monde : un candidat de priorité 2 qui doit
voir quatre recruteurs peut coûter trois candidats de priorité 1. Sans
priorités, c'est le nombre de candidats qui voient tous leurs recruteurs.

`urgency` regarde les **jours restants** de chaque candidat : les dates du
fichier, à partir du jour planifié, où le candidat et tous ses recruteurs sont
disponibles. Un candidat à son dernier jour passe avant n'importe quel nombre
de candidats à qui il reste deux jours, et ainsi de suite ; quelqu'un
disponible tous les jours n'est jamais urgent. Placé après `priorities`, il ne
coûte jamais un candidat : le nombre de candidats vus reste le plus haut
possible, et `urgency` choisit seulement lesquels, pour que le candidat qui ne
peut venir qu'aujourd'hui ne soit pas sacrifié à celui qui peut venir demain.
Les jours restants ne comptent que les dates (on ne vérifie pas que tout le
monde est libre en même temps ce jour-là), et la commande les affiche en
`last day`, `2 days left`…

La **pause du déjeuner** sépare le matin de l'après-midi : avec plusieurs
pauses, la plus longue ; sans pause, midi. Un recruteur qui n'a d'entretiens
qu'avant le déjeuner compte une demi-journée, des deux côtés deux.

L'ordre change le résultat. Avec la règle par défaut (toutes les rencontres
d'un candidat le même jour), `meetings` en premier favorise les candidats qui
doivent voir beaucoup de recruteurs, `complete-candidates` en premier ceux
qui en voient peu.

## Suivre, arrêter et reprendre

- **Suivre** : `-progress` affiche les objectifs à chaque amélioration, avec
  le temps écoulé.
- **Arrêter** : Ctrl+C arrête le calcul et affiche le meilleur planning trouvé
  jusque-là, comme si le timeout était atteint. Un second Ctrl+C tue le
  programme.
- **Reprendre** : avec `-out plan.yaml`, le planning est aussi écrit dans un
  fichier plan. `-start plan.yaml` repart de lui, pour qu'un calcul
  interrompu ou trop court continue plus tard. `-start` et `-out` peuvent
  désigner le même fichier, remplacé seulement à la fin :

  ```sh
  igiveyouarendezvous -date 2026-11-03 -timeout 10m -out plan.yaml problem.yaml
  igiveyouarendezvous -date 2026-11-03 -timeout 10m -start plan.yaml -out plan.yaml problem.yaml
  ```

Le fichier plan liste les candidats, chacun **figé** ou non, et note la date
du planning : partir du planning d'un autre jour est refusé.

```yaml
date: 2026-11-03
candidates:
  candidat A:
    fixed: true
    interviews:
      - time: 08:00-09:00
        recruiters: [recruteur 2, recruteur 3]
  candidat B:
    fixed: false
    interviews:
      - time: 09:00-09:30
        recruiters: [recruteur 1]
```

- **Un candidat figé** (`fixed: true`) a été convoqué : ses rendez-vous sont
  gardés exactement tels quels, et la recherche planifie autour. Les plannings
  figés sont ajoutés un par un, dans l'ordre du fichier. Un planning qui ne
  tient plus (un recruteur qui n'est plus disponible à cette heure, une
  nouvelle pause, une rencontre ajoutée ou retirée depuis, un chevauchement
  avec un planning ajouté avant…) n'est **pas** gardé : une **alerte** en tête
  de l'affichage dit pourquoi, et ce qu'est devenu le candidat, replanifié
  entièrement (nouveaux horaires, ou pas planifié aujourd'hui). La commande
  sort alors avec le code 2, pour que la convocation soit renvoyée.
- **Un candidat non figé** peut être déplacé, ou laissé de côté. Ses
  entretiens qui ne tiennent plus sont abandonnés, chacun avec la raison, et
  le candidat est replanifié. Sur un problème inchangé, le résultat n'est
  jamais pire que le départ.
- **`fixed` est obligatoire** : un candidat sans est une erreur, car oublier
  de figer un candidat convoqué déplacerait ses rendez-vous sans prévenir.
  `-out` écrit `false` pour les candidats qu'il planifie, et garde `true` pour
  les figés ; une fois les convocations envoyées, `-fix-all` fige tout le
  fichier.

## Comme bibliothèque

```go
import (
	"github.com/FrederickFrance/igiveyouarendezvous/config"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

f, err := config.Load("problem.yaml") // ou config.Parse(data), YAML ou JSON
p, o, err := f.Problem("2026-11-03")  // le problème et les options de ce jour

ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()

s, err := planning.Search(ctx, p, o, planning.SearchOptions{
	Patience: 10 * time.Second,
	Workers:  runtime.NumCPU(),
	Progress: func(pr planning.Progress) { log.Println(pr.Elapsed, pr.Scores) },
})
// s.Interviews : le planning ; s.Scores : la valeur atteinte sur chaque objectif ;
// s.Unscheduled, s.Impossible : ce qui n'a pas pu être placé.

next, err := f.Next(p, s)          // le fichier problème du lendemain
err = next.Write(w, config.YAML)   // avec ses commentaires
```

Le problème peut aussi se construire directement :

```go
p := &planning.Problem{
	Recruiters: []planning.Recruiter{{Name: "R1"}, {Name: "R2", Duration: 30 * time.Minute}, {Name: "R3"}},
	Candidates: []planning.Candidate{{Name: "A", Priority: 2}, {Name: "B"}},
	Requests: []planning.Request{
		{Candidate: 0, Recruiter: 0},
		{Candidate: 0, Recruiter: 1},
		{Candidate: 0, Recruiter: 2},
		{Candidate: 1, Recruiter: 0},
	},
	JointGroups: [][]int{{0, 1}}, // R1 et R2 peuvent recevoir ensemble, R3 seul
}

s, err := planning.Search(ctx, p, planning.Options{
	Day:    []planning.Interval{{Start: 8 * time.Hour, End: 17 * time.Hour}},
	Breaks: []planning.Interval{{Start: 12 * time.Hour, End: 13*time.Hour + 30*time.Minute}},
}, planning.SearchOptions{Patience: 10 * time.Second})
```

- **Suivre** : `Progress` est appelé à chaque amélioration. En plus des
  scores, `pr.Schedule` est le planning lui-même, à afficher ou à garder.
- **Arrêter** : `Search` s'arrête quand son contexte est terminé, et rend
  alors le meilleur planning trouvé jusque-là, sans erreur. Pour l'arrêter à
  la demande, utilisez `context.WithCancel` et appelez `cancel()`, depuis une
  autre goroutine ou depuis `Progress`.
- **Reprendre** : `SearchOptions{Start: s}` part du planning `s`, par exemple
  rendu par un `Search` précédent ou reçu par `Progress`. Les plannings des
  candidats dont des entretiens sont marqués `Fixed` sont ajoutés un par un
  et gardés tels quels, sauf s'ils ne tiennent plus. `planning.Clean` dit à
  l'avance ce qui est gardé, les plannings figés rejetés et les entretiens
  abandonnés, avec les raisons. `config.WritePlan` et `config.ReadPlan`
  enregistrent un planning et le relisent face à un problème qui a changé.
- **Priorités et urgence** : `Candidate.Priority` et `Candidate.DaysLeft`
  (calculé par `config` à partir des dates) ; `Score.Priority` et
  `Score.DaysLeft` donnent le niveau de chaque score des objectifs
  `priorities` et `urgency`.

`planning.Solve(ctx, gomilp.New(), p, options)` fait de même avec le solveur
exact, sur de petits problèmes seulement (voir [le modèle exact](#le-modèle-exact)).

| Paquet | Contenu |
|---|---|
| `planning` | le problème, les options, `Search`, `Solve`, `Check` (vérifie un planning indépendamment), `Evaluate`, `Clean` |
| `mip` | modèle linéaire générique et interface `Solver`, pour brancher un autre solveur exact |
| `mip/gomilp` | le solveur exact fourni |
| `config` | lit le fichier problème (`Load`, `Parse`, `Problem(date)`), écrit celui du lendemain (`Next`, `Write`) ; écrit et lit les fichiers plan (`WritePlan`, `ReadPlan`), en YAML ou en JSON |

`planning` et `mip` ne dépendent que de la bibliothèque standard.

## Qualité de la recherche locale

Comparaison avec l'optimum, prouvé avec le solveur exact
[HiGHS](https://highs.dev) (qui ne fait pas partie du projet), sur des
problèmes aléatoires (1 à 4 rencontres par candidat), avec les règles par
défaut (toutes les rencontres d'un candidat le même jour, enchaînées) et les
objectifs `complete-candidates, meetings, interviews, recruiter-span` (un
ancien ordre par défaut ; sans priorités, `priorities` donne la même chose
que `complete-candidates`). Valeurs : candidats complets / rencontres /
entretiens / présence des recruteurs. Une recherche (un cœur, graine 1),
arrêtée après 30 s.

| Cas | Optimum | Recherche | Dernière amélioration |
|---|---|---|---|
| 100 × 10, matin, entretiens d'une heure | 35 / 40 / 35 / 40h00 | idem | < 0,1 s |
| 100 × 12, matin, entretiens d'une heure | 32 / 48 / 32 / 48h00 | idem | < 0,1 s |
| 100 × 10, durées et disponibilités variées | 32 / 41 / 39 / 32h30 | idem | 5 s |
| 60 × 10, durées et disponibilités variées | 31 / 56 / 42 / 35h30 | 31 / 55 / 41 / 35h45 | 3 s |
| 100 × 12, durées et disponibilités variées | 36 / 52 / 48 / 38h00 | 36 / 51 / 46 / 37h30 | 3 s |
| 60 × 10, journée 08:00-17:00 avec déjeuner | 34 / 61 / 38 / 66h00 | 34 / 61 / 38 / 67h00 | 2 s |
| 100 × 12, journée 08:00-17:00 avec déjeuner | 56 / 111 / 76 / 98h30 | 56 / 103 / 77 / 98h00 | 26 s |
| idem, 2 recherches, `-timeout 10m -patience 3m` | | 56 / 110 / 84 / 99h30 | 50 s |
| idem, attente illimitée | 56 / 111 / 76 / 98h00 | 56 / 111 / 78 / 99h00 | 6 s |

Le premier objectif est optimal dans tous les cas, en moins de 10 s.
Ailleurs, il manque en général une rencontre sur le deuxième objectif, ou une
heure de présence ; sur le cas le plus dur (une journée entière avec
déjeuner), il manque 8 rencontres sur 111 en 30 s, et 1 quand la recherche
tourne plus longtemps.

Les tests vérifient aussi, sur 60 petits problèmes aléatoires (avec ou sans
pauses, attente, priorités et urgence), que la recherche trouve le
même résultat qu'une énumération de tous les plannings possibles ; elle le
fait aussi sur 300. Chaque planning rendu, par n'importe quel moteur, est
revérifié par `planning.Check`.

## Le modèle exact

`planning.Solve` écrit le problème comme un programme linéaire en nombres
entiers : le temps est découpé en créneaux ; chaque variable binaire dit
« cet entretien (un candidat et un groupe de recruteurs permis) commence à ce
créneau ». Les objectifs sont optimisés l'un après l'autre, chacun étant figé
avant le suivant.

Avec [go-milp](https://github.com/daniel-sullivan/go-milp) (Go pur), il
prouve l'optimum de petits problèmes, quelques candidats : les tests s'en
servent de référence. Sur des cas réels il ne sert à rien, d'où son absence
de la commande : sur 60 à 100 candidats, il ne trouve aucun planning en
plusieurs minutes, même avec la valeur du planning glouton comme borne, et
la recherche locale suivie d'une réoptimisation exacte de morceaux du
planning ne gagne aucune rencontre non plus.

## Développement

```sh
go test -race ./...     # tests unitaires, exemples et comparaison exhaustive sur de petits problèmes
golangci-lint run       # linter strict (.golangci.yml), version 2.15 ou plus pour Go 1.27
```

Les tests vérifient les exemples et les fichiers écrits par la commande face
aux schémas JSON, et que les schémas connaissent les mêmes clés et objectifs
que le code.

La CI GitHub (`.github/workflows/ci.yml`) lance les tests avec le détecteur
de concurrence et la couverture, le linter et `osv-scanner` à chaque push et
pull request, et l'analyse à nouveau chaque semaine. Dependabot propose les
mises à jour des modules Go et des actions. Un tag de version (`git tag
v0.1.0 && git push origin v0.1.0`) publie une version avec les zips prêts à
l'emploi du tutoriel (`.github/workflows/release.yml`, `scripts/package.sh`).

## Dépendances

| Module | Licence | Utilisé par |
|---|---|---|
| go.yaml.in/yaml/v3 | MIT et Apache-2.0 | `config` |
| github.com/daniel-sullivan/go-milp | MIT | `mip/gomilp` |
| github.com/stretchr/testify | MIT | tests seulement |
| github.com/santhosh-tekuri/jsonschema/v6 | Apache-2.0 | tests seulement (les schémas) |

Un binaire distribué doit être accompagné de leurs mentions de copyright et
de licence (et du `NOTICE` de go.yaml.in/yaml).

## Licence

Au choix, [0BSD](LICENSE) ou [WTFPL](WTFPL.txt), version 2 : faites-en ce que
vous voulez, sans condition. Les deux disent la même chose ; la 0BSD est celle
que pkg.go.dev reconnaît, pour qu'il affiche la documentation.

Ce programme est fourni sans aucune garantie, dans la mesure permise par la
loi.
