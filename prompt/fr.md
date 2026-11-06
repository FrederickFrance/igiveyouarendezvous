# Consigne pour un assistant IA

Copiez tout ce qui suit le trait dans un assistant IA (Claude, ChatGPT,
Mistral…), écrivez vos règles à la fin, en français courant, et collez le
fichier qu'il renvoie dans [la démo en ligne](https://frederickfrance.github.io/igiveyouarendezvous/)
ou dans un fichier pour le programme.

**Données personnelles.** Ce que vous tapez part chez le fournisseur de
l'assistant. Utilisez des noms de remplacement (`Candidat 1`, des initiales,
des fonctions) plutôt que les vrais noms, et remettez les vrais noms dans le
fichier ensuite : la démo et le programme, eux, n'envoient rien.

---

Tu écris le fichier problème d'igiveyouarendezvous, un planificateur
d'entretiens, à partir des règles que je décris à la fin en français courant.
Le format est YAML. Son schéma JSON documente chaque clé :
https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json
et un exemple commenté :
https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/example/fr/problem.yaml

Le format, en bref (`recruiters` et `candidates` sont obligatoires, le reste
a des valeurs par défaut) :

- `from`, `to` : la période, AAAA-MM-JJ, les deux jours inclus. `weekdays` :
  les jours de la semaine avec entretiens, `[mon, tue, wed, thu, fri]` par
  défaut. `closed` : les dates sans entretien, comme les jours fériés.
- `day` : les horaires de tout le monde, chaque jour, comme `08:30-17:30`.
  Par défaut, seulement `08:00-12:00` : écris-le toujours. `breaks` : aucun
  entretien pendant, comme le déjeuner `12:00-13:30`.
- `duration` : la durée par défaut d'un entretien (`20m`, `45m`, `1h`,
  `1h30`). Aussi par recruteur, `recruiters.<nom>.duration`, ou par candidat,
  `candidates.<nom>.duration` (elle remplace alors celles des recruteurs).
- `min-gap`, `max-gap` : le temps entre la fin d'un entretien d'un candidat
  et le début du suivant. Une durée (`30m`, `2h`), des jours de calendrier
  (`3d`), ou pour `max-gap` seulement, `unlimited`.
  - les deux à `0`, par défaut : à la suite, le même jour ;
  - « le même jour, rapprochés » : `min-gap: 0`, `max-gap: 1h` ;
  - « des jours différents, au moins 2 jours d'écart » : `min-gap: 2d`,
    `max-gap: unlimited` ;
  - « n'importe quand dans la période » : `min-gap: 0`,
    `max-gap: unlimited`.

  Les écarts s'appliquent entre deux entretiens consécutifs quelconques d'un
  candidat. Quand j'en donne un pour certains seulement (« les entretiens
  techniques deux jours après celui de RH »), choisis le réglage le plus
  proche et dis après le fichier ce que ça change pour les autres.
- `availability` d'un recruteur ou d'un candidat : la même chaque jour
  (`09:00-12:00, 14:00-17:00`), `none` (absent), ou par date, une table des
  dates vers les plages horaires ; une personne donnée par date est absente
  les autres dates. Sans `availability` : les horaires de `day`. Quelqu'un
  d'absent quelques jours : liste par date tous ses jours de présence ; les
  seuls jours d'absence, avec `none`, le rendraient absent toute la période.
- `candidates.<nom>.meet` : les recruteurs à rencontrer, en liste dans
  n'importe quel ordre, `[Anne, Bruno]`, ou avec des rangs pour imposer un
  ordre, `{Anne: 1, Bruno: 2, Carl: X}` : le rang 1 d'abord, puis le
  rang 2…, `X` n'importe quand.
- `candidates.<nom>.priority` : `1` par défaut ; plus il est grand, plus le
  candidat compte quand tout le monde ne tient pas.
- `joint-groups` : des listes de recruteurs qui PEUVENT recevoir un candidat
  ensemble, en un seul entretien ; le programme décide. `max-joint-size` : le
  nombre maximal de recruteurs dans un tel entretien.
- `panels` : des listes de recruteurs qui reçoivent TOUJOURS ensemble, comme
  un jury : les membres qu'un candidat doit voir, il les voit en un seul
  entretien. Un recruteur appartient à un jury au plus, et alors à aucun
  groupe commun.
- `candidates.<nom>.together: true` : une seule réunion avec tous les
  recruteurs de `meet` à la fois (une émission, une séance d'atelier) ; pas
  de rangs alors. D'autres usages reprennent les mêmes mots : les invités
  sont des « recruteurs », les épisodes ou séances des « candidats ».

Comment l'écrire :

1. La première ligne est
   `# yaml-language-server: $schema=https://raw.githubusercontent.com/FrederickFrance/igiveyouarendezvous/main/schema/problem.schema.json`
2. Les clés exactement comme ci-dessus, en anglais ; les noms des personnes
   exactement comme je les donne, écrits pareil partout.
3. Les heures en HH:MM sur 24 heures, les dates en AAAA-MM-JJ. Transforme
   les jours que je nomme (« mardi », « le 17 ») en dates de la période ;
   sans année, prends la prochaine date correspondante et dis-le.
4. Une clé par ligne (style bloc). Des accolades seulement pour `meet` ; des
   listes courtes entre crochets, c'est bien. Jamais plusieurs plages
   horaires entre accolades.
5. Chaque nom de `meet`, `joint-groups` et `panels` figure aussi sous
   `recruiters` ; un recruteur sans réglage s'écrit `Nom:` seul.
6. N'invente aucune règle : ce que je ne dis pas garde sa valeur par défaut.
   S'il manque une information nécessaire (la période, les horaires, qui
   rencontre qui), demande-la-moi avant d'écrire le fichier.
7. Un court commentaire (`# …`) en français avant chaque règle que j'ai
   donnée, pour que je puisse vérifier le fichier.
8. Après le fichier, liste en quelques lignes ce que tu as supposé ou
   interprété.

Si je colle un message d'erreur du programme (il donne le numéro de ligne),
corrige le fichier et redonne-le en entier.

Mes règles :

