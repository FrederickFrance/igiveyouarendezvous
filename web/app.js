"use strict";

// The online demo: the problem file is edited here, and planned by the Go
// program compiled to WebAssembly, in a worker (worker.js). Nothing leaves
// the browser; only the draft is kept, in its storage.

const examples = [
  { file: "problem.yaml", en: "A week of interviews", fr: "Une semaine d'entretiens" },
  { file: "simple.yaml", en: "A day of interviews", fr: "Une journée d'entretiens" },
  { file: "panel.yaml", en: "A selection panel", fr: "Une commission de recrutement" },
  { file: "show.yaml", en: "Recordings of a show", fr: "Les enregistrements d'une émission" },
  { file: "colloquium.yaml", en: "A colloquium", fr: "Un colloque" },
  { file: "roleplay.yaml", en: "Role-playing games", fr: "Des parties de jeu de rôle" },
  { file: "tour.yaml", en: "A tour of shoots in five cities", fr: "Une tournée de tournages dans cinq villes" },
];

const texts = {
  en: {
    tagline: "Plan interviews, recordings or a colloquium, right here in your browser.",
    privacy: "The planner runs on this computer: nothing you type is sent anywhere.",
    source: "Source code",
    problemHeading: "Your problem",
    examplesLabel: "Examples",
    editorLabel: "Problem file",
    editorHelp: "Who must meet whom, and when everybody is free, in YAML (or JSON). The comments of the examples explain every key. Tab indents; Esc then Tab leaves the editor.",
    startSummary: "Start from a plan",
    startHelp: "To plan again around appointments already sent: paste a plan file, import one, or take the last schedule computed here. Candidates with fixed: true keep their appointments exactly; the others may move.",
    useLast: "Take the last schedule",
    importFile: "Import a file",
    clearStart: "Clear",
    startLabel: "Plan file",
    fixAll: "Fix every appointment of this plan",
    notBefore: "No new interview before",
    from: "From",
    to: "To",
    timeLimit: "Search for at most",
    periodHelp: "Leave the dates empty to plan the period of the file. The search also stops when it no longer improves.",
    plan: "Plan",
    stop: "Stop",
    scheduleHeading: "The schedule",
    loading: "Loading the planner…",
    loadFailed: (m) => `The planner could not be loaded (${m}). Reload the page; if it persists, download the program.`,
    idle: "Choose an example or write your problem, then plan.",
    searching: (s, best) => (best ? `Searching for ${s} s. Best so far: ${best}…` : `Searching for ${s} s…`),
    done: (s) => `Finished after ${s} s.`,
    stoppedKept: "Stopped: here is the best schedule found.",
    stoppedNothing: "Stopped before any schedule was found.",
    error: "The problem cannot be planned:",
    errorLine: "The line is selected in the editor.",
    alerts: "Some fixed appointments no longer hold: the details are at the top of the text output.",
    legend: "One row per person, one colour per candidate. Hatched: a fixed appointment.",
    oneDay: "The day",
    allPlanned: "All meetings are planned.",
    planLater: "To plan later:",
    absent: "Impossible, available at no time of the period:",
    impossible: "Impossible, never free together long enough:",
    empty: "No interview could be planned.",
    textSummary: "Text output of the program",
    downloadsLabel: "Download:",
    dlSchedule: "the schedule (text)",
    dlTable: "the table (CSV, for a spreadsheet)",
    dlPlan: "the plan (JSON)",
    dlRest: "the problem left for later (YAML)",
    dlProblem: "the problem (YAML)",
    contactTitle: "A problem, an idea?",
    contactHelp:
      "Copy the link of your case, or download its file, and send it to me: by a message on LinkedIn, or in an issue on GitHub if you have an account. The link and the file hold the whole problem: use made-up names, an issue is public.",
    copyLink: "Copy the link of my case",
    downloadProblem: "Download my file",
    writeLinkedIn: "Write to me on LinkedIn",
    openIssue: "Open an issue on GitHub",
    linkCopied: "The link is copied: paste it in your message. It opens this page with your problem.",
    linkShown: "Copy this link: it opens this page with your problem.",
    linkLong: "The link is long: some messages cut it. Send the file as well.",
    linkFailed: "This browser cannot make the link: download the file instead.",
    linkLoaded: "The problem comes from the link you opened.",
    linkBroken: "The link is damaged or incomplete: the problem could not be read.",
    limitsTitle: "What this demo does not do",
    limitCores: "It searches on one processor core, for at most 5 minutes. The program uses every core, for as long as you want, and does better on large problems.",
    limitFiles: "It reads and writes no file on your computer: plans and results are imported and downloaded. Only your draft is kept, in this browser.",
    limitProgram: "For regular use, in scripts or with Ctrl+C to stop, download the program: same files, same results.",
    download: "Download the program",
    documentation: "Documentation",
    exampleFailed: (f) => `The example ${f} could not be loaded.`,
    importFailed: "This file could not be read.",
  },
  fr: {
    tagline: "Planifier des entretiens, des enregistrements ou un colloque, ici, dans votre navigateur.",
    privacy: "Le calcul se fait sur cet ordinateur : rien de ce que vous tapez n'est envoyé.",
    source: "Code source",
    problemHeading: "Votre problème",
    examplesLabel: "Exemples",
    editorLabel: "Fichier problème",
    editorHelp: "Qui doit rencontrer qui, et quand chacun est libre, en YAML (ou JSON). Les commentaires des exemples expliquent chaque clé. Tab indente ; Échap puis Tab quitte l'éditeur.",
    startSummary: "Partir d'un plan",
    startHelp: "Pour replanifier autour de rendez-vous déjà envoyés : collez un fichier plan, importez-en un, ou reprenez le dernier planning calculé ici. Les candidats en fixed: true gardent exactement leurs rendez-vous ; les autres peuvent bouger.",
    useLast: "Reprendre le dernier planning",
    importFile: "Importer un fichier",
    clearStart: "Effacer",
    startLabel: "Fichier plan",
    fixAll: "Figer tous les rendez-vous de ce plan",
    notBefore: "Aucun nouvel entretien avant le",
    from: "Du",
    to: "Au",
    timeLimit: "Chercher au plus",
    periodHelp: "Laissez les dates vides pour planifier la période du fichier. La recherche s'arrête aussi quand elle ne s'améliore plus.",
    plan: "Planifier",
    stop: "Arrêter",
    scheduleHeading: "Le planning",
    loading: "Chargement du planificateur…",
    loadFailed: (m) => `Le planificateur n'a pas pu être chargé (${m}). Rechargez la page ; si ça persiste, téléchargez le programme.`,
    idle: "Choisissez un exemple ou écrivez votre problème, puis planifiez.",
    searching: (s, best) => (best ? `Recherche depuis ${s} s. Meilleur résultat : ${best}…` : `Recherche depuis ${s} s…`),
    done: (s) => `Terminé en ${s} s.`,
    stoppedKept: "Arrêté : voici le meilleur planning trouvé.",
    stoppedNothing: "Arrêté avant qu'un planning soit trouvé.",
    error: "Le problème ne peut pas être planifié :",
    errorLine: "La ligne est sélectionnée dans l'éditeur.",
    alerts: "Des rendez-vous figés ne tiennent plus : le détail est en tête de la sortie texte.",
    legend: "Une ligne par personne, une couleur par candidat. Hachuré : un rendez-vous figé.",
    oneDay: "La journée",
    allPlanned: "Toutes les rencontres sont planifiées.",
    planLater: "À planifier plus tard :",
    absent: "Impossible, disponible à aucun moment de la période :",
    impossible: "Impossible, jamais libres ensemble assez longtemps :",
    empty: "Aucun entretien n'a pu être planifié.",
    textSummary: "Sortie texte du programme",
    downloadsLabel: "Télécharger :",
    dlSchedule: "le planning (texte)",
    dlTable: "le tableau (CSV, pour un tableur)",
    dlPlan: "le plan (JSON)",
    dlRest: "le problème restant (YAML)",
    dlProblem: "le problème (YAML)",
    contactTitle: "Un souci ? Une idée ?",
    contactHelp:
      "Copiez le lien de votre cas, ou téléchargez son fichier, et envoyez-le-moi : par un message sur LinkedIn, ou dans une issue sur GitHub si vous avez un compte. Le lien et le fichier contiennent tout le problème : utilisez des noms inventés, une issue est publique.",
    copyLink: "Copier le lien de mon cas",
    downloadProblem: "Télécharger mon fichier",
    writeLinkedIn: "M'écrire sur LinkedIn",
    openIssue: "Ouvrir une issue sur GitHub",
    linkCopied: "Le lien est copié : collez-le dans votre message. Il ouvre cette page avec votre problème.",
    linkShown: "Copiez ce lien : il ouvre cette page avec votre problème.",
    linkLong: "Le lien est long : certaines messageries le coupent. Envoyez aussi le fichier.",
    linkFailed: "Ce navigateur ne sait pas fabriquer le lien : téléchargez plutôt le fichier.",
    linkLoaded: "Le problème vient du lien que vous avez ouvert.",
    linkBroken: "Le lien est abîmé ou incomplet : le problème n'a pas pu être lu.",
    limitsTitle: "Ce que la démo ne fait pas",
    limitCores: "Elle cherche sur un seul cœur du processeur, 5 minutes au plus. Le programme utilise tous les cœurs, aussi longtemps que vous voulez, et fait mieux sur les gros problèmes.",
    limitFiles: "Elle ne lit ni n'écrit aucun fichier de votre ordinateur : les plans et les résultats s'importent et se téléchargent. Seul votre brouillon est gardé, dans ce navigateur.",
    limitProgram: "Pour un usage régulier, dans des scripts ou avec Ctrl+C pour arrêter, téléchargez le programme : mêmes fichiers, mêmes résultats.",
    download: "Télécharger le programme",
    documentation: "Documentation",
    exampleFailed: (f) => `L'exemple ${f} n'a pas pu être chargé.`,
    importFailed: "Ce fichier n'a pas pu être lu.",
  },
};

