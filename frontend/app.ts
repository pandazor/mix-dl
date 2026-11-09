import { t, message, setLanguage, loadCatalog, locale, language, languages, formatBytes } from "./i18n.js";
type Filter = {
    images: boolean;
    videos: boolean;
    extensions: string[];
};
type Profile = {
    id: string;
    site: string;
    name: string;
    method: string;
    state: string;
};
type Site = {
    id: string;
    name: string;
    methods: string[];
    guide: string;
    url: string;
};
type Job = {
    destinationMode?: string;
    id: string;
    url: string;
    site: string;
    destination: string;
    layout: string;
    filename: string;
    profileId: string;
    profileMode: string;
    status: string;
    added: string;
    started: string;
    ended: string;
    error: string;
    exitCode: number;
    bytes: number;
    files: number;
    skipped: number;
    current: string;
    order: number;
};
type SiteSettings = {
    destination: string;
    layout: string;
    filename: string;
    profileId: string;
    filters?: Filter;
};
type CustomSite = {
    name: string;
    domain: string;
};
type Settings = {
    language?: string;
    customSites?: Record<string, CustomSite>;
    destination: string;
    layout: string;
    filename: string;
    filters: Filter;
    concurrency: number;
    quotaBytes: number;
    rateKiB: number;
    galleryPath: string;
    ytdlpPath: string;
    ffmpegPath: string;
    sites: Record<string, SiteSettings>;
};
type State = {
    settings: Settings;
    jobs: Job[];
    profiles: Profile[];
    usedBytes: number;
    running: boolean;
    active: number;
    lastError: string;
};
type Envelope = {
    state: State;
    sites: Site[];
    dataDir: string;
    version: string;
    platform: string;
};
const $ = <T extends HTMLElement = HTMLElement>(s: string) => document.querySelector<T>(s)!;
const esc = (v: unknown) => String(v ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]!));
const token = $<HTMLMetaElement>('meta[name="dl-token"]').content;
const statuses = (): Record<string, string> => ( { queued: t("En attente"), running: t("En cours"), paused: t("En pause"), interrupted: t("Interrompu"), completed: t("Terminé"), failed: t("Erreur"), authentication_required: t("Connexion requise"), cancelled: t("Annulé"), paused_quota: t("Limite atteinte") });
let data: Envelope;
let page = 'downloads';
let search = '';
let modalJob = '';
let loading: Promise<void> | null = null;
let toastTimer: ReturnType<typeof setTimeout>;
async function api(path: string, body?: unknown): Promise<any> { const r = await fetch('/api/' + path, { method: body === undefined ? 'GET' : 'POST', headers: body === undefined ? {} : { 'Content-Type': 'application/json', 'X-DL-Token': token }, body: body === undefined ? undefined : JSON.stringify(body) }); const v = await r.json(); if (!r.ok)
    throw Error(message(v.error) || t("Erreur {0}", r.status)); return v; }
function toast(s: string, error = false) { const t = $('#toast'); t.textContent = s; t.className = error ? 'error' : ''; t.style.display = 'block'; clearTimeout(toastTimer); toastTimer = setTimeout(() => t.style.display = 'none', error ? 9000 : 4500); }
function run(fn: () => Promise<void>) { fn().catch(e => toast(message(e.message), true)); }
function bytes(n: number) { return formatBytes(n); }
function date(s: string) { return s ? new Date(s).toLocaleString(locale(), { dateStyle: 'short', timeStyle: 'short' }) : '—'; }
function siteName(id: string) { return id === 'generic' ? t('Autres sites') : data.sites.find(s => s.id === id)?.name || id; }
function val(id: string) { return $<HTMLInputElement>('#' + id).value; }
function check(id: string) { return $<HTMLInputElement>('#' + id).checked; }
function opts(values: [
    string,
    string
][], selected: string) { return values.map(([v, label]) => `<option value="${esc(v)}" ${v === selected ? 'selected' : ''}>${esc(label)}</option>`).join(''); }
const layouts = (): [string,string][] => [['flat', t("Dossier de destination")], ['site', t("Site")], ['creator', t("Site / Créateur")]];
const filenames = (): [string,string][] => [['default', t("Nommage du site (identifiants inclus)")], ['id-title', t("Identifiant + titre")]];
function extensions(prefix: string, f: Filter) { return `<div class="row wrap"><label class="check"><input id="${prefix}-images" type="checkbox" ${f.images ? 'checked' : ''}>${t("Images")}</label><label class="check"><input id="${prefix}-videos" type="checkbox" ${f.videos ? 'checked' : ''}>${t("Vidéos")}</label></div><div class="extensions">${['jpg', 'jpeg', 'png', 'webp', 'gif', 'mp4', 'webm', 'mov', 'avi', 'mkv'].map(x => `<label class="extension"><input class="${prefix}-extension" type="checkbox" value="${x}" ${f.extensions.includes(x) ? 'checked' : ''}>${x}</label>`).join('')}</div><label>${t("Extensions supplémentaires (séparées par des virgules)")}<input id="${prefix}-custom" value="${esc(f.extensions.filter(x => !['jpg', 'jpeg', 'png', 'webp', 'gif', 'mp4', 'webm', 'mov', 'avi', 'mkv'].includes(x)).join(', '))}" placeholder="avif, bmp…"></label>`; }
function readFilters(prefix: string): Filter { return { images: check(prefix + '-images'), videos: check(prefix + '-videos'), extensions: [...new Set([...Array.from(document.querySelectorAll<HTMLInputElement>('.' + prefix + '-extension:checked')).map(x => x.value), ...val(prefix + '-custom').split(',').map(x => x.trim().toLowerCase().replace(/^\./, '')).filter(Boolean)])] }; }
function pathInput(id: string, value: string, placeholder = t("Hériter du dossier global"), kind = 'directory') { return `<div class="pathfield"><input id="${id}" value="${esc(value)}" placeholder="${esc(placeholder)}"><button type="button" data-pick="${id}" data-kind="${kind}" title="${esc(t("Parcourir"))}">…</button></div>`; }
async function load(fresh = false) { if (loading) {
    await loading;
    if (fresh)
        return load(true);
    return;
} loading = (async () => { data = await api('state'); setLanguage(data.state.settings.language || 'fr'); if (!$('#main'))
    shell(); updateChrome(); if (page === 'downloads' || page === 'history')
    updateJobs(); if (modalJob)
    await refreshLog(); })(); try {
    await loading;
}
finally {
    loading = null;
} }
function shell() { $('#app').innerHTML = `<div class="layout"><aside class="sidebar"><div class="brand"><span class="logo">↓</span><span class="brand-text">dl-manager</span></div><nav class="nav" aria-label="${esc(t("Navigation principale"))}">${[['downloads', '↓', t("Téléchargements")], ['history', '◷', t("Historique")], ['accounts', '♙', t("Comptes & sites")], ['settings', '⚙', t("Paramètres")]].map(([id, icon, label]) => `<button data-page="${id}" class="${page === id ? 'active' : ''}"><span class="nav-icon">${icon}</span><span class="nav-label">${label}</span></button>`).join('')}</nav><div class="side-bottom"><div>${t("VOTRE FILE, À VOTRE RYTHME")}</div><div id="side-state">${t("En pause")}</div><button id="hide-app">${t("Réduire en arrière-plan")}</button><button class="ghost" id="quit-app">${t("Quitter l'application")}</button><div class="section-gap">${t("Version {0} · {1}", esc(data.version), data.platform === 'linux' ? 'Linux' : 'Windows')}</div></div></aside><main class="main" id="main"></main></div>`; renderPage(); }
function header(title: string, sub: string, actions = '') { return `<header class="page-head"><div><div class="eyebrow">${t("DL-MANAGER / {0}", page === 'downloads' ? t("FILE D’ATTENTE") : page === 'history' ? t("BIBLIOTHÈQUE") : page === 'accounts' ? t('CONNEXIONS') : t("PRÉFÉRENCES"))}</div><h1>${title}</h1><p>${sub}</p></div><div class="head-actions">${actions}</div></header>`; }
function renderPage() { document.querySelectorAll('[data-page]').forEach(x => x.classList.toggle('active', (x as HTMLElement).dataset.page === page)); if (page === 'downloads')
    renderDownloads();
else if (page === 'history')
    renderHistory();
else if (page === 'accounts')
    renderAccounts();
else
    renderSettings(); updateChrome(); }
