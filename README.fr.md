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
de rôle… voir [Autres usages](#autres-usages).

**[Essayer en ligne](https://frederickfrance.github.io/igiveyouarendezvous/)** : modifiez un exemple et planifiez-le, dans
votre navigateur, sans rien installer. Tout se calcule sur votre ordinateur :
ce que vous tapez n'est envoyé nulle part.

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

La commande affiche le planning par heure (avec les dates, sur plusieurs
jours), l'agenda de chaque recruteur jour par jour, la valeur atteinte sur
chaque objectif et les rencontres restantes. Elle signale aussi les
rencontres **impossibles** : un candidat et un recruteur jamais libres
ensemble assez longtemps. Elles restent dans le fichier problème de la
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
| `objectives` | voir plus bas | objectifs, le plus important d'abord |
| `recruiters.<nom>.duration` | `duration` | durée des entretiens avec ce recruteur ; un entretien commun dure autant que sa plus longue partie |
| `recruiters.<nom>.availability` | `day` | plages du recruteur (`none` : absent), chaque jour ou par date (voir plus bas) |
| `candidates.<nom>.meet` | obligatoire | les recruteurs à rencontrer, dans n'importe quel ordre, ou avec leurs rangs (voir plus bas) |
| `candidates.<nom>.availability` | `day` | plages du candidat, chaque jour ou par date |
| `candidates.<nom>.priority` | `1` | plus c'est haut, plus c'est important (objectif `priorities`) |
| `candidates.<nom>.duration` | celle des recruteurs | durée des entretiens de ce candidat, à la place de celles des recruteurs |
| `candidates.<nom>.together` | `false` | `true` : une réunion, tous les recruteurs à la fois en un seul entretien, ou rien (voir [Autres usages](#autres-usages)) |

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
Quand d'autres en ont une, c'est peut-être un oubli : la commande les liste
dans une note. Une personne dont la disponibilité est donnée par date est
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

**L'ordre.** `meet` est une liste, dans n'importe quel ordre, ou les
recruteurs avec leurs rangs : le rang 1 d'abord, puis le rang 2, et ainsi de
suite ; `X` (ou `any`, ou rien) à n'importe quel moment. Avec
`{recruteur 1: 1, recruteur 2: X, recruteur 3: 2}`, le recruteur 1 passe
avant le recruteur 3, et le recruteur 2 avant, entre ou après. Plusieurs
recruteurs peuvent avoir le même rang, dans n'importe quel ordre entre eux ;
un entretien commun ne mélange jamais deux rangs.

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

## Autres usages

Le programme place des « candidats » qui doivent rencontrer des
« recruteurs », chacun à un seul endroit à la fois. Bien d'autres plannings
entrent dans ce moule : les noms des clés restent, seul leur sens change.

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

## Objectifs

| Nom | Sens |
|---|---|
| `priorities` | maximise les candidats qui voient tous leurs recruteurs, niveau de priorité par niveau, le plus haut d'abord |
| `urgency` | parmi eux, préfère les candidats à qui il reste le moins de jours (voir plus bas) |
| `complete-candidates` | maximise les candidats qui voient tous leurs recruteurs, quelle que soit leur priorité |
| `meetings` | maximise les rencontres tenues |
| `recruiters` | minimise les recruteurs qui ont au moins un entretien, jour par jour : un recruteur qui vient deux jours compte deux |
| `recruiter-half-days` | minimise les demi-journées (matin, après-midi) où des recruteurs ont un entretien, sommées sur les recruteurs et les jours |
| `interviews` | minimise les entretiens, donc les rencontres groupées quand c'est permis |
| `recruiter-span` | minimise le temps de présence des recruteurs, du début de leur premier entretien à la fin du dernier, jour par jour |
| `candidate-wait` | minimise l'attente des candidats entre deux entretiens d'une même journée (utile quand `max-gap` n'est pas `0`) |
| `earliest` | place les entretiens le plus tôt possible dans la période |

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

La **pause du déjeuner** sépare le matin de l'après-midi : avec plusieurs
pauses, la plus longue ; sans pause, midi. Un recruteur qui n'a d'entretiens
qu'avant le déjeuner compte une demi-journée, des deux côtés deux.

L'ordre change le résultat. Comme toutes les rencontres d'un candidat ont
lieu dans la période, `meetings` en premier favorise les candidats qui
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

La CI GitHub (`.github/workflows/ci.yml`) lance les tests avec le détecteur
de concurrence et la couverture, le linter et `osv-scanner` à chaque push et
pull request, et l'analyse à nouveau chaque semaine. Dependabot propose les
mises à jour des modules Go et des actions. Un tag de version (`git tag
v0.1.0 && git push origin v0.1.0`) publie une version avec les zips prêts à
l'emploi du tutoriel (`.github/workflows/release.yml`, `scripts/package.sh`).

Les changements de chaque version sont dans le [journal des
modifications](CHANGELOG.md) (en anglais) : avant de pousser un tag, ajoutez-y
la section de la version ; la page de la version la reprend comme notes
(`scripts/notes.sh`).

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

## Licence

Au choix, [0BSD](LICENSE) ou [WTFPL](WTFPL.txt), version 2 : faites-en ce que
vous voulez, sans condition. Les deux disent la même chose ; la 0BSD est celle
que pkg.go.dev reconnaît, pour qu'il affiche la documentation.

Ce programme est fourni sans aucune garantie, dans la mesure permise par la
loi.