// Browser storage may be missing or refused: the page works without it.
const store = {
  get(key) {
    try {
      return localStorage.getItem(key);
    } catch {
      return null;
    }
  },
  set(key, value) {
    try {
      localStorage.setItem(key, value);
    } catch {
      // nothing to keep
    }
  },
};

const $ = (id) => document.getElementById(id);

let lang = store.get("igy.lang") || ((navigator.language || "en").toLowerCase().startsWith("fr") ? "fr" : "en");

function t(key, ...args) {
  const value = texts[lang][key] ?? texts.en[key];
  return typeof value === "function" ? value(...args) : value;
}

// The state of the page.
let worker = null;
let ready = false;
let running = false;
let startedAt = 0;
let ticker = 0;
let status = { key: "loading", args: [] };
let progressText = "";
let best = null; // {text, plan, table, rest}: the best schedule so far, while searching
let lastPlan = ""; // the plan of the last schedule, for "take the last schedule"
let currentExample = store.get("igy.example") || examples[0].file;
let loadedExample = ""; // the example as loaded, to follow the language while untouched
let loading = 0; // the last example asked for: an earlier one that arrives late is ignored
let downloadURLs = [];

// ---------- texts and language

function applyTexts() {
  document.documentElement.lang = lang;
  for (const element of document.querySelectorAll("[data-i18n]")) {
    element.textContent = t(element.dataset.i18n);
  }
  for (const button of document.querySelectorAll("[data-lang]")) {
    button.setAttribute("aria-pressed", String(button.dataset.lang === lang));
  }
  renderExamples();
  renderStatus();
  updateButtons();
  if (best) {
    renderBoard(best.plan);
    renderDownloads(best);
  }
}