function renderDownloads() { const s = data.state.settings; $('#main').innerHTML = header(t("Téléchargements"), t("Ajoutez vos liens. La file s’occupe de la suite."), `<button id="pause-all">${t("Ⅱ Tout mettre en pause")}</button><button class="primary" id="start-all">${t("▶ Démarrer / reprendre")}</button>`) + `<div id="queue-notice"></div><section class="panel"><div class="panel-title"><h2>${t("Ajouter à la file")}</h2><span class="tag">${t("Plusieurs liens acceptés")}</span></div><form id="add-form"><div class="add-row"><textarea id="urls" aria-label="${esc(t("Adresses à télécharger"))}" placeholder="${esc(t("Collez une adresse, ou plusieurs — une par ligne"))}" required></textarea><button class="primary" type="submit">${t("＋ Ajouter")}</button></div><div class="media-row"><span id="detected" class="muted smalltext">${t("Site détecté automatiquement")}</span><span class="divider"></span><span class="muted smalltext">${t("Réglages du site appliqués par défaut")}</span></div><details><summary>${t("Options pour ces téléchargements")}</summary><div class="form-grid"><label class="full">${t("Destination{0}", pathInput('job-dest', '', t("Dossier du site ou dossier global")))}</label><label>${t("Organisation")}<select id="job-layout">${opts([['', t("Réglage du site")], ...layouts()], '')}</select></label><label>${t("Compte")}<select id="job-profile"><option value="inherit">${t("Compte par défaut du site")}</option><option value="">${t("Sans compte")}</option>${data.state.profiles.map(p => `<option value="${p.id}">${esc(siteName(p.site) + ' · ' + p.name)}</option>`).join('')}</select></label><div class="full"><label class="check"><input id="override-media" type="checkbox">${t("Personnaliser les médias pour ces liens")}</label><div id="job-filter-options" class="hidden section-gap">${extensions('job', s.filters)}</div></div></div></details></form></section><div id="stats" class="stats"></div><section class="panel table-panel"><div class="table-toolbar"><h2>${t("File d'attente")} <span id="queue-count" class="muted"></span></h2><button class="small" id="refresh-destinations">${t("Actualiser les destinations en attente")}</button></div><div id="job-table" class="table-wrap"></div></section><div class="bottom-status"><span id="queue-status"></span><span>${t("Les fichiers partiels sont conservés.")}</span></div>`; updateJobs(); }
function renderHistory() { $('#main').innerHTML = header(t("Historique"), t("Retrouvez les tâches terminées, les interruptions et les erreurs."), `<button id="clear-history">${t("Retirer les tâches terminées")}</button>`) + `<section class="panel table-panel"><div class="table-toolbar"><h2>${t("Téléchargements précédents")}</h2><input id="history-search" placeholder="${esc(t("Rechercher une adresse ou un site…"))}" value="${esc(search)}" aria-label="${esc(t("Rechercher dans l'historique"))}"></div><div id="job-table" class="table-wrap"></div></section>`; updateJobs(); }
function updateChrome() { if (!data)
    return; const s = data.state; const el = $('#side-state'); if (el)
    el.textContent = s.running ? t("{0} tâche(s) en cours", s.active) : s.active > 0 ? t("Arrêt en cours…") : t("En pause · reprise manuelle"); const start = $<HTMLButtonElement>('#start-all'); if (start)
    start.disabled = s.active > 0 || !s.jobs.some(j => ['queued', 'paused', 'interrupted', 'paused_quota', 'authentication_required'].includes(j.status)); const pause = $<HTMLButtonElement>('#pause-all'); if (pause)
    pause.disabled = !s.running && s.active === 0; }
