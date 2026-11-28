# igiveyouarendezvous

[![CI](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml/badge.svg)](https://github.com/FrederickFrance/igiveyouarendezvous/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/FrederickFrance/igiveyouarendezvous.svg)](https://pkg.go.dev/github.com/FrederickFrance/igiveyouarendezvous)

[English](README.md) · **Français** · [Deutsch](README.de.md) · [Italiano](README.it.md) · [ไทย](README.th.md) · [tlhIngan Hol](README.tlh.md) · [Ch'ti](README.pcd.md)

**Vous avez les disponibilités de chacun ; igiveyouarendezvous fait le
planning.** Qui rencontre qui, et quand, pour cinquante candidats et cinq
recruteurs sur une semaine comme pour une journée : le plus de candidats
possible voient tous leurs recruteurs, et le planning revient en tableau pour
un publipostage, ou en plan pour reprendre plus tard. Un sondage comme Doodle
ou Framadate recueille les disponibilités ; ce programme fait le planning qui
vient après.

**[Essayer en ligne](https://frederickfrance.github.io/igiveyouarendezvous/)** : modifiez un exemple et planifiez-le, dans
votre navigateur, sans rien installer. Tout se calcule sur votre ordinateur :
ce que vous tapez n'est envoyé nulle part. Première fois ? Le [mode
d'emploi](GUIDE.fr.md) vous guide, tâche par tâche.

Une commande et une bibliothèque Go, libres (0BSD ou WTFPL).

Chaque candidat doit rencontrer certains recruteurs. Les règles :

- un recruteur reçoit un candidat à la fois ; un candidat peut rencontrer
  plusieurs recruteurs en un seul entretien commun quand ils font partie d'un
  même **groupe commun** ;
- un candidat rencontre **tous ses recruteurs dans la période** planifiée
  (une journée, une semaine…), ou aucun ;
- entre deux entretiens d'un candidat, un **écart minimal et maximal** :
  enchaînés par défaut, ou par exemple au moins trois jours d'écart ; deux
  entretiens d'une même journée ne sont jamais de part et d'autre d'une
  **pause** comme le déjeuner ;
- les rencontres d'un candidat peuvent avoir un **ordre** : tel recruteur
  d'abord, puis tel autre, les autres à n'importe quel moment ;
- chacun a des disponibilités, les mêmes chaque jour ou **par date**, et
  chaque recruteur une durée d'entretien ;
- les candidats peuvent avoir une **priorité** : un candidat passe avant
  n'importe quel nombre de candidats de priorité inférieure ;
- entre candidats qui se valent, les **plus urgents** passent d'abord : ceux
  à qui il reste le moins de jours.

Tout est décrit dans un seul **fichier problème**, en YAML ou en JSON. Un
lancement planifie **une période**, d'une première à une dernière date, du
lundi au vendredi par défaut, sans les jours fériés ; les candidats qui ne
tiennent pas sont écrits dans le fichier problème de la période suivante.

Pas seulement des entretiens : les enregistrements d'une émission avec des
invités difficiles à réunir, le programme d'un colloque, des parties de jeu
de rôle… voir [Par usage](#par-usage).

## Premier planning, pas à pas

Sans rien connaître à la programmation ni à la ligne de commande : suivez ces
étapes exactement, dans l'ordre. (Pour simplement essayer, la [démo en
ligne](https://frederickfrance.github.io/igiveyouarendezvous/) n'en demande aucune.)

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
seulement les candidats restants. Complétez-le si besoin, et utilisez-le le
lendemain à la place de `problem.yaml`. (Pour planifier toute une semaine
d'un coup, voyez `from` et `to` dans `example-full.yaml`.)

**En cas d'erreur**, le programme indique la ligne du fichier en cause :
vérifiez les espaces en début de ligne, les deux-points, et que les noms sont
écrits comme sous `recruiters`. L'exemple complet et commenté,
`example-full.yaml`, montre tout le reste : disponibilités par personne et
par date, priorités, entretiens communs… `example-panel.yaml` planifie une
commission de recrutement, et les autres fichiers `example-….yaml` autre
chose que des entretiens : une émission, un colloque, des parties de jeu de
rôle. **Le dossier `exemples-fr` les contient tous en français**, avec des
commentaires et des noms en français (les clés, elles, restent en anglais) :
`exemples-fr/simple.yaml` est la version française de `problem.yaml`.

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
planning un peu moins bon sort la recherche des impasses.

Il y a deux autres sortes de mouvements. Un re-horodatage prend une journée
entière, garde qui rencontre qui et resserre les entretiens autant que
possible. Une onde retire un entretien d'un candidat, écarte ses autres
entretiens de la journée et le replace là où le planning est le meilleur,
même si le candidat doit alors attendre : une demi-heure d'attente pour un
candidat permet parfois aux entretiens d'un recruteur de s'enchaîner. La
recherche choisit seule entre les trois sortes : elle donne plus de
mouvements à celle qui a récemment le plus amélioré le planning pour le
travail fourni. Ce travail est compté, jamais chronométré : la même graine
donne le même planning sur toutes les machines. Avec `-exact`,
certains mouvements prennent aussi la journée d'un recruteur qui a un trou et
replacent ses entretiens là où ils laissent le moins de trous possible, pour
lui et pour tous ceux qu'il rencontre : sur des journées de rencontres d'un
créneau, comme dans un forum d'affaires, ce placement est calculé et non
essayé. D'où viennent ces méthodes : [Références](#références). La commande
lance une recherche par cœur.

Elle s'arrête après `-timeout`, après `-patience` sans amélioration, ou dès
que tous les objectifs atteignent une borne évidente (tout le monde servi,
par exemple). Au-delà de ces bornes, elle ne prouve pas l'optimalité.

Plus elle tourne, mieux c'est : sur le cas de test le plus dur, 10 minutes
avec une patience de 3 minutes donnent 110 rencontres pour un optimum de 111,
contre 103 avec les réglages par défaut (voir
[Qualité](#qualité-de-la-recherche-locale)).

## Utilisation

```sh
igiveyouarendezvous -rest semaine2.yaml example/problem.yaml   # la semaine du fichier
igiveyouarendezvous -from 2026-11-16 -to 2026-11-20 -rest semaine3.yaml semaine2.yaml
igiveyouarendezvous -date 2026-11-23 semaine3.yaml             # un seul jour
```

| Option | Sens |
|---|---|
| `-from 2026-11-16` | premier jour de la période à planifier, à la place de `from` dans le fichier |
| `-to 2026-11-20` | dernier jour, inclus, à la place de `to` dans le fichier (par défaut : le premier jour) |
| `-date 2026-11-23` | un seul jour : `-from` et `-to` à la fois |
| `-not-before 2026-11-18` | pour replanifier en cours de période : rien de nouveau avant cette date (voir plus bas) |
| `-rest suite.yaml` | écrit le fichier problème de la période suivante : le même, sans les candidats servis |
| `-objectives a,b,c` | objectifs par priorité, à la place de ceux du fichier problème |
| `-out plan.yaml` | écrit aussi le planning dans un fichier plan, pour reprendre avec `-start` |
| `-csv planning.csv` | écrit aussi le planning en tableau, une ligne par entretien (date, début, fin, candidat, recruteurs, figé), pour Excel ou un publipostage des convocations |
| `-csv-separator ";"` | le séparateur du tableau : `,` (par défaut), `;` (ce qu'attend Excel en français) ou `tab` |
| `-start plan.yaml` | part de ce planning au lieu de partir de zéro (voir plus bas) |
| `-fix-all` | avec `-start` : tous les candidats du fichier sont figés |
| `-timeout 1m` | temps de calcul maximal |
| `-patience 30s` | arrêt après ce temps sans amélioration (`0` : jusqu'au timeout) |
| `-progress` | affiche chaque amélioration |
| `-progress-json` | affiche chaque amélioration en un objet JSON par ligne, pour un programme |
| `-workers N` | recherches lancées en parallèle (par défaut : le nombre de cœurs) |
| `-seed N` | graine aléatoire |
| `-retime S` | pour les mesures : la part des mouvements qui re-horodatent une journée entière, de `0` à `1` ; la recherche choisit quand il est omis |
| `-exact S` | pour les mesures, expérimental : la part des mouvements qui re-horodatent la journée d'un recruteur au moins de trous possible, de `0` (par défaut : jamais) à `1` |
| `-list-objectives` | liste les objectifs |

Les fichiers nommés `.json` sont écrits en JSON, les autres en YAML ; les deux
formats sont lus quel que soit le nom. Sans nom de fichier (ou avec `-`), le
problème est lu sur l'entrée standard.

La commande sort avec le code 0 quand tout va bien, 2 quand des rendez-vous
figés ne tiennent plus (voir plus bas), et 1 en cas d'erreur.

La commande affiche le planning par heure (avec les dates, sur plusieurs
jours), l'agenda de chaque recruteur jour par jour, la valeur atteinte sur
chaque objectif et les rencontres restantes. Elle signale aussi les
rencontres **impossibles** : les personnes disponibles à aucun moment de la
période, puis les candidats et recruteurs jamais libres ensemble assez
longtemps. Elles restent dans le fichier problème de la
période suivante et reviendront à chaque fois tant que les disponibilités ne
changent pas.

## Le fichier problème

Un [exemple commenté](example/problem.yaml), et
[le même en JSON](example/problem.json) :

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
from: 2026-11-09
to: 2026-11-13
closed: [2026-11-11]
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
    meet: {recruteur 1: 1, recruteur 2: X, recruteur 3: 2}
    availability:
      2026-11-09: none
      2026-11-10: 08:00-12:00
```

Les clés sont en anglais, comme le reste du programme. `recruiters` et
`candidates` sont obligatoires, le reste est facultatif :

| Clé | Défaut | Sens |
|---|---|---|
| `from`, `to` | aucune | la période à planifier, les deux jours inclus (`to` : `from` par défaut) ; `-from` et `-to` les remplacent |
| `not-before` | aucune | pour replanifier en cours de période : aucun nouvel entretien avant cette date (voir plus bas) |
| `weekdays` | `[mon, tue, wed, thu, fri]` | les jours de la semaine avec des entretiens (`mon` ou `monday`…) |
| `closed` | aucune | dates sans entretien : `[2026-11-11, 2026-12-25]` |
| `day` | `08:00-12:00` | disponibilité par défaut de tout le monde, chaque jour ; plusieurs plages possibles : `08:00-12:00, 14:00-17:00` |
| `breaks` | aucune | pauses comme le déjeuner (`12:00-13:30`) : pas d'entretien pendant, et jamais deux entretiens d'un candidat d'une même journée de part et d'autre |
| `duration` | `1h` | durée d'entretien par défaut (`45m`, `1h30`, `90`) |
| `step` | automatique | grille horaire ; par défaut la plus grossière sur laquelle tombent toutes les durées et heures |
| `min-gap` | `0` | le temps minimal entre deux entretiens d'un candidat : une durée (`30m`, `2h`) ou des jours calendaires (`3d`) |
| `max-gap` | `0` | le temps maximal entre deux entretiens d'un candidat : une durée, des jours calendaires (`1d` : le lendemain au plus tard) ou `unlimited` ; `0` et `0` : enchaînés |
| `joint-groups` | aucun | groupes de recruteurs qui peuvent recevoir ensemble (voir plus bas) |
| `panels` | aucun | groupes de recruteurs qui reçoivent toujours ensemble, comme un jury (voir plus bas) |
| `max-joint-size` | `0` | nombre maximal de recruteurs dans un entretien commun (`0` : sans limite) |
| `rooms` | `0` | nombre maximal d'entretiens en même temps, quels qu'en soient les participants : les salles, les tables ou les stands dont on dispose (`0` : sans limite) |
| `break-spread` | aucun | écart maximal entre les nombres de pauses des recruteurs, parmi ceux qui ont au moins deux entretiens (`2` : personne n'attend plus de deux fois plus souvent qu'un autre) ; aucun : sans limite |
| `precedences` | aucune | candidats à tenir dans l'ordre : des paires de noms, `[[a, b], [a, c]]` demandant que `a` soit fini avant que `b` et `c` commencent (voir plus bas) |
| `objectives` | voir plus bas | objectifs, le plus important d'abord |
| `recruiters.<nom>.duration` | `duration` | durée des entretiens avec ce recruteur ; un entretien commun dure autant que sa plus longue partie |
| `recruiters.<nom>.availability` | `day` | plages du recruteur (`none` : absent), chaque jour ou par date (voir plus bas) |
| `recruiters.<nom>.min-gap` | `recruiter-min-gap` | le temps minimal entre deux entretiens de ce recruteur, de la fin de l'un au début du suivant, nuits comprises : pour mettre ses notes au propre (`30m`), pour se reposer (`12h`) ; les trajets s'y ajoutent (voir `travel`) ; `0` : aucun |
| `recruiters.<nom>.continuous` | `false` | pour un invité : une fois parti des visites d'un jour, il ne revient pas ce jour-là (voir les invités, plus bas) |
| `recruiter-min-gap` | `0` | le `min-gap` de chaque recruteur qui n'a pas le sien, comme `10m` de battement pour tout le monde |
| `candidates.<nom>.meet` | obligatoire sans `meetings` | les recruteurs à rencontrer, un entretien chacun, dans n'importe quel ordre, ou avec leurs rangs et les durées des rencontres (voir plus bas) |
| `candidates.<nom>.meetings` | aucune | des rencontres avec plusieurs recruteurs à la fois, chacune telle quelle ; un recruteur peut revenir dans plusieurs (voir plus bas) |
| `candidates.<nom>.availability` | `day` | plages du candidat, chaque jour ou par date |
| `candidates.<nom>.priority` | `1` | plus c'est haut, plus c'est important (objectif `priorities`) |
| `candidates.<nom>.required` | `false` | `true` : une contrainte et non plus un objectif — un planning sans ce candidat est refusé et il est nommé (voir plus bas) |
| `candidates.<nom>.duration` | celle des recruteurs | durée des entretiens de ce candidat, à la place de celles des recruteurs (la durée d'une rencontre dans `meet` l'emporte) |
| `candidates.<nom>.together` | `false` | `true` : une réunion, tous les recruteurs à la fois en un seul entretien, ou rien (voir [Par usage](#par-usage)) |
| `candidates.<nom>.place` | aucun | où l'on rencontre le candidat, comme la ville d'un tournage ; aucun : pas de trajet vers lui ni depuis lui, comme un appel (voir plus bas) |
| `travel` | aucun | les trajets entre les lieux : `default`, et `journeys`, comme `[Paris, Lyon, 2h]` (voir plus bas) |

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

Une personne sans disponibilité est disponible dès que `day` le permet.
Quand la plupart des autres en ont une, c'est peut-être un oubli : la commande
les liste dans une note. Une personne dont la disponibilité est donnée par date est
absente les autres jours. Il faut alors une période, `from` et `to` dans le fichier ou
`-from` et `-to` : sans, la commande refuse de tourner, et liste les dates
trouvées dans le fichier. Sans aucune date, un fichier sans période planifie
une journée sans date, comme dans le tutoriel.

**La période.** Les entretiens ont lieu les jours de `weekdays` entre `from`
et `to`, sans les dates de `closed`. Un candidat rencontre tous ses
recruteurs dans la période, ou aucun, et un entretien ne déborde jamais
après minuit.

**Les écarts.** `min-gap` et `max-gap` encadrent le temps entre la fin d'un
entretien d'un candidat et le début du suivant. Les durées comptent le
temps ; les jours comptent les jours calendaires, de la date d'un entretien à
celle du suivant, quelles que soient les heures : avec `min-gap: 3d`, un
entretien un lundi est suivi le jeudi au plus tôt, et avec `max-gap: 1d` le
mardi au plus tard (un vendredi n'est suivi le lundi qu'avec 3 jours).

```yaml
min-gap: 0          # par défaut : enchaînés, le même jour
max-gap: 0

min-gap: 3d         # au moins trois jours entre deux entretiens
max-gap: unlimited

min-gap: 30m        # le même jour, avec 30 minutes à 2 heures entre deux
max-gap: 2h
```

**Les salles.** `rooms` limite le nombre d'entretiens tenus en même temps,
quels qu'en soient les participants : les salles, les tables ou les stands
dont on dispose. Rien d'autre ne le borne, et vingt recruteurs libres à dix
heures sont tous rencontrés à dix heures si le fichier ne dit pas le
contraire — ce qui est juste quand chacun reçoit dans son bureau, et faux
dans un hall qui a quatre tables. `0`, la valeur par défaut, ne limite rien.

**La même attente pour tous.** `break-spread` borne l'écart entre le
recruteur qui attend le plus souvent et celui qui attend le moins : avec
`break-spread: 2`, personne n'attend plus de deux fois plus souvent qu'un
autre. C'est une contrainte dure et non un objectif, donc elle ne se
négocie jamais contre un meilleur score ailleurs : un planning qui l'enfreint
est refusé, par la recherche comme par `-check`. Elle compare les recruteurs
qui ont au moins deux entretiens, puisqu'un seul entretien n'est pas de
l'attente et qu'il n'y a là rien à partager équitablement — les compter
tirerait le minimum à zéro et la règle ne dirait plus que ceci : personne
n'attend plus de deux fois. Omettre la clé ne limite rien ;
`break-spread: 0` est une limite de zéro, qui demande que chaque recruteur
attende exactement aussi souvent que les autres. Les deux ne sont pas la
même chose.

```yaml
rooms: 14                                # un hall de quatorze tables
break-spread: 2                          # l'attente répartie également
objectives: [meetings, recruiter-breaks]
```

**L'ordre entre candidats.** Les rangs de `meet` ordonnent les entretiens
d'**un** candidat ; `precedences` ordonne les candidats entre eux. Chaque
entrée est une paire, le premier fini avant que le second commence : chaque
entretien du premier se termine au plus tard quand commence chaque
entretien du second.

```yaml
precedences:
  - [présélection, jury]     # le jury ne voit personne avant la présélection
  - [présélection, débrief]
  - [jury, débrief]          # et le débrief vient en dernier
```

Des paires plutôt que des rangs, parce qu'une paire ne dit que ce qu'elle
dit. Les rangs découpent les candidats en couches, et deux candidats de
couches différentes sont alors ordonnés même là où rien ne le demande ; une
couche écrite sous la forme des paires qu'elle représente demande exactement
ce qu'elle veut dire. La clé est optionnelle et ne pèse que les ordres
réellement nécessaires.

Un candidat non planifié ne retient personne : l'ordre s'applique entre les
entretiens qui ont lieu. Un cycle est refusé, en nommant le tour, puisqu'un
cycle réparti sur plusieurs lignes n'a pas de paire fautive à montrer.

**L'ordre.** `meet` est une liste, dans n'importe quel ordre, ou les
recruteurs avec leurs rangs : le rang 1 d'abord, puis le rang 2, et ainsi de
suite ; `X` (ou `any`, ou rien) à n'importe quel moment. Avec
`{recruteur 1: 1, recruteur 2: X, recruteur 3: 2}`, le recruteur 1 passe
avant le recruteur 3, et le recruteur 2 avant, entre ou après. Plusieurs
recruteurs peuvent avoir le même rang, dans n'importe quel ordre entre eux ;
un entretien commun ne mélange jamais deux rangs.

**La durée d'une rencontre.** Une rencontre peut avoir sa propre durée, à la
place du `duration` du candidat, ou sinon de celui du recruteur : `meet:
{Alice: 15m, Bob: 6m}`, ou avec un rang, `{Alice: {rank: 1, duration: 15m}}`.
Un nombre seul est un rang : une durée s'écrit avec son unité (`15m`,
`1h30`). Un entretien commun dure autant que sa plus longue rencontre.

**Groupes communs.** Un candidat qui doit rencontrer des recruteurs d'un même
groupe **peut** les voir en un seul entretien commun, ou séparément. Un
recruteur peut appartenir à plusieurs groupes, et un entretien commun
demande que tous ses recruteurs soient dans un même groupe : avec
`[[R2, R3], [R3, R4]]`, 2+3 et 3+4 sont possibles, 2+4 non. Un recruteur sans
groupe reçoit toujours seul ; sans groupes, tous les entretiens sont
individuels.

**Jurys.** Des recruteurs qui reçoivent toujours ensemble, comme un jury :
`panels: [[Alice, Bob, Chloé]]`. Les membres d'un jury qu'un candidat doit
rencontrer, il les rencontre en un seul entretien, sans personne d'autre, ou
pas du tout : jamais l'un après l'autre. Un recruteur appartient à un jury au
plus, et alors à aucun groupe commun ; `max-joint-size` ne s'applique pas aux
jurys. Exemple complet : [une commission de
recrutement](example/fr/panel.yaml).

**Rencontres.** `meet` donne un entretien par recruteur ; `meetings` donne
des rencontres avec plusieurs recruteurs à la fois, telles quelles, sans
personne d'autre, où un recruteur peut revenir : le tournage d'un épisode,
avec l'animatrice et le cadreur, puis avec eux et un invité ; ou un
candidat qui voit le responsable seul, puis avec un collègue.

```yaml
candidates:
  épisode Lille:
    meetings:
      - {with: [Animatrice, Cadreur], duration: 1h, times: 2}   # deux rencontres pareilles
      - {with: [Animatrice, Cadreur, Nathan], duration: 2h, rank: 2}
```

`times` répète une rencontre, `rank` ordonne les rencontres comme dans
`meet` (`1` d'abord, `X` n'importe quand). Une rencontre dure sa
`duration`, sinon celle du candidat, sinon la `duration` par défaut du
fichier, jamais celle d'un recruteur, puisqu'ils sont plusieurs. Le
candidat a tout `meet` et tout `meetings`, ou rien ; les écarts (`min-gap`,
`max-gap`) valent entre tous ses entretiens, et le `min-gap` d'un recruteur
entre deux rencontres d'un même candidat aussi. `together: true` est un
raccourci pour une seule rencontre avec tout `meet` : il ne va donc pas avec
`meetings`.

**Lieux et trajets.** Un candidat peut avoir un lieu, `place`, comme la
ville d'un tournage ; les recruteurs vont de l'un à l'autre. Entre deux
entretiens d'un recruteur à des lieux différents, il lui faut le trajet,
plus son `min-gap`, nuits comprises. Les trajets sont des estimations hautes
que vous donnez, de porte à porte : le programme ne calcule aucun
itinéraire.

```yaml
travel:
  default: 5h                 # tout trajet non listé
  journeys:
    - [Paris, Lyon, 2h30]     # dans les deux sens
    - [Lyon, Marseille, 2h]
candidates:
  épisode 1:
    meet: [Animatrice, Invité 1]
    together: true
    place: Lyon
```

Un trajet par d'autres lieux compte quand il est plus court : Paris à
Marseille prend au plus 2h30 plus 2h ici. C'est ce qui rend un défaut
dangereux, puisqu'il relie tous les couples : avec `default: 10m`, deux
lieux ne sont jamais à plus de vingt minutes l'un de l'autre, quoi que dise
le fichier, et un trajet long écrit pour interdire une liaison ne
l'interdit pas. **Un défaut n'est sûr que si tous les trajets listés sont
plus courts que lui** — comme les `5h` ci-dessus. À côté d'une liste
complète il n'est jamais consulté, et `-check` le dit : retirez-le, pour
qu'un lieu ajouté plus tard sans ses trajets soit refusé plutôt que doté
d'une valeur que rien ne vérifie. Sans `default`, les trajets n'ont pas à
lister chaque paire : une paire atteinte par d'autres lieux coûte ce
passage, donc Paris à Marseille n'a pas besoin de ligne une fois Paris à
Lyon et Lyon à Marseille donnés. `-check` affiche les temps ainsi déduits,
avec le chemin emprunté, pour qu'on puisse les relire avant de s'y fier.
Deux lieux que rien ne relie sont refusés : il n'y a aucun temps avec quoi
planifier, et les tenir pour un même lieu est la façon dont un planning
devient infaisable le jour venu. Avec un `default`, la commande signale un
lieu qu'aucun trajet ne nomme, peut-être une faute de frappe. Un candidat sans
lieu ne demande aucun trajet, mais un recruteur doit quand même aller de
Paris à Lyon autour de lui. Sur la grille du temps, les trajets sont
arrondis aux 5 minutes supérieures : `2h07` compte pour `2h10`, plutôt que
de faire une grille d'une minute. Exemple complet : [une
tournée](example/fr/tour.yaml).

**Les trajets tiennent ensemble.** Un trajet par d'autres lieux compte
quand il est plus court, donc les temps que vous donnez doivent être
cohérents : Paris-Dijon ne peut pas durer plus que Paris-Lyon puis
Lyon-Dijon. Deux conséquences, surprenantes toutes les deux. Un **trajet
par défaut relie tous les couples de lieux** : avec un défaut de dix
minutes, deux lieux ne sont jamais à plus de vingt minutes l'un de
l'autre, quoi que dise le fichier — un trajet long donné pour interdire
une liaison ne l'interdit pas. Et **un défaut de zéro n'est pas « pas de
liaison », c'est une liaison gratuite** : tout couple absent du fichier
devient un passage sans coût. Un lieu cité dans `journeys` est un lieu
même si aucun candidat n'y est rencontré, donc un nom mal orthographié
ouvre un passage que personne ne voulait. Le programme signale chaque
trajet qu'un passage rend plus court que ce que vous avez indiqué.

**La période suivante.** `-rest` réécrit le fichier problème, sans les
candidats servis ; les autres gardent toutes leurs rencontres. La période
(`from`, `to`, `not-before`), planifiée, est retirée, pour choisir la
suivante. Tout le reste est conservé tel quel, les commentaires aussi en
YAML (mais pas les lignes vides), pour que le fichier de la période suivante
se modifie à la main comme le premier : nouveaux candidats, nouvelles
disponibilités…

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

### Avec un assistant IA

Pour ne pas écrire le YAML soi-même : [une consigne](prompt/fr.md) à copier
dans un assistant IA (Claude, ChatGPT, Mistral…), suivie de vos règles en
français courant (« chaque candidat voit d'abord la RH 30 minutes, puis le
manager… »). L'assistant écrit le fichier, commenté règle par règle, et dit
ce qu'il a supposé ; le programme le vérifie, et une erreur, avec son numéro
de ligne, se recolle dans l'assistant pour qu'il corrige.

Ce que vous tapez part chez le fournisseur de l'assistant : donnez des noms
de remplacement (`Candidat 1`, des initiales), et remettez les vrais noms
dans le fichier ensuite. La démo et le programme, eux, n'envoient rien.

## Par usage

Le programme place des « candidats » qui doivent rencontrer des
« recruteurs », chacun à un seul endroit à la fois. Bien d'autres plannings
entrent dans ce moule : les noms des clés restent, seul leur sens change.

La plupart des clés ci-dessous ne servent qu'à un usage ; absentes, elles ne
changent rien pour les autres.

### Recrutement

Les entretiens proprement dits : candidats, recruteurs, rangs, entretiens
communs et jurys (voir [le fichier problème](#le-fichier-problème)),
priorités et urgence. Voir
[example/fr/problem.yaml](example/fr/problem.yaml),
[example/fr/panel.yaml](example/fr/panel.yaml) et
[example/fr/assessors.yaml](example/fr/assessors.yaml).

**Un recruteur parmi plusieurs** s'écrit avec une liste dans la liste de
`meet` : `meet: [Manager, [Ada, Ben, Cléo]]` fait voir le manager et un des
trois évaluateurs, celui qui arrange le planning. Dans la forme en mapping,
les alternatives portent le même `choice`, un nom au choix, et partagent
leur rang : `{Manager: 1, Ada: {rank: 2, choice: évaluateur}, Ben: {rank: 2,
choice: évaluateur}}`. Dans une réunion, c'est une liste dans `with` :
`{with: [Animateur, [Caméra 1, Caméra 2]]}`. Le candidat est complet avec
n'importe lequel d'entre eux, le choix compte pour une rencontre dans
`meetings`, et les autres ne sont pas à planifier plus tard : on n'en avait
pas besoin. Avec `optional: true` sur chacun, le choix est un invité qui
peut être n'importe lequel, ou personne. Un choix n'est jamais vu deux fois
dans une même rencontre, même conjointe, et ses alternatives ne sont pas
membres d'un panel, qui se voit entier. Voir
[example/fr/assessors.yaml](example/fr/assessors.yaml).

### Réunions, enregistrements et tournages

**Réunions : tout le monde à la fois.** Avec `together: true`, un candidat
rencontre tous ses recruteurs en un seul entretien, quels que soient les
groupes communs, ou pas du tout : un enregistrement, une partie, une réunion.
Le programme trouve un moment où tous sont libres ; s'il n'y en a pas, la
réunion n'est pas planifiée (et signalée comme impossible), jamais découpée
en morceaux. `duration` sur le candidat donne sa durée. Une réunion seule,
c'est ce que fait un sondage comme Doodle ; le programme aide quand plusieurs
réunions se disputent les mêmes personnes.

```yaml
from: 2026-11-16
to: 2026-11-20
day: 09:00-18:00
recruiters:              # les personnes
  Camille (animatrice):
  Ada: {availability: {2026-11-17: 09:00-12:00, 2026-11-19: 14:00-18:00}}
  Blaise: {availability: {2026-11-19: 09:00-18:00}}
candidates:              # les émissions
  les joies de la recherche opérationnelle:
    meet: [Camille (animatrice), Ada, Blaise]
    together: true
    duration: 2h
  une courte sur la planification:
    meet: [Camille (animatrice), Blaise]
    together: true
    duration: 30m
```

L'animatrice est de chaque émission : comme un « recruteur » ne fait qu'une
chose à la fois, deux enregistrements ne se chevauchent jamais, pas plus que
pour les invités communs à plusieurs émissions. Exemples complets : [les
enregistrements d'une émission](example/fr/show.yaml), [des parties de jeu de
rôle](example/fr/roleplay.yaml) dont la meneuse de jeu anime deux tables.

**Un tournage sur plusieurs jours d'affilée.** Le tournage est un candidat ;
chaque personne y figure une fois par jour (`Ada jour 1`, `Ada jour 2`…, avec
ses disponibilités) ; les personnes d'un même jour forment un jury, pour que
chaque journée réunisse toute son équipe ; les rangs donnent l'ordre des
jours, et les écarts les font se suivre :

```yaml
min-gap: 1d
max-gap: 1d
panels:
  - [Ada jour 1, Blaise jour 1]
  - [Ada jour 2, Blaise jour 2]
candidates:
  le tournage:
    duration: 6h
    meet: {Ada jour 1: 1, Blaise jour 1: 1, Ada jour 2: 2, Blaise jour 2: 2}
```

Sans deux jours d'affilée avec toute l'équipe, le tournage n'est pas
planifié. La limite : pour le programme, `Ada jour 1` n'est pas Ada, donc un
conflit avec un autre enregistrement de la même personne, dans le même
fichier, passe inaperçu.

**Des tournages dans plusieurs villes.** Le tournage d'un épisode est un
candidat, à son lieu, `place`, avec ses rencontres, `meetings` :
l'animatrice et le cadreur, seuls ou avec un invité, tous recruteurs.
L'équipe va d'une ville à l'autre avec les trajets de `travel`, qui peuvent
compter le temps de ranger le matériel : un `min-gap` s'appliquerait aussi
entre deux rencontres d'un même tournage. Voir [l'exemple d'une
tournée](example/fr/tour.yaml).

### Un programme : colloques et jeux

**Un programme : une chose après l'autre.** Pour un colloque dans une seule
salle, le colloque est le candidat et les intervenants ses recruteurs : il
n'est qu'à un endroit à la fois, donc les interventions s'enchaînent. La
`duration` de chaque intervenant est la durée de son intervention, les rangs
mettent l'ouverture en premier et la clôture en dernier, et leurs
disponibilités fixent leurs jours ([exemple complet](example/fr/colloquium.yaml)) :

```yaml
day: 09:00-12:00, 14:00-17:00   # le déjeuner comme un trou dans la journée : interventions avant et après
max-gap: unlimited
objectives: [complete-candidates, earliest]
candidates:
  colloque:
    meet: {ouverture: 1, Ada: 2, Blaise: 2, Grace: 2, clôture: 3}
```

Quelques pièges :
- le déjeuner se met dans `day`, pas dans `breaks` : une pause interdit les
  entretiens d'un candidat de part et d'autre dans la même journée, donc une
  journée n'aurait d'interventions que le matin ou que l'après-midi ;
- l'objectif `candidate-wait` compte le déjeuner comme de l'attente et tasse
  les interventions le matin : ne le mettez pas ;
- tout ou rien : un intervenant impossible à placer laisse tout le colloque
  non planifié (l'affichage dit qui) ; une seule salle.

Les jeux de rôle fonctionnent de même : chaque partie est un candidat, les
joueurs et le meneur ses recruteurs. Voir
[example/fr/roleplay.yaml](example/fr/roleplay.yaml) et
[example/fr/colloquium.yaml](example/fr/colloquium.yaml).

### Sorties et voyages

Pas seulement des entretiens, et pas seulement une journée de travail : le
même fichier planifie [deux jours de voyage en train](example/fr/trip.yaml),
commenté ligne à ligne. Le modèle se retourne : la famille ou le groupe est
un recruteur, qui n'est qu'à un endroit à la fois ; chaque lieu est un
recruteur aussi, dont les disponibilités sont les heures d'ouverture ;
chaque visite est un candidat, une rencontre entre les deux. L'hôtel et les
trains sont `required`, une amie se joint à certaines visites en invitée,
et le déjeuner se prend à l'un de deux endroits.

**`required: true`** fait d'un candidat une contrainte et non plus un
objectif : le programme refuse de rendre un planning sans lui et le nomme
à la place. C'est pour ce autour de quoi la journée se construit et qui
n'a aucun intérêt propre — le train déjà réservé, les clefs rendues à
l'hôtel, la salle à libérer à dix-huit heures. Une `priority` classe ce
qu'on préfère avoir ; ceci dit qu'il n'y a rien à classer, donc un
candidat indispensable reste en dehors du décompte des candidats servis.

**`optional: true`** sur un des recruteurs de `meet` l'invite sans en
avoir besoin : `meet: {Marché: {}, Annick: {optional: true}}`. Le
candidat est complet sans lui, et la rencontre se tient avec les invités
libres à ce moment-là et sans les autres, chacun pour son propre compte.
C'est l'invité qui se joint s'il peut — une amie pendant une sortie, un
second évaluateur s'il est disponible, un manager qui passe sur
l'entretien quand sa journée le permet. Une rencontre n'est jamais
abandonnée pour un invité : elle se tient sans lui quand il ne peut pas
être là à une heure qui convient aux autres. Une fois tenue, l'invité qui
l'a manquée n'est pas redemandé : le candidat est servi, et n'est pas
dans le fichier de ce qui reste (`-rest`). Un invité coûte son temps
comme tout le monde, et `meetings` et `guest-time` sont les seuls
objectifs qui le comptent, combien de fois et combien de temps : il vient
quand l'un d'eux passe avant ceux sur le temps des recruteurs, comme
`meetings` par défaut, et jamais quand aucun des deux n'est parmi les
`objectives`, ce que `-check` signale. Un candidat dont toutes les
rencontres sont facultatives est un bonus : compté dans `meetings`,
jamais parmi les candidats servis.

Dans une visite (`together: true`), un invité peut avoir sa propre durée,
celle de la visite quand il vient : `meet: {Nous: {}, Musée: {}, Guide: {optional: true, duration: 1h30m}}`
avec `duration: 1h` fait une visite d'une heure, d'une heure et demie
avec le guide ; la plus longue ne se tient avec le guide que là où elle
tient. De même pour une alternative d'un choix (le repas qui dure plus au
marché). Une visite dure autant que sa plus longue partie présente : une
telle durée ne peut que l'allonger.

Entre deux plannings aussi bons sur tous les objectifs, celui où un invité ne quitte pas les visites du jour
pour revenir à une suivante est préféré : l'amie est avec nous au marché
et à la cathédrale, pas au marché, absente à la cathédrale et de retour
au musée. Pour l'interdire tout à fait, donnez à l'invité
`continuous: true` parmi les recruteurs : une fois parti des visites du
jour, il ne revient pas ce jour-là (il peut toujours arriver tard ou
partir tôt). Une visite qui a besoin de lui, où il n'est pas facultatif,
compte comme les autres : sans la cathédrale, l'amie qui a les billets du
musée ne peut pas être là pour le musée ; la recherche la garde avec nous,
ou absente jusqu'au musée, jamais les deux.

Chacun d'un choix peut dire où le candidat est vu quand c'est lui qui est
choisi, pour les trajets : `Déjeuner: {meet: {Nous: {}, Bistro: {choice:
déjeuner, place: Vieille ville}, Café: {choice: déjeuner, place: Ville
voisine}}, together: true}` est un déjeuner au bistro ou au café, selon ce
que permettent les trajets d'avant et d'après. Seules les alternatives d'un
choix d'un candidat ont des lieux.

## Objectifs

| Nom | Sens |
|---|---|
| `priorities` | maximise les candidats qui voient tous leurs recruteurs, niveau de priorité par niveau, le plus haut d'abord |
| `urgency` | parmi eux, préfère les candidats à qui il reste le moins de jours (voir plus bas) |
| `complete-candidates` | maximise les candidats qui voient tous leurs recruteurs, quelle que soit leur priorité |
| `guest-time` | maximise le temps que les invités (`optional: true`) passent dans les rencontres où ils viennent : un invité présent pour une longue visite compte plus qu'un autre qui passe pour une courte |
| `meetings` | maximise les rencontres tenues |
| `recruiters` | minimise les recruteurs qui ont au moins un entretien, jour par jour : un recruteur qui vient deux jours compte deux |
| `recruiter-half-days` | minimise les demi-journées (matin, après-midi) où des recruteurs ont un entretien, sommées sur les recruteurs et les jours |
| `interviews` | minimise les entretiens, donc les rencontres groupées quand c'est permis |
| `recruiter-span` | minimise le temps de présence des recruteurs, du début de leur premier entretien à la fin du dernier, jour par jour |
| `recruiter-breaks` | minimise le nombre de fois où un recruteur attend entre deux entretiens d'une même journée, chaque attente comptée une fois quelle qu'en soit la durée |
| `recruiter-idle` | minimise le temps libre à l'intérieur des journées des recruteurs : du premier entretien à la fin du dernier, moins les entretiens eux-mêmes |
| `recruiter-idle-spread` | minimise l'écart entre le recruteur qui a le plus de temps libre à l'intérieur de sa journée et celui qui en a le moins, parmi ceux qui ont au moins deux entretiens ce jour-là |
| `candidate-wait` | minimise l'attente des candidats entre deux entretiens d'une même journée (utile quand `max-gap` n'est pas `0`) |
| `candidate-breaks` | minimise le nombre de fois où un candidat attend entre deux entretiens d'une même journée, chaque attente comptée une fois quelle qu'en soit la durée |
| `earliest` | place les entretiens le plus tôt possible dans la période |
| `day-end` | finit chaque journée le plus tôt possible : réduit la fin du dernier entretien du jour, additionnée sur les jours avec entretiens |
| `span` | termine le plus tôt possible : minimise le temps entre le début du premier entretien et la fin du dernier, sur toute la période, nuits comprises |

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

`urgency` regarde les **jours restants** de chaque candidat : les dates
ouvrées du fichier, à partir du premier jour planifié (ou de `not-before`),
où le candidat et tous ses recruteurs sont disponibles. Un candidat à son
dernier jour passe avant n'importe quel nombre de candidats à qui il reste
deux jours, et ainsi de suite ; quelqu'un disponible tous les jours n'est
jamais urgent. Placé après `priorities`, il ne coûte jamais un candidat : le
nombre de candidats vus reste le plus haut possible, et `urgency` choisit
seulement lesquels, pour que le candidat qui ne peut venir qu'à une date ne
soit pas sacrifié à celui qui en a d'autres.
Les jours restants ne comptent que les dates (on ne vérifie pas que tout le
monde est libre en même temps ce jour-là), et la commande les affiche en
`last day`, `2 days left`…

`candidate-wait` et `recruiter-span` comptent du temps ; `candidate-breaks`
et `recruiter-breaks` comptent des **fois**. Un trou de trois heures est une
pause et trois heures d'attente ; trois trous d'une heure sont trois pauses
et les mêmes trois heures. Choisir laquelle minimiser est un vrai choix :
qui est venu pour la journée préfère une longue pause à trois renvois, et
qui est venu pour une heure préfère attendre moins au total. Les deux
mesures classent ces journées en sens inverse, et c'est pourquoi les deux
existent. Une pause compte une fois quelle qu'en soit la durée, et des
entretiens qui s'enchaînent sans rien entre eux ne sont pas une pause.

`recruiter-idle-spread` parle du partage et non du total : il vaut zéro
quand personne n'attend, et zéro tout autant quand tout le monde attend une
heure. C'est son sens et non un défaut, mais c'est pourquoi il a rarement le
dernier mot : placez `recruiter-idle` après lui, et parmi les journées aussi
bien partagées la recherche garde celle où l'on attend le moins au total.

```yaml
objectives: [meetings, recruiter-idle-spread, recruiter-idle]
```

`recruiter-idle` lui-même est `recruiter-span` moins les entretiens, qui
sont les mêmes où qu'on les place : les deux classent les plannings pareil
tant que les mêmes entretiens ont lieu, et ne se séparent que lorsqu'un
candidat n'est pas servi ou qu'un entretien commun en réunit deux. Prenez
`recruiter-span` sauf si c'est bien le temps libre que vous voulez compter.

La **pause du déjeuner** sépare le matin de l'après-midi : avec plusieurs
pauses, la plus longue ; sans pause, midi. Un recruteur qui n'a d'entretiens
qu'avant le déjeuner compte une demi-journée, des deux côtés deux.

L'ordre change le résultat. Comme toutes les rencontres d'un candidat ont
lieu dans la période, `meetings` en premier favorise les candidats qui
doivent voir beaucoup de recruteurs, `complete-candidates` en premier ceux
qui en voient peu.

## Suivre, arrêter et reprendre

- **Combien de candidats les heures permettent.** Chaque entretien prend du
  temps à un recruteur, donc aucun planning ne sert plus de candidats que
  ce que les plus courts d'entre eux font tenir dans les heures
  disponibles. Le programme le calcule et l'affiche à côté du compte :
  `147/200, at most 154`. Il ignore l'ordre des entretiens, ce qui en fait
  une borne et non un planning, et c'est la seule chose qui dise si une
  réponse est bonne sur une journée où la demande dépasse les heures. Il
  compte aussi les heures de chaque recruteur : si tout le monde doit voir
  Alice, présente deux heures, quatre entretiens d'une demi-heure avec elle,
  c'est tout ce que la journée peut tenir, si libres que soient les autres.
  Là où le compte l'atteint, la réponse est prouvée sans rien énumérer.
- **Suivre** : `-progress` affiche les objectifs à chaque amélioration, avec
  le temps écoulé. `-progress-json` affiche la même chose en un objet JSON
  par ligne, pour un programme : `{"seconds":4.7,"iterations":812345,
  "scores":[{"objective":"day-end","label":"day-end","value":1283,
  "unit":"minutes","direction":"min","proven":false}]}`. C'est ce que lit la
  démo pour tracer ses courbes. Les deux vont sur la sortie d'erreur, donc le
  planning de la sortie standard reste propre.
- **Arrêter** : Ctrl+C arrête le calcul et affiche le meilleur planning trouvé
  jusque-là, comme si le timeout était atteint. Un second Ctrl+C tue le
  programme.
- **Reprendre** : avec `-out plan.yaml`, le planning est aussi écrit dans un
  fichier plan. `-start plan.yaml` repart de lui, pour qu'un calcul
  interrompu ou trop court continue plus tard. `-start` et `-out` peuvent
  désigner le même fichier, remplacé seulement à la fin :

  ```sh
  igiveyouarendezvous -timeout 10m -out plan.yaml problem.yaml
  igiveyouarendezvous -timeout 10m -start plan.yaml -out plan.yaml problem.yaml
  ```

Le fichier plan liste les candidats, chacun **figé** ou non, avec la date de
chaque entretien, et note la période du planning : partir du planning d'une
autre période est refusé.

```yaml
from: 2026-11-09
to: 2026-11-13
candidates:
  candidat A:
    fixed: true
    interviews:
      - date: 2026-11-10
        time: 08:00-09:00
        recruiters: [recruteur 2, recruteur 3]
  candidat B:
    fixed: false
    interviews:
      - date: 2026-11-12
        time: 09:00-09:30
        recruiters: [recruteur 1]
```

- **Un candidat figé** (`fixed: true`) a été convoqué : ses rendez-vous sont
  gardés exactement tels quels, et la recherche planifie autour. Les plannings
  figés sont ajoutés un par un, dans l'ordre du fichier. Un planning qui ne
  tient plus (un recruteur qui n'est plus disponible à cette heure, une
  nouvelle pause, une rencontre ajoutée ou retirée depuis, un chevauchement
  avec un planning ajouté avant…) n'est **pas** gardé : une **alerte** en tête
  de l'affichage dit pourquoi, et ce qu'est devenu le candidat, replanifié
  entièrement (nouveaux horaires, ou pas planifié dans la période). La commande
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

**Replanifier en cours de période.** Mercredi, le candidat X a été recalé
après deux entretiens, A et B sont convoqués, C pas encore, et N arrive. Dans
le fichier plan de la semaine, A et B passent à `fixed: true` et C reste à
`fixed: false` ; dans le fichier problème, X est retiré et N ajouté. Puis,
sur la même période :

```sh
igiveyouarendezvous -start plan.yaml -not-before 2026-11-11 -out plan.yaml problem.yaml
```

Les rendez-vous de A et B restent tels quels, lundi et mardi compris ; rien
de nouveau n'est placé avant mercredi, donc C, dont l'entretien tombait
lundi, et N sont planifiés à partir de mercredi. X apparaît en alerte (code
de sortie 2) : un rappel pour annuler ses rendez-vous restants.

## Comme bibliothèque

```go
import (
	"github.com/FrederickFrance/igiveyouarendezvous/config"
	"github.com/FrederickFrance/igiveyouarendezvous/planning"
)

f, err := config.Load("problem.yaml")             // ou config.Parse(data), YAML ou JSON
p, o, err := f.Problem("2026-11-09", "2026-11-13") // le problème et les options de cette période

ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
defer cancel()

s, err := planning.Search(ctx, p, o, planning.SearchOptions{
	Patience: 10 * time.Second,
	Workers:  runtime.NumCPU(),
	Progress: func(pr planning.Progress) { log.Println(pr.Elapsed, pr.Scores) },
})
// s.Interviews : le planning ; s.Scores : la valeur atteinte sur chaque objectif ;
// s.Unscheduled, s.Impossible : ce qui n'a pas pu être placé.

next, err := f.Next(p, s)          // le fichier problème de la période suivante
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
	Days:   5,                 // une semaine : les heures comptent depuis le minuit de son premier jour
	Closed: []int{2},          // sans le mercredi
	Day:    []planning.Interval{{Start: 8 * time.Hour, End: 17 * time.Hour}},
	Breaks: []planning.Interval{{Start: 12 * time.Hour, End: 13*time.Hour + 30*time.Minute}},
	MinGap: planning.Gap{Days: 3}, // au moins trois jours entre deux entretiens
	MaxGap: planning.Gap{Duration: planning.NoLimit},
}, planning.SearchOptions{Patience: 10 * time.Second})
```

Le jour `d` de la période commence à `d*planning.DayLength` : un entretien
le jeudi à 9:00 commence à `3*planning.DayLength + 9*time.Hour`, et
`Interval.Day` donne son jour. `Request.Rank` donne l'ordre des rencontres,
et `Options.NotBefore` le premier jour des nouveaux entretiens.

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
- **Réunions** : `Candidate.Together` (tout le monde à la fois, ou rien) et
  `Candidate.Duration` (à la place des durées des recruteurs).

| Paquet | Contenu |
|---|---|
| `planning` | le problème, les options, `Search`, `Check` (vérifie un planning indépendamment), `Evaluate`, `Clean` |
| `config` | lit le fichier problème (`Load`, `Parse`, `Problem(from, to)`), écrit celui de la période suivante (`Next`, `Write`) ; écrit et lit les fichiers plan (`WritePlan`, `ReadPlan`), en YAML ou en JSON |

`planning` ne dépend que de la bibliothèque standard.

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

Les tests vérifient aussi, sur 60 petits problèmes aléatoires d'une journée
(avec ou sans pauses, attente, priorités et urgence) et sur 80 petites
périodes aléatoires de deux ou trois jours (écarts en temps ou en jours,
ordres, jours fermés, disponibilités par jour), que la recherche trouve le
même résultat qu'une énumération de tous les plannings possibles. Chaque
planning rendu est revérifié par `planning.Check`.

**Pourquoi pas de solveur exact.** Jusqu'à la v0.3, la bibliothèque proposait
aussi un modèle exact, résolu par
[go-milp](https://github.com/daniel-sullivan/go-milp) (Go pur). Il prouvait
l'optimum pour quelques candidats, mais ne trouvait aucun planning en
plusieurs minutes sur 60 à 100, même avec le planning glouton comme borne. Il
a été retiré en v0.4.0, avec sa dépendance : les solveurs qui prouvent ces
optimums, comme HiGHS, sont écrits en C ou en C++.

## Développement

```sh
go test -race ./...     # tests unitaires, exemples et comparaison exhaustive sur de petits problèmes
golangci-lint run       # linter strict (.golangci.yml), version 2.15 ou plus pour Go 1.27
```

Les tests vérifient les exemples et les fichiers écrits par la commande face
aux schémas JSON, et que les schémas connaissent les mêmes clés et objectifs
que le code. Pour contribuer, signaler un problème ou une vulnérabilité : le
[guide de contribution](CONTRIBUTING.md), en anglais (on peut écrire en
français), et la [politique de sécurité](SECURITY.md).

La CI GitHub (`.github/workflows/ci.yml`) lance les tests avec la
couverture, de nouveau les tests avec le détecteur de concurrence à côté
(sans les comparaisons exhaustives, `-short`), le linter et `osv-scanner` à
chaque push et pull request, et l'analyse à nouveau chaque semaine. Dependabot propose les
mises à jour des modules Go et des actions ; ce qu'il ne voit pas, le Go de
`go.mod` et la version de golangci-lint, un workflow hebdomadaire
(`.github/workflows/versions.yml`) le surveille : il ouvre une issue quand une
version plus récente sort, et la ferme une fois les deux montées. Chaque
dimanche soir, un passage profond (`.github/workflows/deep.yml`,
`scripts/deep.sh`) rejoue les comparaisons avec une recherche exhaustive et
les marches sur des problèmes aléatoires sur des milliers de problèmes,
nouveaux chaque semaine, cherche une centaine de gros problèmes, jusqu'à
deux cents candidats, et dix de mille candidats et plus, où aucun meilleur
planning n'est connu mais où chaque règle doit tenir, et ouvre une issue qui
nomme la graine de celui qui échoue : `DEEP_FROM=<graine> DEEP_SEEDS=1 go test -run <test> ./planning/`
le rejoue. Un tag de version (`git tag
v0.1.0 && git push origin v0.1.0`) publie une version avec les zips prêts à
l'emploi du tutoriel (`.github/workflows/release.yml`, `scripts/package.sh`).

Les changements de chaque version sont dans le [journal des
modifications](CHANGELOG.md) (en anglais) : avant de pousser un tag, ajoutez-y
la section de la version ; la page de la version la reprend comme notes
(`scripts/notes.sh`), et son titre vient de l'en-tête, avec le nom de la
version après la date : `## [0.4.1] - 2026-11-08 "Cinderella" 👠` donne
`v0.4.1 Cinderella 👠` (`scripts/title.sh`).

La [démo en ligne](https://frederickfrance.github.io/igiveyouarendezvous/) est le programme compilé en WebAssembly
(`cmd/igiveyouarendezvous/main_js.go`) derrière une page statique (`web/`),
qui le fait tourner dans un worker : les fichiers qu'il lit et écrit restent
en mémoire (`demo.go`). `scripts/web.sh` la construit dans `dist/web`
(`python3 -m http.server -d dist/web` pour l'essayer), la CI lance les tests
compilés en WebAssembly avec Node.js, et `.github/workflows/pages.yml` la
publie sur GitHub Pages à chaque changement de `main`. Il faut une fois
régler Pages sur « GitHub Actions », dans Settings > Pages.

## Dépendances

| Module | Licence | Utilisé par |
|---|---|---|
| go.yaml.in/yaml/v3 | MIT et Apache-2.0 | `config` |
| github.com/stretchr/testify | MIT | tests seulement |
| github.com/santhosh-tekuri/jsonschema/v6 | Apache-2.0 | tests seulement (les schémas) |

Un binaire distribué doit être accompagné de leurs mentions de copyright et
de licence (et du `NOTICE` de go.yaml.in/yaml).

## Écrire le fichier avec un modèle sur sa propre machine

Rien n'a besoin de sortir de votre ordinateur. Le programme lit le JSON
aussi bien que le YAML — YAML est un sur-ensemble de JSON, donc un fichier
`.json` se charge tel quel — et un modèle qui tourne en local (Ollama,
llama.cpp) peut être contraint à répondre selon un schéma JSON, que le
décodage impose alors : il ne peut pas inventer une clef ni oublier une
durée. Donnez-lui `schema/problem.schema.json` comme schéma et
`prompt/fr.md` comme consignes, puis passez sa réponse directement au
programme.

Ce qu'un schéma ne vérifie pas, c'est le fond : un fichier qui recopie
votre planning au lieu de le modéliser, un lieu écrit de deux façons, une
priorité à l'envers. Lisez `prompt/fr.md` vous-même une fois — c'est court,
et ça nomme les erreurs qu'un modèle commet sur ce format.

## Références

La recherche assemble des méthodes de la littérature, listées ici selon la
norme APA 7 et citées dans la documentation du code là où chacune sert :
accepter un mouvement qui n'est pas pire que le planning d'il y a mille
mouvements (Burke & Bykov, 2017) ; retirer quelques candidats et les remettre
(Schrimpf et al., 2000), choisis parmi des candidats liés, comme ceux qui
partagent un recruteur ou une plage horaire (Shaw, 1998) ; les sortes de
mouvements choisies d'après ce qu'elles ont récemment apporté (Ropke &
Pisinger, 2006) ; des objectifs dans
un ordre strict (Ehrgott, 2005) ; le re-horodatage exact de la journée d'un
recruteur (`-exact`), une programmation dynamique sur les sous-ensembles de
ses entretiens (Held & Karp, 1962) ; et, pour savoir quand aucun planning ne
peut recevoir plus de candidats, un sac à dos des heures de chaque recruteur
(Kellerer et al., 2004).

- Burke, E. K., & Bykov, Y. (2017). The late acceptance hill-climbing
  heuristic. *European Journal of Operational Research, 258*(1), 70–78.
  https://doi.org/10.1016/j.ejor.2016.07.012
- Ehrgott, M. (2005). *Multicriteria optimization* (2nd ed.). Springer.
  https://doi.org/10.1007/3-540-27659-9
- Held, M., & Karp, R. M. (1962). A dynamic programming approach to sequencing
  problems. *Journal of the Society for Industrial and Applied Mathematics,
  10*(1), 196–210. https://doi.org/10.1137/0110015
- Kellerer, H., Pferschy, U., & Pisinger, D. (2004). *Knapsack problems*.
  Springer. https://doi.org/10.1007/978-3-540-24777-7
- Ropke, S., & Pisinger, D. (2006). An adaptive large neighborhood search
  heuristic for the pickup and delivery problem with time windows.
  *Transportation Science, 40*(4), 455–472.
  https://doi.org/10.1287/trsc.1050.0135
- Schrimpf, G., Schneider, J., Stamm-Wilbrandt, H., & Dueck, G. (2000). Record
  breaking optimization results using the ruin and recreate principle.
  *Journal of Computational Physics, 159*(2), 139–171.
  https://doi.org/10.1006/jcph.1999.6413
- Shaw, P. (1998). Using constraint programming and local search methods to
  solve vehicle routing problems. In M. Maher & J.-F. Puget (Eds.),
  *Principles and practice of constraint programming — CP98* (Lecture Notes in
  Computer Science, Vol. 1520, pp. 417–431). Springer.
  https://doi.org/10.1007/3-540-49481-2_30

## Licence

Au choix, [0BSD](LICENSE) ou [WTFPL](WTFPL.txt), version 2 : faites-en ce que
vous voulez, sans condition. Les deux disent la même chose ; la 0BSD est celle
que pkg.go.dev reconnaît, pour qu'il affiche la documentation.

Ce programme est fourni sans aucune garantie, dans la mesure permise par la
loi.