function setLanguage(next) {
  const untouched = $("problem").value === loadedExample;

  lang = next;
  store.set("igy.lang", next);
  applyTexts();

  if (untouched) {
    loadExample(currentExample); // the same example, in the new language
  }
}

// ---------- examples and editor

function renderExamples() {
  const list = document.querySelector(".example-list");
  list.replaceChildren(
    ...examples.map((example) => {
      const button = document.createElement("button");
      button.type = "button";
      button.textContent = example[lang];
      button.setAttribute("aria-pressed", String(example.file === currentExample));
      button.addEventListener("click", () => loadExample(example.file));
      return button;
    }),
  );
}

async function loadExample(file) {
  const ticket = ++loading;

  try {
    const response = await fetch(lang === "fr" ? `examples/fr/${file}` : `examples/${file}`);
    if (!response.ok) {
      throw new Error(response.statusText);
    }
    const text = await response.text();
    if (ticket !== loading) {
      return; // another example was asked for meanwhile
    }
    $("problem").value = text;
    loadedExample = $("problem").value;
    currentExample = file;
    store.set("igy.example", file);
    store.set("igy.problem", $("problem").value);
    renderExamples();
  } catch {
    showError(t("exampleFailed", file));
  }
}

function setupEditor(editor) {
  let leaving = false; // Esc then Tab leaves the editor, for keyboard users

  editor.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      leaving = true;
      return;
    }
    if (event.key === "Tab" && !event.shiftKey && !event.ctrlKey && !event.altKey && !event.metaKey && !leaving) {
      event.preventDefault();
      editor.setRangeText("  ", editor.selectionStart, editor.selectionEnd, "end");
      editor.dispatchEvent(new Event("input"));
    }
    leaving = false;
  });
}