function jobErrorMessage(j: Job): string {
    if (!/rate limit|too many requests|\b429\b/i.test(j.error))
        return message(j.error);
    const reset = j.error.match(/rate limit will reset at (\d{2}:\d{2}:\d{2})/i);
    const summary = j.site === 'tumblr' ? t("Quota de l’API Tumblr atteint.") : t("Limite de requêtes du site atteinte.");
    return summary + (reset ? t(" Remise à zéro annoncée par le moteur : ") + reset[1] + t(". Réessayez après cette heure.") : t(" Réessayez plus tard."));
}
function updateJobs() {
    if (!$('#job-table'))
        return;
    const s = data.state;
    const all = s.jobs;
    const activeStatuses = ['queued', 'running', 'paused', 'interrupted', 'paused_quota', 'authentication_required', 'failed'];
    let jobs = page === 'downloads' ? all.filter(j => activeStatuses.includes(j.status)) : all.filter(j => !['queued', 'running'].includes(j.status));
    if (page === 'history' && search)
        jobs = jobs.filter(j => (j.url + ' ' + siteName(j.site)).toLowerCase().includes(search.toLowerCase()));
    $('#job-table').innerHTML = jobs.length ? `<table><thead><tr><th>${t("Source")}</th><th>${t("État")}</th><th>${t("Fichiers")}</th><th>${t("Volume ajouté")}</th><th>${t("Actions")}</th></tr></thead><tbody>${jobs.map(j => `<tr><td class="url"><div class="url-main" title="${esc(j.url)}">${esc(j.url)}</div><div class="url-sub">${esc(siteName(j.site))} · ${esc(j.current || j.destination)}</div></td><td><span class="status ${esc(j.status)}">${esc(statuses()[j.status] || j.status)}</span>${j.error ? `<div class="job-error error" title="${esc(j.error)}">${esc(jobErrorMessage(j))}</div>` : ''}</td><td>${j.files}<span class="muted"> ${t("/ {0} ignoré(s)", j.skipped)}</span></td><td>${bytes(j.bytes)}</td><td><div class="job-actions"><button class="small" data-detail="${j.id}">${t("Détails")}</button>${j.status === 'running' ? `<button class="small" data-action="pause" data-id="${j.id}" title="${esc(t("Mettre en pause"))}">Ⅱ</button>` : `<button class="small" data-action="retry" data-id="${j.id}" title="${esc(t("Remettre en attente"))}">↻</button><button class="small" data-action="up" data-id="${j.id}" title="${esc(t("Monter"))}">↑</button><button class="small" data-action="down" data-id="${j.id}" title="${esc(t("Descendre"))}">↓</button><button class="small danger" data-action="delete" data-id="${j.id}" title="${esc(t("Retirer de la liste"))}">×</button>`}</div></td></tr>`).join('')}</tbody></table>` : `<div class="empty"><div class="empty-icon">${page === 'downloads' ? '↓' : '◷'}</div><h3>${page === 'downloads' ? t("Votre file est prête") : t("Aucun téléchargement ici")}</h3><p class="smalltext">${page === 'downloads' ? t("Collez vos premières adresses ci-dessus, puis cliquez sur Démarrer.") : t("Les tâches terminées et leurs détails apparaîtront ici.")}</p></div>`;
    if (page === 'downloads') {
        const limit = s.settings.quotaBytes;
        $('#stats').innerHTML = `<div class="stat"><div class="label">${t("EN ATTENTE")}</div><div class="value">${all.filter(j => j.status === 'queued').length}</div></div><div class="stat"><div class="label">${t("EN COURS")}</div><div class="value">${s.active}<span class="muted" style="font-size:13px"> / ${s.settings.concurrency}</span></div></div><div class="stat"><div class="label">${t("TERMINÉS")}</div><div class="value">${all.filter(j => j.status === 'completed').length}</div></div><div class="stat"><div class="row"><div class="label">${t("VOLUME DEPUIS LA REMISE À ZÉRO")}</div><div class="spacer"></div><button class="small ghost" id="reset-quota">${t("Réinitialiser")}</button></div><div class="value">${bytes(s.usedBytes)}<span class="muted" style="font-size:12px"> / ${limit ? bytes(limit) : t("sans limite")}</span></div><div class="meter ${limit && s.usedBytes >= limit ? 'limit' : ''}"><span style="width:${limit ? Math.min(100, s.usedBytes / limit * 100) : 0}%"></span></div></div>`;
        $('#queue-count').textContent = `(${jobs.length})`;
        $('#queue-status').textContent = s.running ? t("● File active — vous pouvez ajouter d’autres liens") : t("○ File en pause — aucun démarrage automatique");
        $('#queue-notice').innerHTML = s.lastError ? `<div class="notice">${esc(message(s.lastError))}</div>` : limit && s.usedBytes >= limit ? `<div class="notice">${t("Limite de volume atteinte. Réinitialisez le compteur ou augmentez la limite, puis cliquez sur Reprendre.")}</div>` : !s.settings.destination ? `<div class="notice">${t("Choisissez votre dossier global dans Paramètres, ou une destination dans les options de téléchargement.")}</div>` : '';
    }
}
function renderAccounts() { $('#main').innerHTML = header(t("Comptes & sites"), t("Réglez une fois la destination et la connexion de chaque plateforme."), `<button id="custom-site-add" class="primary">${t("＋ Ajouter un site")}</button><button id="backup-import">${t("Importer une sauvegarde")}</button><button id="backup-export">${t("Exporter les sites et comptes")}</button>`) + `<div class="site-grid">${data.sites.map(s => { const n = data.state.profiles.filter(p => p.site === s.id).length; return `<button class="site-card" data-site="${s.id}"><span class="site-icon">${esc(siteName(s.id).slice(0, 2))}</span><span><h3>${esc(siteName(s.id))}</h3>${data.state.settings.customSites?.[s.id] ? `<p class="site-domain">${esc(data.state.settings.customSites[s.id].domain)}</p>` : ''}<p>${n ? n + t(" compte(s) enregistré(s)") : t("Public ou à configurer")}</p></span></button>`; }).join('')}</div><section class="panel section-gap"><h2>${t("Vos comptes")}</h2>${data.state.profiles.length ? data.state.profiles.map(p => `<div class="profile-row"><div><h3>${esc(p.name)}</h3><p class="muted smalltext">${esc(siteName(p.site))} · ${esc(p.method)} · ${p.state === 'authentication_required' ? t("Connexion à renouveler") : t("Configuré")}</p></div><div class="row"><button class="small" data-profile-check="${p.id}">${t("Vérifier le stockage")}</button><button class="small" data-profile-edit="${p.id}">${t("Modifier")}</button><button class="small danger" data-profile-delete="${p.id}">${t("Supprimer")}</button></div></div>`).join('') : `<p class="muted section-gap">${t("Aucun compte enregistré. Les sites publics restent accessibles sans compte.")}</p>`}<p class="footer-note">${t("Les connexions se font dans votre navigateur. Les secrets importés sont protégés par le coffre de votre session. Pour changer de PC ou d’OS, utilisez une sauvegarde chiffrée avec une phrase secrète.")}</p></section>`; }
function customSiteForm(id = '', name = '', domain = '') { return `<form id="custom-site-form" data-site-id="${esc(id)}"><div class="form-grid"><label>${t("Nom du site")}<input id="custom-name" required maxlength="100" value="${esc(name)}" placeholder="${esc(t("Mes illustrations"))}"></label><label>${t("Domaine")}<input id="custom-domain" required maxlength="253" value="${esc(domain)}" placeholder="exemple.org" ${id ? 'readonly' : ''}></label></div><p class="footer-note">${id ? t("Le domaine reste fixe pour préserver les comptes et les reprises.") : t("Indiquez le domaine seul, sans https:// ni chemin. Ses sous-domaines seront reconnus. Le site doit être compatible avec gallery-dl.")}</p><div class="settings-footer">${id ? `<button type="button" class="danger" id="custom-site-delete" data-site-id="${esc(id)}">${t("Supprimer le site")}</button>` : ''}<button class="primary">${id ? t("Enregistrer le nom") : t("Créer le site")}</button></div></form>`; }
function openCustomSite() { showModal(`<div class="modal-head"><h2>${t("Ajouter un site personnalisé")}</h2><button data-close>×</button></div>${customSiteForm()}`); }
async function openSite(id: string, profile?: Profile) {
    const site = data.sites.find(x => x.id === id)!;
    const s = data.state.settings.sites[id] || { destination: '', layout: '', filename: '', profileId: '' };
    const profiles = data.state.profiles.filter(p => p.site === id);
    const selected = profile?.id || s.profileId || profiles[0]?.id || '';
    showModal(`<div class="modal-head"><h2>${esc(siteName(site.id))}</h2><button data-close>×</button></div>
 ${data.state.settings.customSites?.[id] ? customSiteForm(id, data.state.settings.customSites[id].name, data.state.settings.customSites[id].domain) : ''}
 <form id="site-form" data-site-id="${id}"><h3>${t("Destination et classement")}</h3><div class="form-grid">
 <label class="full check"><input id="site-use-global" type="checkbox" ${!s.destination ? 'checked' : ''}>${t("Utiliser le dossier global")}</label><label class="full">${t("Dossier de téléchargement{0}", pathInput('site-dest', s.destination || '', t("Choisir un dossier propre à ce site")))}</label><p class="full footer-note">${t("Dossier global actuel :")} <code>${esc(data.state.settings.destination)}</code>${t(". Cochez pour suivre ses changements ; décochez pour choisir un dossier propre au site. Les tâches déjà commencées gardent leur dossier.")}</p>
 <label>${t("Organisation")}<select id="site-layout">${opts([['', t("Hériter du réglage global")], ...layouts()], s.layout || '')}</select></label>
 <label>${t("Nommage des fichiers")}<select id="site-filename">${opts([['', t("Hériter du réglage global")], ...filenames()], s.filename || '')}</select></label>
 <label class="full">${t("Compte par défaut pour les tâches héritées")}<select id="site-default">${opts([['', t("Sans compte")], ...profiles.map(p => [p.id, p.name] as [
            string,
            string
        ])], s.profileId || '')}</select></label>
 <div class="full"><label class="check"><input type="checkbox" id="site-media-override" ${s.filters ? 'checked' : ''}>${t("Filtres propres à ce site")}</label><div class="section-gap" id="site-filter-options">${extensions('site', s.filters || data.state.settings.filters)}</div></div></div>
 <div class="settings-footer"><button class="primary">${t("Enregistrer les réglages du site")}</button></div></form>
 <div class="guide">${esc(data.state.settings.customSites?.[id] ? t('Site personnalisé : importez uniquement les cookies de {0}. Les liens de ce domaine et de ses sous-domaines utilisent ces réglages. La compatibilité dépend de gallery-dl.', data.state.settings.customSites[id].domain) : t(site.guide))}<br><button class="small ghost" type="button" data-help="${id}" style="margin-top:10px">${t("Ouvrir le site / la documentation ↗")}</button></div>
 <label>${t("Compte à consulter ou modifier")}<select id="profile-select" data-site-id="${id}">${opts([['', t("Ajouter un nouveau compte")], ...profiles.map(p => [p.id, p.name] as [
            string,
            string
        ])], selected)}</select></label>
 <div id="account-editor" class="section-gap"></div>`);
    updateSiteDestination();
    $('#site-filter-options').classList.toggle('hidden', !s.filters);
    await showProfileEditor(id, selected);
}
async function showProfileEditor(siteId: string, id: string) {
    cancelOAuth();
    const site = data.sites.find(x => x.id === siteId)!;
    const p = data.state.profiles.find(p => p.id === id);
    $('#account-editor').textContent = t("Lecture du compte…");
    const saved = p ? await api('profile?id=' + encodeURIComponent(id)) : {};
    // Ignore responses for a modal/profile that has since been closed or replaced.
    const selector = $<HTMLSelectElement>('#profile-select');
    if (!selector || selector.dataset.siteId !== siteId || selector.value !== id)
        return;
    const isDefault = data.state.settings.sites[siteId]?.profileId || '';
    $('#account-editor').innerHTML = `<form id="profile-form" data-site-id="${siteId}" data-profile-id="${p?.id || ''}">
 <h3>${p ? t("Modifier ") + esc(p.name) : t("Ajouter un compte")}</h3>
 ${p ? `<p class="notice">${t("Compte enregistré. Les champs secrets restent masqués ; laissez-les vides pour conserver leur valeur.")}</p>` : ''}
 <div class="form-grid"><label>${t("Nom du compte")}<input id="profile-name" required value="${esc(p?.name || t("Principal"))}"></label>
 <label>${t("Méthode")}<select id="profile-method">${opts(site.methods.map(m => [m, { api: 'User ID + API Key', cookies: t("Fichier cookies.txt"), token: 'Refresh token', oauth1: 'OAuth Token + Secret' }[m] || m]), p?.method || site.methods[0])}</select></label>
 <label class="full" id="cookie-field">${t("Fichier cookies.txt (Netscape){0}", pathInput('profile-cookie', '', saved.hasCookies ? t("Cookies enregistrés — laisser vide pour les conserver") : t("Sélectionner le fichier exporté"), 'cookies'))}</label>
 <label id="userid-field">User ID<input id="profile-user" value="${esc(saved.userId || '')}" inputmode="numeric" autocomplete="off"></label>
 <label id="token-field">Token / API Key<input id="profile-token" type="password" autocomplete="new-password" placeholder="${saved.hasToken ? t("Clé enregistrée — laisser vide pour la conserver") : t("Saisir la clé")}"></label>
 <label id="secret-field">OAuth Token Secret<input id="profile-secret" type="password" autocomplete="new-password" placeholder="${saved.hasTokenSecret ? t("Secret enregistré — laisser vide pour le conserver") : t("Saisir le secret")}"></label>
${['tumblr', 'deviantart'].includes(siteId) ? `<div id="oauth-controls" class="full"><p class="footer-note">${t("Vous devrez vous connecter dans votre navigateur et autoriser gallery-dl. Les tokens rempliront les champs ci-dessus ; enregistrez ensuite le compte.")}</p><div class="row section-gap"><button type="button" id="oauth-start">${t("Générer le token")}</button><button type="button" id="oauth-cancel" class="hidden">${t("Annuler la connexion")}</button><button type="button" id="oauth-open" class="hidden">${t("Ouvrir le navigateur")}</button></div><input id="oauth-url" class="hidden section-gap" aria-label="${esc(t("Lien de connexion"))}" readonly><p id="oauth-status" class="footer-note" role="status"></p></div>` : ""}
 <label class="full check"><input type="checkbox" id="profile-default" ${(p ? isDefault === id : !isDefault) ? 'checked' : ''}>${t("Utiliser ce compte par défaut pour ce site")}</label></div>
 <p class="footer-note">${t("Le compte par défaut s'applique aussi aux tâches existantes réglées sur « Compte par défaut du site ». Un changement de méthode demande les nouveaux identifiants.")}</p>
 <div class="settings-footer"><button class="primary">${t("Enregistrer le compte")}</button></div></form>`;
    toggleMethod();
}
type OAuthAttempt = {
    id: string;
    form: HTMLFormElement;
    cancelled: boolean;
};
let oauthAttempt: OAuthAttempt | null = null;
function cancelOAuth() { const a = oauthAttempt; if (!a)
    return; oauthAttempt = null; a.cancelled = true; if (a.id)
    api('oauth/cancel', { id: a.id }).catch(() => { }); if (a.form.isConnected) {
    a.form.querySelector<HTMLButtonElement>('#oauth-start')!.disabled = false;
    a.form.querySelector<HTMLButtonElement>('button.primary')!.disabled = false;
    a.form.querySelector('#oauth-cancel')!.classList.add('hidden');
    a.form.querySelector('#oauth-open')!.classList.add('hidden');
    const link = a.form.querySelector<HTMLInputElement>('#oauth-url')!;
    link.value = '';
    link.classList.add('hidden');
} }
async function generateOAuth() {
    const form = $<HTMLFormElement>('#profile-form');
    if (!form || oauthAttempt)
        return;
    const a: OAuthAttempt = { id: '', form, cancelled: false };
    oauthAttempt = a;
    const current = () => oauthAttempt === a && !a.cancelled && form.isConnected;
    const status = form.querySelector<HTMLElement>('#oauth-status')!;
    form.querySelector<HTMLButtonElement>('#oauth-start')!.disabled = true;
    form.querySelector<HTMLButtonElement>('button.primary')!.disabled = true;
    form.querySelector('#oauth-cancel')!.classList.remove('hidden');
    status.textContent = t("Préparation de la connexion…");
    status.className = 'footer-note';
    try {
        const started = await api('oauth/start', { site: form.dataset.siteId });
        a.id = started.id;
        if (!current()) {
            await api('oauth/cancel', { id: a.id });
            return;
        }
        let opened = false;
        while (current()) {
            const v = await api('oauth/status', { id: a.id });
            if (!current())
                return;
            if (v.state === 'failed')
                throw Error(message(v.error) || t("Connexion interrompue."));
            if (v.state === 'completed') {
                const c = await api('oauth/result', { id: a.id });
                if (!current())
                    return;
                $<HTMLInputElement>('#profile-token').value = c.token;
                $<HTMLInputElement>('#profile-secret').value = c.tokenSecret || '';
                status.textContent = t("Token reçu. Cliquez sur Enregistrer le compte pour le conserver.");
                return;
            }
            status.textContent = t("Connectez-vous dans le navigateur et autorisez gallery-dl. Attente du retour…");
            if (v.url && !opened) {
                opened = true;
                const link = form.querySelector<HTMLInputElement>('#oauth-url')!;
                link.value = v.url;
                link.classList.remove('hidden');
                form.querySelector('#oauth-open')!.classList.remove('hidden');
                try {
                    await api('oauth/open', { id: a.id });
                }
                catch {
                    if (current())
                        status.textContent = t("Ouvrez le lien affiché dans votre navigateur pour vous connecter.");
                }
            }
            await new Promise(r => setTimeout(r, 500));
        }
    }
    catch (e) {
        if (current()) {
            status.textContent = (e as Error).message;
            status.className = 'footer-note error';
        }
    }
    finally {
        if (oauthAttempt === a)
            cancelOAuth();
    }
}
function jobProfileMode(j: Job) { return j.profileMode || (j.profileId ? 'explicit' : 'inherit'); }
function jobProfileLabel(j: Job) { const mode = jobProfileMode(j); const id = mode === 'inherit' ? data.state.settings.sites[j.site]?.profileId : j.profileId; const p = data.state.profiles.find(x => x.id === id); return mode === 'none' ? t("Sans compte") : (mode === 'inherit' ? t("Par défaut : ") : '') + (p?.name || t("Sans compte")); }
function toggleMethod() { cancelOAuth(); const m = val('profile-method'); const controls = $('#oauth-controls'); if (controls)
    controls.classList.toggle('hidden', m === 'cookies'); $('#cookie-field').classList.toggle('hidden', m !== 'cookies'); $('#userid-field').classList.toggle('hidden', m !== 'api'); $('#token-field').classList.toggle('hidden', m === 'cookies'); $('#secret-field').classList.toggle('hidden', m !== 'oauth1'); }
