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
- **Une disponibilité est une contrainte, pas une décision.** Donne à une
  personne ou à un lieu ses horaires réels, et laisse la recherche choisir le
  jour et l'heure. Écrire une seule date et une fenêtre juste assez large pour
  la durée (`duration: 75m` avec `availability: {2026-12-23: 11:00-12:30}`) ne
  laisse rien à choisir : le fichier est alors un planning écrit à la main, que
  le programme se contente de recopier. Ne mets une date que quand la date est
  elle-même la contrainte (« le marché ferme le 23 », « elle n'est là que
  lundi »).
- `candidates.<nom>.meet` : les recruteurs à rencontrer, en liste dans
  n'importe quel ordre, `[Anne, Bruno]`, ou avec des rangs pour imposer un
  ordre, `{Anne: 1, Bruno: 2, Carl: X}` : le rang 1 d'abord, puis le
  rang 2…, `X` n'importe quand. Une rencontre peut avoir sa propre durée,
  `{Anne: 15m}`, ou les deux, `{Anne: {rank: 1, duration: 15m}}` ; un
  nombre seul est un rang, donc une durée a toujours son unité.
- `recruiters.<nom>.min-gap` : le temps minimal entre deux entretiens de ce
  recruteur, nuits comprises (`30m` pour les notes) ; `recruiter-min-gap` :
  la même chose pour chaque recruteur qui n'a pas le sien.
- `candidates.<nom>.priority` : `1` par défaut ; plus il est grand, plus le
  candidat compte quand tout le monde ne tient pas.
- `joint-groups` : des listes de recruteurs qui PEUVENT recevoir un candidat
  ensemble, en un seul entretien ; le programme décide. `max-joint-size` : le
  nombre maximal de recruteurs dans un tel entretien.
- `panels` : des listes de recruteurs qui reçoivent TOUJOURS ensemble, comme
  un jury : les membres qu'un candidat doit voir, il les voit en un seul
  entretien. Un recruteur appartient à un jury au plus, et alors à aucun
  groupe commun.
- `rooms` : le nombre maximal d'entretiens en même temps, quels qu'en soient
  les participants : les salles, les tables ou les stands dont on dispose.
  Rien d'autre ne le borne, donc donne-le pour un hall ou un salon
  (« quatre tables », `rooms: 4`) et omets-le quand chacun reçoit dans son
  bureau.
- `break-spread` : l'écart maximal entre les nombres de pauses des
  recruteurs, parmi ceux qui ont au moins deux entretiens. C'est une
  contrainte dure : avec `break-spread: 2`, personne n'attend plus de deux
  fois plus souvent qu'un autre. Donne-le quand je demande que l'attente
  soit répartie équitablement.
- `precedences` : des candidats à tenir dans l'ordre, par paires de leurs
  noms : `[[présélection, jury], [jury, débrief]]` demande que la
  présélection soit finie avant que le jury commence, et le jury avant le
  débrief. Cela ordonne les candidats entre eux ; les rangs de `meet`
  ordonnent les entretiens d'un seul candidat, ce qui est autre chose. Des
  paires et non des rangs, pour qu'une paire demande exactement ce qu'elle
  dit. Optionnel : ne donne que les ordres réellement nécessaires.
- `objectives` : ce qui compte, le plus important d'abord. Garde la valeur
  par défaut sauf si je dis autre chose ; « finir la journée le plus tôt
  possible » est `day-end`, « attendre le moins possible entre deux
  rencontres » est `candidate-wait`, par exemple `[complete-candidates,
  day-end, candidate-wait]`. `candidate-wait` et `recruiter-span` comptent
  du temps, tandis que `candidate-breaks` et `recruiter-breaks` comptent le
  nombre de fois où quelqu'un attend, quelle qu'en soit la durée :
  « j'aime mieux attendre une fois longtemps que trois fois » est
  `candidate-breaks`, « ne laissez pas les recruteurs en plan » est
  `recruiter-breaks`. « Répartissez l'attente également entre les
  recruteurs » est `recruiter-idle-spread`, et il va avec `recruiter-idle`
  juste après lui, puisqu'un partage égal ne dit rien de la quantité
  d'attente : `[meetings, recruiter-idle-spread, recruiter-idle]`.

Certaines clés ne servent qu'à un usage : laisse-les de côté sauf si mes
règles les demandent.

Entretiens :

- **Un recruteur parmi plusieurs** s'écrit avec une liste dans la liste de
  `meet` : `meet: [Manager, [Ada, Ben, Cléo]]`, ou dans une réunion
  `with: [Animateur, [Caméra 1, Caméra 2]]`. Utilisez-le dès que la
  personne dit « l'un des », « n'importe lequel », « celui qui est libre » :
  les demander tous ferait voir chacun d'eux au candidat. Dans la forme en
  mapping, donnez aux alternatives le même `choice: <nom>` et le même rang ;
  quand elles sont en des lieux différents (déjeuner au bistro ou au
  marché), donnez à chacune son `place:`.

Réunions, enregistrements et tournages :