function selectLine(editor, n) {
  const lines = editor.value.split("\n");
  if (n < 1 || n > lines.length) {
    return;
  }
  const start = lines.slice(0, n - 1).reduce((sum, line) => sum + line.length + 1, 0);
  editor.focus();
  editor.setSelectionRange(start, start + lines[n - 1].length);
  const lineHeight = parseFloat(getComputedStyle(editor).lineHeight) || 20;
  editor.scrollTop = Math.max(0, (n - 4) * lineHeight);
}

// ---------- the worker

function startWorker() {
  ready = false;
  worker = new Worker("worker.js");
  worker.onmessage = onMessage;
  worker.onerror = (event) => {
    event.preventDefault();
    setStatus("loadFailed", event.message || "worker");
  };
  updateButtons();
}

function onMessage(event) {
  const message = event.data;

  switch (message.type) {
    case "ready":
      ready = true;
      if (status.key === "loading") {
        setStatus("idle");
      }
      updateButtons();
      break;
    case "failed":
      setStatus("loadFailed", message.message);
      break;
    case "line":
      onLine(message.line);
      break;
    case "best":
      best = { text: message.text, plan: message.plan, table: message.table, rest: "" };
      showSchedule(best);
      break;
    case "done":
      finish(message.result);
      break;
    default:
  }
}

// onLine handles a line printed on the standard error output: a progress
// line ("   4.7 s  priority 1 56, meetings 96, ...") or a note.
function onLine(line) {
  const progress = /^\s*\d+(?:\.\d+)?\s+s\s{2}(.*)$/.exec(line);
  if (progress) {
    progressText = progress[1].split(", ").slice(0, 3).join(", ");
    renderStatus();
    return;
  }
  if (line.trim() !== "") {
    addNote(line);
  }
}

// ---------- planning

function plan() {
  if (!ready || running) {
    return;
  }

  clearResult();
  running = true;
  progressText = "";
  startedAt = performance.now();

  const limit = Number($("time-limit").value);
  const patience = Math.min(30, Math.max(5, Math.round(limit / 6)));
  // The table separates its columns as a spreadsheet of the language expects:
  // Excel in French reads semicolons.
  const args = ["-timeout", `${limit}s`, "-patience", `${patience}s`, "-progress", "-out", "plan.json", "-rest", "rest.yaml",
    "-csv", "schedule.csv", "-csv-separator", lang === "fr" ? ";" : ","];

  if ($("from").value) {
    args.push("-from", $("from").value);
  }
  if ($("to").value) {
    args.push("-to", $("to").value);
  }

  const inputs = {};
  const start = $("start-plan").value;
  if (start.trim() !== "") {
    inputs["start.plan"] = start;
    args.push("-start", "start.plan");
    if ($("fix-all").checked) {
      args.push("-fix-all");
    }
  }
  if ($("not-before").value) {
    args.push("-not-before", $("not-before").value);
  }

  worker.postMessage({ problem: $("problem").value, args, inputs });

  ticker = setInterval(renderStatus, 500);
  setStatus("searching");
  updateButtons();
}

function elapsed() {
  return ((performance.now() - startedAt) / 1000).toFixed(0);
}

function finish(result) {
  running = false;
  clearInterval(ticker);

  if (result.error) {
    best = null;
    clearSchedule();
    showError(result.error);
    setStatus("idle");
    updateButtons();
    return;
  }

  const files = result.files || {};
  lastPlan = files["plan.json"] || "";
  best = { text: result.output, plan: lastPlan, table: files["schedule.csv"] || "", rest: files["rest.yaml"] || "" };
  showSchedule(best);

  if (result.alerts) {
    addNote(t("alerts"));
    $("output-box").open = true;
  }

  setStatus("done", elapsed());
  updateButtons();
}