let destinationSave: Promise<void> = Promise.resolve();
async function saveGlobalDestination() {
    const input = $<HTMLInputElement>('#global-dest');
    if (!input)
        return destinationSave;
    const destination = input.value.trim();
    const status = $('#global-save-status');
    const pending = destinationSave.catch(() => { }).then(async () => {
        try {
            if (destination !== data.state.settings.destination) {
                if (status.isConnected) {
                    status.className = 'footer-note';
                    status.textContent = t("Enregistrement du dossier global…");
                }
                await api('settings/destination', { destination });
                await load(true);
            }
            if (input.isConnected && input.value.trim() === destination) {
                input.value = destination;
                status.className = 'footer-note';
                status.textContent = t("Dossier global enregistré. Les sites qui en héritent utilisent ce chemin.");
            }
        }
        catch (e) {
            if (status.isConnected) {
                status.className = 'footer-note error';
                status.textContent = t("Dossier non enregistré : ") + (e as Error).message;
            }
            throw e;
        }
    });
    destinationSave = pending;
    return pending;
}
function updateSiteDestination() {
    const input = $<HTMLInputElement>('#site-dest');
    const inherited = check('site-use-global');
    if (inherited) {
        if (!input.disabled)
            input.dataset.customDestination = input.value;
        input.value = data.state.settings.destination;
    }
    else if (input.disabled) {
        input.value = input.dataset.customDestination || '';
    }
    input.disabled = inherited;
    $<HTMLButtonElement>('[data-pick="site-dest"]').disabled = inherited;
}
function renderSettings() { const s = data.state.settings; $('#main').innerHTML = header(t("Paramètres"), t("Le dossier global s’applique aux tâches héritées jamais démarrées. Les autres valeurs s’appliquent aux nouveaux ajouts.")) + `<form id="settings-form"><section class="panel settings-panel"><h2>${t("Langue")}</h2><label class="language-label">${t("Langue")}<select id="language">${opts(languages, language())}</select></label><p class="footer-note">${t("La langue est enregistrée automatiquement.")}</p></section><section class="panel settings-panel"><h2>${t("Destination & classement")}</h2><div class="form-grid"><label class="full">${t("Dossier global de téléchargement{0}", pathInput('global-dest', s.destination, t("Choisir un dossier")))}</label><p id="global-save-status" class="full footer-note" role="status">${t("Le dossier global est enregistré automatiquement après modification. Les autres paramètres se sauvegardent avec le bouton ci-dessous.")}</p><label>${t("Organisation des dossiers")}<select id="global-layout">${opts(layouts(), s.layout)}</select></label><label>${t("Nommage des fichiers")}<select id="global-filename">${opts(filenames(), s.filename)}</select></label><div class="full preview" id="path-preview"></div></div></section><section class="panel settings-panel"><h2>${t("Rythme & limite de volume")}</h2><div class="form-grid"><label>${t("Téléchargements simultanés")}<select id="concurrency">${opts([['1', t("1 tâche")], ['2', t("2 tâches")], ['3', t("3 tâches")]], String(s.concurrency))}</select></label><label>${t("Débit par tâche (Kio/s, 0 = illimité)")}<input id="rate" type="number" min="0" value="${s.rateKiB}"></label><label>${t("Pause après ce volume (Go, 0 = désactivé)")}<input id="quota" type="number" min="0" step="0.1" value="${s.quotaBytes / 1e9}"></label><div><label>${t("Volume actuel")}</label><p class="section-gap">${bytes(data.state.usedBytes)} <button type="button" class="small" id="reset-quota">${t("Réinitialiser")}</button></p></div></div><p class="footer-note">${t("Le compteur est conservé au redémarrage. Le réinitialiser met les tâches en pause et attend votre clic pour reprendre. Un même site et dossier, ou un même compte, est traité par une seule tâche à la fois pour éviter les conflits.")}</p></section><section class="panel settings-panel"><h2>${t("Médias par défaut")}</h2><div class="section-gap">${extensions('global', s.filters)}</div><p class="footer-note">${t("Les extensions filtrent les fichiers disponibles à la source. Aucun transcodage automatique. Les extensions personnalisées inconnues sont classées comme images.")}</p></section><section class="panel settings-panel"><h2>${t("Moteurs")}</h2><div id="engines" class="section-gap muted">${t("Vérification des versions…")}</div><details><summary>${t("Chemins personnalisés (facultatifs)")}</summary><div class="form-grid"><label class="full">gallery-dl${pathInput('gallery-path', s.galleryPath, t("Utiliser le moteur du paquet"), 'engine')}</label><label class="full">yt-dlp${pathInput('ytdlp-path', s.ytdlpPath, t("Utiliser le moteur du paquet"), 'engine')}</label></div></details><p class="footer-note">${t("Données :")} <code>${esc(data.dataDir)}</code><br>${t("Le paquet garde les outils dans tools/ et les données dans data/.")}</p><div class="settings-footer"><button class="primary">${t("Enregistrer les paramètres")}</button></div></section></form>`; preview(); run(async () => { const es = await api('engines'); if (page === 'settings')
    $('#engines').innerHTML = es.map((e: any) => `<div class="engine"><span>${esc(e.name)}</span><span>${e.error ? `<span class="error">${esc(message(e.error))}</span>` : `<code>${esc(e.version)}</code>`}</span></div>`).join(''); }); }