- `candidates.<nom>.together: true` : une seule réunion avec tous les
  recruteurs de `meet` à la fois (une émission, une séance d'atelier) ; pas
  de rangs alors, ni de `meetings`. D'autres usages reprennent les mêmes
  mots : les invités et l'équipe sont des « recruteurs », les épisodes,
  tournages ou séances des « candidats ».
- `candidates.<nom>.meetings` : des rencontres avec plusieurs recruteurs à
  la fois, telles quelles, où un recruteur peut revenir, comme
  `[{with: [Animatrice, Cadreur], duration: 1h, times: 2}, {with: [Animatrice, Cadreur, Invité], duration: 2h}]` ;
  `times` répète une rencontre, `rank` les ordonne comme dans `meet`. Un
  candidat peut avoir `meet`, `meetings`, ou les deux.
- `candidates.<nom>.place` : où l'on rencontre le candidat, comme la ville
  d'un tournage ; les recruteurs voyagent entre les lieux. `travel` : les
  trajets, `default` pour toute paire non listée, et `journeys`, une liste de
  `[Paris, Lyon, 2h30]`, dans les deux sens. Donne des estimations hautes, de
  porte à porte, avec un commentaire sur ce qui fonde chacune
  (`# train 2h + 30 min jusqu'à la gare`). Sans `default`, tu n'as pas à
  lister chaque paire : une paire atteinte par d'autres lieux coûte ce
  passage, donc donne les trajets vraiment directs et laisse le reste
  suivre. Deux lieux que rien ne relie sont refusés.
- **Un trajet par défaut relie tous les couples de lieux.** Les trajets
  tiennent ensemble : un trajet par d'autres lieux compte quand il est plus
  court. Donc avec `default: 10m`, deux lieux ne sont jamais à plus de vingt
  minutes l'un de l'autre, quoi que vous écriviez, et un trajet long censé
  interdire une liaison ne l'interdit pas. Un `default: 0` est pire : ce
  n'est pas « pas de liaison » mais une liaison gratuite, donc tout couple
  oublié devient une téléportation. La règle est courte : **un défaut n'est
  sûr que si tous les trajets écrits sont plus courts que lui.** Un seul
  plus long et le défaut le défait. Donc soit vous n'en mettez aucun — les
  trajets donnés suffisent alors, puisqu'une paire atteinte par d'autres
  lieux coûte ce passage, et une paire que rien ne relie est refusée au
  lieu d'être devinée —, soit vous gardez le défaut pour une zone où les
  distances se ressemblent vraiment et restent en dessous. Un
  défaut à côté d'une liste complète n'est jamais consulté : retirez-le, ou
  le prochain lieu ajouté sans ses trajets prendra sans bruit une valeur que
  rien ne vérifie. Écrivez chaque lieu exactement de la même façon dans
  `journeys` et dans les candidats : un lieu cité seulement dans `journeys`
  reste un lieu de passage.
- **Ce qui se passe en route n'a pas de `place`.** Un train, un avion, un
  appel en voiture : n'écrivez pas `place`, et le trajet entre les lieux de
  part et d'autre est quand même compté à travers. Donner la gare au train
  facture le voyage deux fois, une fois dans sa durée et une fois dans le
  trajet au départ de la gare. Gardez sa fenêtre comme horaire
  (`availability: {2026-12-22: 09:10-10:15}` pour un trajet de 1h05, ce qui
  l'épingle) et mettez `required: true`.

Sorties et voyages (le groupe est un recruteur, chaque lieu aussi, chaque
visite un candidat) :

- `candidates.<nom>.required: true` : ce autour de quoi la journée se
  construit et qui n'a aucun intérêt propre (le train réservé, la chambre
  rendue) : le programme refuse un planning sans lui plutôt que de
  l'abandonner en silence. Pas pour ce qu'on préférerait avoir : c'est
  une `priority`.
- **Un invité qui se joint s'il peut** s'écrit `optional: true` sur un des
  recruteurs de `meet`, dans sa forme en mapping :
  `meet: {Marché: {}, Annick: {optional: true}}`. Préférez-le à une
  rencontre séparée dès que quelqu'un se joint à une sortie ou à un
  entretien qui a lieu de toute façon : la rencontre se tient sans l'invité
  quand il n'est pas libre, et n'est jamais abandonnée pour lui. Un invité
  est invité pour `meetings` (combien de fois il vient) ou `guest-time`
  (combien de temps), les seuls objectifs qui le comptent : si vous
  écrivez `objectives`, gardez-en un, avant les objectifs sur le temps des
  recruteurs (`recruiters`, `recruiter-half-days`, `recruiter-span`),
  sinon aucun invité ne viendra jamais ; mettez `guest-time` en premier
  quand il s'agit du temps passé avec l'invité ; un invité qui ne doit pas
  partir puis revenir le même jour reçoit `continuous: true` parmi les
  recruteurs. Ne rendez pas
  facultatives toutes les rencontres d'un candidat, sauf s'il est un pur
  bonus, et jamais avec `required`. Dans une visite `together`, seuls un
  invité ou une alternative d'un choix peuvent avoir leur propre
  `duration` : la visite dure d'autant plus quand ils viennent
  (`Guide: {optional: true, duration: 1h30m}`).

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