function stop() {
  if (!running) {
    return;
  }

  worker.terminate(); // the search cannot be told to stop: it is restarted
  running = false;
  clearInterval(ticker);

  if (best) {
    lastPlan = best.plan;
    setStatus("stoppedKept");
  } else {
    setStatus("stoppedNothing");
  }

  startWorker();
  updateButtons();
}

// ---------- result

function setStatus(key, ...args) {
  status = { key, args };
  renderStatus();
}

function renderStatus() {
  const element = $("status");
  element.textContent = status.key === "searching" ? t("searching", elapsed(), progressText) : t(status.key, ...status.args);
  element.classList.toggle("is-searching", status.key === "searching");
  element.classList.toggle("is-failed", status.key === "loadFailed");
}

function updateButtons() {
  const button = $("plan");
  button.disabled = !ready && !running;
  button.textContent = running ? t("stop") : t("plan");
  button.classList.toggle("is-stop", running);
  $("use-last").disabled = running || lastPlan === "";
}

function clearResult() {
  $("error").hidden = true;
  $("notes").replaceChildren();
  $("notes").hidden = true;
  best = null;
}

function clearSchedule() {
  $("board").replaceChildren();
  $("legend").hidden = true;
  $("summary").hidden = true;
  $("downloads").hidden = true;
  $("output-box").hidden = true;
}

function showError(message) {
  const box = $("error");
  const line = /line (\d+)/.exec(message);

  box.replaceChildren();
  const title = document.createElement("strong");
  title.textContent = t("error");
  const text = document.createElement("span");
  text.textContent = ` ${message}`;
  box.append(title, text);

  if (line) {
    const hint = document.createElement("span");
    hint.textContent = ` ${t("errorLine")}`;
    box.append(hint);
    selectLine($("problem"), Number(line[1]));
  }

  box.hidden = false;
}

function addNote(text) {
  const item = document.createElement("li");
  item.textContent = text;
  $("notes").append(item);
  $("notes").hidden = false;
}

function showSchedule(result) {
  const { text, plan } = result;
  renderBoard(plan);

  // The conclusion of the program, in the language of the page.
  const summary = text
    .split("\n")
    .map((line) => {
      if (line === "All meetings are planned.") {
        return t("allPlanned");
      }
      if (line.startsWith("To plan later:")) {
        return `${t("planLater")}${line.slice("To plan later:".length)}`;
      }
      for (const [key, prefix] of [
        ["absent", "Impossible, available at no time of the period:"],
        ["impossible", "Impossible, never free together long enough:"],
      ]) {
        if (line.startsWith(prefix)) {
          return `${t(key)}${line.slice(prefix.length)}`;
        }
      }
      return "";
    })
    .filter((line) => line !== "")
    .join(" ");
  $("summary").textContent = summary;
  $("summary").hidden = summary === "";

  $("output").textContent = text;
  $("output-box").hidden = false;

  renderDownloads(result);
}

// ---------- the board: one row per person, a strip per interview

const minutes = (clock) => {
  const [h, m] = clock.split(":").map(Number);
  return h * 60 + m;
};