function preview() { if (!$('#path-preview'))
    return; let p = val('global-dest') || t("Destination"); const l = val('global-layout'); if (l === 'site' || l === 'creator')
    p += (data.platform === 'linux' ? '/' : '\\') + 'artstation'; if (l === 'creator')
    p += (data.platform === 'linux' ? '/' : '\\') + 'nom_artiste'; p += (data.platform === 'linux' ? '/' : '\\') + 'identifiant_titre.jpg'; $('#path-preview').textContent = p; }
function showModal(content: string) { cancelOAuth(); modalJob = ''; $('#modal-content').innerHTML = content; const d = $<HTMLDialogElement>('#modal'); if (!d.open)
    d.showModal(); }
function closeModal() { cancelOAuth(); modalJob = ''; $<HTMLDialogElement>('#modal').close(); $('#modal-content').innerHTML = ''; }
async function openDetail(id: string) { const j = data.state.jobs.find(x => x.id === id); if (!j)
    return; showModal(`<div class="modal-head"><h2>${t("Détails du téléchargement")}</h2><button data-close>×</button></div><div class="detail-grid"><span>${t("Source")}</span><span>${esc(j.url)}</span><span>${t("Destination")}</span><span>${esc(j.destination)}<br><small class="muted">${j.started ? t("Dossier conservé pour la reprise") : j.destinationMode === 'inherit' ? t("Dossier du site ou global, actualisé au premier départ") : j.destinationMode === 'explicit' ? t("Dossier personnalisé pour cette tâche") : t("Ancienne tâche : dossier fixé à l’ajout")}</small></span><span>${t("Compte")}</span><span>${esc(jobProfileLabel(j))}</span><span>${t("Ajout")}</span><span>${date(j.added)}</span><span>${t("Dernier démarrage")}</span><span>${date(j.started)}</span><span>${t("État")}</span><span>${t("{0} · code {1}", esc(statuses()[j.status]), j.exitCode)}</span><span>${t("Message")}</span><span class="error">${esc(jobErrorMessage(j) || '—')}</span></div>${j.status !== 'running' ? `<form id="job-profile-form" data-job-id="${j.id}" class="section-gap"><label>${t("Compte pour cette tâche")}<select id="detail-profile">${opts([['inherit', t("Compte par défaut du site")], ['', t("Sans compte")], ...data.state.profiles.filter(p => p.site === j.site || p.site === 'generic').map(p => [p.id, p.name] as [
        string,
        string
    ])], jobProfileMode(j) === 'inherit' ? 'inherit' : jobProfileMode(j) === 'none' ? '' : j.profileId)}</select></label><div class="settings-footer"><button class="primary">${t("Enregistrer le compte de cette tâche")}</button></div></form>` : ''}<h3>${t("Journal")}</h3><pre id="logs" class="logs section-gap">${t("Chargement…")}</pre><div class="settings-footer"><button data-action="cancel" data-id="${j.id}" class="danger">${t("Annuler cette tâche")}</button><button data-close>${t("Fermer")}</button></div>`); modalJob = id; await refreshLog(); }
async function refreshLog() { const id = modalJob; const logs = await api('logs?id=' + encodeURIComponent(id)); if (modalJob === id && $('#logs'))
    $('#logs').textContent = logs.length ? logs.map((l: any) => `${date(l.at)}  ${l.line}`).join('\n') : t("Aucun message pour le moment."); }
document.addEventListener('click', e => {
    const b = (e.target as HTMLElement).closest<HTMLElement>('button');
    if (!b)
        return;
    run(async () => {
        if (b.id === 'custom-site-add') {
            openCustomSite();
            return;
        }
        if (b.id === 'custom-site-delete') {
            await api('custom-site/delete', { id: b.dataset.siteId });
            await load(true);
            closeModal();
            renderAccounts();
            toast(t("Site supprimé."));
            return;
        }
        if (b.id === 'oauth-start') {
            await generateOAuth();
            return;
        }
        if (b.id === 'oauth-cancel') {
            cancelOAuth();
            $('#oauth-status').textContent = t("Connexion annulée.");
            return;
        }
        if (b.id === 'oauth-open') {
            if (oauthAttempt?.id)
                await api('oauth/open', { id: oauthAttempt.id });
            return;
        }
        if (b.dataset.page) {
            if (page === 'settings')
                await saveGlobalDestination();
            page = b.dataset.page;
            renderPage();
            return;
        }
        if (b.hasAttribute('data-close')) {
            closeModal();
            return;
        }
        if (b.dataset.pick) {
            const r = await api('pick', { kind: b.dataset.kind || 'directory' });
            if (r.path) {
                $<HTMLInputElement>('#' + b.dataset.pick).value = r.path;
                if (b.dataset.pick === 'global-dest')
                    await saveGlobalDestination();
            }
            preview();
            return;
        }
        if (b.dataset.action) {
            if (b.dataset.action === 'delete' && !confirm(t("Retirer cette tâche de la liste ? Les fichiers téléchargés seront conservés.")))
                return;
            await api('job', { id: b.dataset.id, action: b.dataset.action });
            await load(true);
            return;
        }
        if (b.dataset.detail) {
            await openDetail(b.dataset.detail);
            return;
        }
        if (b.dataset.site) {
            await openSite(b.dataset.site);
            return;
        }
        if (b.dataset.help) {
            await api('help', { site: b.dataset.help });
            return;
        }
        if (b.dataset.profileEdit) {
            const p = data.state.profiles.find(x => x.id === b.dataset.profileEdit)!;
            await openSite(p.site, p);
            return;
        }
        if (b.dataset.profileDelete) {
            if (!confirm(t("Supprimer ce compte enregistré ?")))
                return;
            await api('profile/delete', { id: b.dataset.profileDelete });
            await load(true);
            renderAccounts();
            return;
        }
        if (b.dataset.profileCheck) {
            const r = await api('profile/validate', { id: b.dataset.profileCheck });
            toast(message(r.message));
            return;
        }
        switch (b.id) {
            case 'backup-pause': {
                b.setAttribute('disabled', '');
                try {
                    await api('queue/pause', {});
                    const until = Date.now() + 30000;
                    do {
                        await load(true);
                        const status = $('#backup-queue-status');
                        if (status)
                            status.textContent = data.state.active ? t("Arrêt des tâches en cours…") : t("File en pause : vous pouvez importer.");
                        if (!data.state.active)
                            break;
                        if (Date.now() > until)
                            throw Error(t("Arrêt toujours en cours. Attendez puis réessayez."));
                        await new Promise(r => setTimeout(r, 200));
                    } while (true);
                }
                finally {
                    b.removeAttribute('disabled');
                }
                return;
            }
            case 'backup-export':
                openBackup('export');
                return;
            case 'backup-import':
                openBackup('import');
                return;
            case 'refresh-destinations': {
                const ids = data.state.jobs.filter(canRefreshDestination).map(j => j.id);
                if (!ids.length) {
                    toast(t("Aucune tâche éligible."));
                    return;
                }
                if (data.state.running || data.state.active)
                    throw Error(t("Mettez la file en pause et attendez l’arrêt des tâches."));
                if (!confirm(t("Appliquer le dossier du site ou le dossier global aux {0} tâches jamais démarrées ? Les tâches commencées et les destinations personnalisées sont conservées.", ids.length)))
                    return;
                for (let i = 0; i < ids.length; i += 500)
                    await api('jobs/destinations', { ids: ids.slice(i, i + 500) });
                toast(t("Destinations actualisées. Reprise manuelle."));
                break;
            }
            case 'start-all':
                await api('queue/start', {});
                break;
            case 'pause-all':
                await api('queue/pause', {});
                break;
            case 'reset-quota':
                if (!confirm(t("Remettre le compteur à zéro ? Les tâches actives seront mises en pause. L’historique et les fichiers restent conservés.")))
                    return;
                await api('quota/reset', {});
                toast(t("Compteur remis à zéro. Reprise uniquement sur votre clic."));
                break;
            case 'clear-history':
                if (!confirm(t("Retirer de l’historique les tâches terminées ? Les fichiers restent conservés.")))
                    return;
                await api('history/clear', {});
                break;
            case 'hide-app':
                await api('window/hide', {});
                return;
            case 'quit-app':
                await api('window/quit', {});
                return;
            default: return;
        }
        await load(true);
        if (page === 'settings')
            renderSettings();
    });
});
document.addEventListener('change', e => { const el = e.target as HTMLInputElement; if (el.id === 'language') { run(() => changeLanguage(el.value)); return; } if (el.id === 'global-dest')
    run(saveGlobalDestination); if (el.id === 'site-use-global')
    updateSiteDestination(); if (el.id === 'override-media')
    $('#job-filter-options').classList.toggle('hidden', !el.checked); if (el.id === 'site-media-override')
    $('#site-filter-options').classList.toggle('hidden', !el.checked); if (el.id === 'profile-method')
    toggleMethod(); if (el.id === 'profile-select')
    run(() => showProfileEditor(el.dataset.siteId!, el.value)); if (el.id === 'site-default' && el.value) {
    const sel = $<HTMLSelectElement>('#profile-select');
    sel.value = el.value;
    run(() => showProfileEditor(sel.dataset.siteId!, el.value));
} if (el.id === 'global-layout')
    preview(); });