function formatDay(date) {
  if (!date) {
    return t("oneDay");
  }
  const day = new Date(`${date}T00:00:00`);
  return new Intl.DateTimeFormat(lang, { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(day);
}

function renderBoard(planText) {
  const board = $("board");
  board.replaceChildren();

  let plan;
  try {
    plan = JSON.parse(planText);
  } catch {
    return;
  }

  const candidates = Object.keys(plan.candidates || {});
  const days = new Map();

  for (const [candidate, entry] of Object.entries(plan.candidates || {})) {
    for (const interview of entry.interviews || []) {
      const [start, end] = interview.time.split("-").map(minutes);
      const key = interview.date || "";
      if (!days.has(key)) {
        days.set(key, []);
      }
      days.get(key).push({ candidate, start, end, recruiters: interview.recruiters, fixed: entry.fixed });
    }
  }

  $("legend").hidden = days.size === 0;

  if (days.size === 0) {
    const empty = document.createElement("p");
    empty.className = "empty";
    empty.textContent = t("empty");
    board.append(empty);
    return;
  }

  for (const key of [...days.keys()].sort()) {
    board.append(renderDay(key, days.get(key), candidates));
  }
}

function renderDay(key, interviews, candidates) {
  const section = document.createElement("section");
  section.className = "day";

  const heading = document.createElement("h3");
  heading.textContent = formatDay(key);
  section.append(heading);

  const from = Math.floor(Math.min(...interviews.map((iv) => iv.start)) / 60) * 60;
  const to = Math.ceil(Math.max(...interviews.map((iv) => iv.end)) / 60) * 60;
  const span = Math.max(60, to - from);
  const position = (m) => `${((m - from) / span) * 100}%`;

  // The people of the day, the earliest first.
  const people = [];
  for (const iv of [...interviews].sort((a, b) => a.start - b.start)) {
    for (const person of iv.recruiters) {
      if (!people.includes(person)) {
        people.push(person);
      }
    }
  }

  const grid = document.createElement("div");
  grid.className = "day-grid";
  grid.style.setProperty("--hours", String(span / 60));

  const scale = document.createElement("div");
  scale.className = "scale";
  scale.setAttribute("aria-hidden", "true");
  for (let m = from; m <= from + span; m += 60) {
    const tick = document.createElement("span");
    tick.style.left = position(m);
    tick.textContent = `${String(Math.floor(m / 60)).padStart(2, "0")}:00`;
    scale.append(tick);
  }
  grid.append(document.createElement("span"), scale);

  for (const person of people) {
    const name = document.createElement("div");
    name.className = "person";
    name.textContent = person;

    const track = document.createElement("div");
    track.className = "track";

    for (const iv of interviews.filter((i) => i.recruiters.includes(person))) {
      const strip = document.createElement("div");
      strip.className = iv.fixed ? "strip is-fixed" : "strip";
      strip.style.left = position(iv.start);
      strip.style.width = `${((iv.end - iv.start) / span) * 100}%`;
      strip.style.setProperty("--strip", `var(--candidate-${candidates.indexOf(iv.candidate) % 8})`);
      strip.textContent = iv.candidate;

      const clock = (m) => `${String(Math.floor(m / 60)).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`;
      strip.title = `${iv.candidate}, ${clock(iv.start)}-${clock(iv.end)}, ${iv.recruiters.join(" + ")}`;
      track.append(strip);
    }

    grid.append(name, track);
  }

  const scroller = document.createElement("div");
  scroller.className = "day-scroll";
  scroller.append(grid);
  section.append(scroller);

  return section;
}

// ---------- downloads

function renderDownloads({ text, plan, table, rest }) {
  for (const url of downloadURLs) {
    URL.revokeObjectURL(url);
  }
  downloadURLs = [];

  const box = $("downloads");
  box.replaceChildren();

  const label = document.createElement("span");
  label.textContent = t("downloadsLabel");
  box.append(label);

  const files = [
    ["dlSchedule", "schedule.txt", text, "text/plain"],
    ["dlTable", "schedule.csv", table, "text/csv"],
    ["dlPlan", "plan.json", plan, "application/json"],
    ["dlRest", "rest.yaml", rest, "text/yaml"],
    ["dlProblem", "problem.yaml", $("problem").value, "text/yaml"],
  ];

  for (const [key, name, content, type] of files) {
    if (!content) {
      continue;
    }
    const url = URL.createObjectURL(new Blob([content], { type: `${type};charset=utf-8` }));
    downloadURLs.push(url);

    const link = document.createElement("a");
    link.href = url;
    link.download = name;
    link.textContent = t(key);
    box.append(link);
  }

  box.hidden = false;
}

// ---------- start plan

async function importPlan(input) {
  const [file] = input.files;
  if (!file) {
    return;
  }
  try {
    $("start-plan").value = await file.text();
  } catch {
    showError(t("importFailed"));
  }
  input.value = "";
}

// ---------- the link of a case: the problem, compressed, after the #, which
// the browser never sends to the server

const linkVersion = "v1.";
const maxLinkData = 200_000; // characters after the #
const maxCaseSize = 2_000_000; // bytes, once inflated

function toBase64URL(bytes) {
  let binary = "";
  for (let i = 0; i < bytes.length; i += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
  }
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

function fromBase64URL(text) {
  const binary = atob(text.replaceAll("-", "+").replaceAll("_", "/"));
  return Uint8Array.from(binary, (c) => c.charCodeAt(0));
}

async function encodeCase() {
  const data = JSON.stringify({ problem: $("problem").value, from: $("from").value, to: $("to").value });
  const stream = new Blob([data]).stream().pipeThrough(new CompressionStream("deflate-raw"));
  return linkVersion + toBase64URL(new Uint8Array(await new Response(stream).arrayBuffer()));
}

// decodeCase reads a case, refusing what would inflate beyond maxCaseSize.
async function decodeCase(text) {
  if (!text.startsWith(linkVersion) || text.length > maxLinkData) {
    throw new Error("not a case");
  }
  const reader = new Blob([fromBase64URL(text.slice(linkVersion.length))])
    .stream()
    .pipeThrough(new DecompressionStream("deflate-raw"))
    .getReader();
  const chunks = [];
  let size = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) {
      break;
    }
    size += value.length;
    if (size > maxCaseSize) {
      await reader.cancel();
      throw new Error("too large");
    }
    chunks.push(value);
  }
  const data = JSON.parse(await new Blob(chunks).text());
  if (typeof data.problem !== "string") {
    throw new Error("no problem");
  }
  const date = (value) => (typeof value === "string" && /^\d{4}-\d{2}-\d{2}$/.test(value) ? value : "");
  return { problem: data.problem, from: date(data.from), to: date(data.to) };
}

function showContactStatus(key) {
  $("contact-status").textContent = t(key);
  $("contact-status").hidden = false;
}

async function copyCaseLink() {
  let url;
  try {
    url = `${location.origin}${location.pathname}#${await encodeCase()}`;
  } catch {
    showContactStatus("linkFailed");
    return;
  }

  const field = $("case-link");
  field.value = url;
  field.hidden = false;

  let copied = false;
  try {
    await navigator.clipboard.writeText(url);
    copied = true;
  } catch {
    field.select(); // the clipboard is refused: the link is there to copy
  }
  showContactStatus(copied ? "linkCopied" : "linkShown");
  if (url.length > 8000) {
    $("contact-status").textContent += ` ${t("linkLong")}`;
  }
}

function downloadProblem() {
  const url = URL.createObjectURL(new Blob([$("problem").value], { type: "text/yaml;charset=utf-8" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = "problem.yaml";
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// loadCaseFromLink puts the case of the link in the editor, then removes it
// from the address, so that a reload keeps the edits; false without a
// readable case.
async function loadCaseFromLink() {
  const hash = location.hash.slice(1);
  if (!hash.startsWith(linkVersion)) {
    return false;
  }
  history.replaceState(null, "", `${location.pathname}${location.search}`);

  try {
    const found = await decodeCase(hash);
    $("problem").value = found.problem;
    $("from").value = found.from;
    $("to").value = found.to;
    loadedExample = "";
    currentExample = "";
    store.set("igy.problem", found.problem);
    store.set("igy.example", "");
    renderExamples();
    addNote(t("linkLoaded"));
    return true;
  } catch {
    addNote(t("linkBroken")); // the draft, or an example, instead
    return false;
  }
}

// ---------- start

document.addEventListener("DOMContentLoaded", async () => {
  for (const button of document.querySelectorAll("[data-lang]")) {
    button.addEventListener("click", () => setLanguage(button.dataset.lang));
  }

  const editor = $("problem");
  setupEditor(editor);
  setupEditor($("start-plan"));

  let saving = 0;
  editor.addEventListener("input", () => {
    clearTimeout(saving);
    saving = setTimeout(() => store.set("igy.problem", editor.value), 400);
  });

  $("plan").addEventListener("click", () => (running ? stop() : plan()));
  $("use-last").addEventListener("click", () => {
    $("start-plan").value = lastPlan;
  });
  $("clear-start").addEventListener("click", () => {
    $("start-plan").value = "";
    $("fix-all").checked = false;
    $("not-before").value = "";
  });
  $("import-plan").addEventListener("change", (event) => importPlan(event.target));
  $("copy-link").addEventListener("click", copyCaseLink);
  $("download-problem").addEventListener("click", downloadProblem);

  applyTexts();
  startWorker();

  if (await loadCaseFromLink()) {
    return;
  }

  const draft = store.get("igy.problem");
  if (draft) {
    editor.value = draft;
  } else {
    loadExample(currentExample || examples[0].file);
  }
});