document.addEventListener('input', e => { const el = e.target as HTMLInputElement; if (el.id === 'history-search') {
    search = el.value;
    updateJobs();
} if (el.id === 'global-dest') {
    preview();
    const status = $('#global-save-status');
    status.className = 'footer-note';
    status.textContent = t("Dossier modifié — enregistrement à la sortie du champ.");
} if (el.id === 'urls') {
    const lines = el.value.split(/\r?\n/).filter(x => x.trim());
    const hosts = new Set(lines.map(x => { try {
        return new URL(x).hostname.replace(/^www\./, '');
    }
    catch {
        return t("Adresse à vérifier");
    } }));
    $('#detected').textContent = lines.length ? t("{0} adresse(s) · {1}", lines.length, [...hosts].slice(0, 3).join(', ')) : t("Site détecté automatiquement");
} });
document.addEventListener('submit', e => {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    run(async () => {
        const button = form.querySelector<HTMLButtonElement>('button[type="submit"],.settings-footer .primary,.add-row .primary');
        if (button)
            button.disabled = true;
        try {
            if (form.id === 'custom-site-form') {
                const r = await api('custom-site', { id: form.dataset.siteId || '', name: val('custom-name'), domain: val('custom-domain') });
                await load(true);
                renderAccounts();
                await openSite(r.id);
                toast(t("Site enregistré."));
                return;
            }
            if (form.id === 'backup-form') {
                await submitBackup(form);
                return;
            }
            if (form.id === 'add-form') {
                const p = val('job-profile');
                const body: any = { urls: val('urls').split(/\r?\n/).map(x => x.trim()).filter(Boolean), destination: val('job-dest'), layout: val('job-layout') };
                if (p !== 'inherit')
                    body.profileId = p;
                if (check('override-media'))
                    body.filters = readFilters('job');
                const r = await api('jobs', body);
                toast(t("{0} tâche(s) ajoutée(s).{1}", r.added, r.errors.length ? '\n' + r.errors.map(message).join('\n') : ''), !!r.errors.length);
                if (!r.errors.length)
                    $<HTMLTextAreaElement>('#urls').value = '';
                await load(true);
            }
            if (form.id === 'settings-form') {
                await saveGlobalDestination();
                const s = { ...data.state.settings, destination: val('global-dest'), layout: val('global-layout'), filename: val('global-filename'), filters: readFilters('global'), concurrency: Number(val('concurrency')), quotaBytes: Math.round(Number(val('quota')) * 1e9), rateKiB: Number(val('rate')), galleryPath: val('gallery-path'), ytdlpPath: val('ytdlp-path') };
                await api('settings', s);
                await load(true);
                toast(t("Paramètres enregistrés."));
            }
            if (form.id === 'site-form') {
                const id = form.dataset.siteId!;
                const sc: SiteSettings = { destination: check('site-use-global') ? '' : val('site-dest').trim(), layout: val('site-layout'), filename: val('site-filename'), profileId: val('site-default') };
                if (check('site-media-override'))
                    sc.filters = readFilters('site');
                const s = { ...data.state.settings, sites: { ...data.state.settings.sites, [id]: sc } };
                await api('settings', s);
                await load(true);
                toast(t("Réglages du site enregistrés."));
                closeModal();
            }
            if (form.id === 'job-profile-form') {
                const p = val('detail-profile');
                await api('job/profile', { id: form.dataset.jobId, mode: p === 'inherit' ? 'inherit' : p ? 'explicit' : 'none', profileId: p === 'inherit' ? '' : p });
                await load(true);
                await openDetail(form.dataset.jobId!);
                toast(t("Compte affecté. Cliquez sur Démarrer / reprendre."));
            }
            if (form.id === 'profile-form') {
                await api('profile', { id: form.dataset.profileId, name: val('profile-name'), site: form.dataset.siteId, method: val('profile-method'), userId: val('profile-user'), token: val('profile-token'), tokenSecret: val('profile-secret'), cookiePath: val('profile-cookie'), setDefault: check('profile-default') });
                await load(true);
                closeModal();
                renderAccounts();
                toast(t("Compte enregistré et relu avec succès. La reprise des téléchargements reste manuelle."));
            }
        }
        finally {
            if (button)
                button.disabled = false;
        }
    });
});
$<HTMLDialogElement>('#modal').addEventListener('close', () => { cancelOAuth(); modalJob = ''; $('#modal-content').innerHTML = ''; });
run(async () => { await loadCatalog(); await load(); setInterval(() => run(load), 1200); });
function canRefreshDestination(j: Job) { return !j.started && !j.bytes && !j.files && !j.skipped && !['running', 'completed', 'cancelled'].includes(j.status) && j.destinationMode !== 'explicit'; }
function openBackup(mode: 'export' | 'import') {
    const exporting = mode === 'export';
    showModal(`<div class="modal-head"><h2>${exporting ? t("Exporter les sites et comptes") : t("Importer les sites et comptes")}</h2><button data-close>×</button></div>
 <div class="notice">${exporting ? t("Cette sauvegarde contient vos clés, tokens et cookies de connexion. Toute personne possédant le fichier et la phrase secrète pourra accéder à ces comptes.") : t("Importez uniquement une sauvegarde de confiance. Les comptes seront ajoutés au coffre de cette session ; aucun téléchargement ne sera lancé.")}</div>
 <form id="backup-form" data-mode="${mode}" class="section-gap"><div class="form-grid">
 <label class="full">${exporting ? t("Nouveau fichier de sauvegarde") : t("Fichier de sauvegarde")}${pathInput('backup-path', '', t("Fichier .dlmbackup"), exporting ? 'backup-export' : 'backup-import')}</label>
 <label class="full">${t("Phrase secrète (16 caractères minimum)")}<input id="backup-phrase" type="password" autocomplete="${exporting ? 'new-password' : 'off'}" required></label>
 ${exporting ? `<label class="full">${t("Confirmer la phrase secrète")}<input id="backup-confirm" type="password" autocomplete="new-password" required></label>
 <label class="full check"><input id="backup-ack" type="checkbox" required>${t("Je comprends que ce fichier contient mes accès aux comptes.")}</label>` : `<label class="full check"><input id="backup-replace" type="checkbox">${t("Remplacer les réglages des sites déjà configurés")}</label>`}
 </div><p class="footer-note">${exporting ? t("Le fichier est chiffré. Choisissez une phrase longue et unique : elle ne sera pas enregistrée et ne pourra pas être récupérée. Utilisez un nouveau nom de fichier à chaque export.") : t("Les comptes existants sont conservés. Réimporter le même fichier ajoute des doublons. Les chemins incompatibles avec cet OS héritent du dossier global ; vérifiez vos destinations après l’import.")}<br>${t("{0} Les téléchargements, l’historique et les réglages globaux ne font pas partie de cette sauvegarde.", exporting ? t("L’export est possible pendant les téléchargements.") : t("L’import nécessite une file en pause et l’arrêt des tâches actives."))}</p>
 ${!exporting ? `<div class="row wrap"><button type="button" id="backup-pause">${t("Mettre la file en pause")}</button><span id="backup-queue-status" class="smalltext muted" role="status"></span></div>` : ''}
 <div class="settings-footer"><button type="button" data-close>${t("Annuler")}</button><button class="primary" type="submit">${exporting ? t("Chiffrer et exporter") : t("Déchiffrer et importer")}</button></div></form>`);
}
async function submitBackup(form: HTMLFormElement) {
    const exporting = form.dataset.mode === 'export';
    const phrase = val('backup-phrase');
    if ([...phrase].length < 16)
        throw Error(t("La phrase secrète doit contenir au moins 16 caractères."));
    if (exporting && phrase !== val('backup-confirm'))
        throw Error(t("Les deux phrases secrètes ne correspondent pas."));
    const body: any = { path: val('backup-path'), passphrase: phrase };
    if (exporting) {
        body.confirmation = val('backup-confirm');
        body.acknowledge = check('backup-ack');
    }
    else
        body.replaceSites = check('backup-replace');
    try {
        const r = await api(exporting ? 'backup/export' : 'backup/import', body);
        closeModal();
        await load(true);
        renderAccounts();
        toast(exporting ? t("Sauvegarde chiffrée créée : {0} compte(s), {1} site(s).", r.accounts, r.sites) : t("Import terminé : {0} compte(s) ajouté(s), {1} site(s) appliqué(s).{2}", r.accounts, r.sites, r.resetPaths ? ' ' + r.resetPaths + t(" chemin(s) remis au dossier global.") : ''));
    }
    finally {
        form.querySelectorAll<HTMLInputElement>('input[type=password]').forEach(input => input.value = '');
        body.passphrase = '';
        body.confirmation = '';
    }
}

async function changeLanguage(next: string) {
 const select=$<HTMLSelectElement>('#language');select.disabled=true;
 const navigation=Array.from(document.querySelectorAll<HTMLButtonElement>('[data-page]'));navigation.forEach(button=>button.disabled=true);
 try {
  await destinationSave;await api('settings/language',{language:next});await load(true);
  // Capture immediately before rendering: the user may have typed or navigated while saving.
  const draft=Array.from(document.querySelectorAll<HTMLInputElement|HTMLSelectElement|HTMLTextAreaElement>('#main input, #main select, #main textarea')).filter(el=>el.id&&el.id!=='language').map(el=>({id:el.id,value:el.value,checked:(el as HTMLInputElement).checked}));
  const details=Array.from(document.querySelectorAll<HTMLDetailsElement>('#main details')).map(el=>el.open);
  const scroll=window.scrollY;shell();
  for(const saved of draft){const el=document.getElementById(saved.id) as HTMLInputElement|HTMLSelectElement|HTMLTextAreaElement|null;if(el){el.value=saved.value;if(el instanceof HTMLInputElement)el.checked=saved.checked}}
  document.querySelectorAll<HTMLDetailsElement>('#main details').forEach((el,i)=>el.open=details[i]);preview();window.scrollTo(0,scroll);
 }catch(e){if(select.isConnected)select.value=language();throw e}finally{if(select.isConnected)select.disabled=false;navigation.forEach(button=>{if(button.isConnected)button.disabled=false})}
}
