"use strict";
const $ = (s) => document.querySelector(s);
const esc = (v) => String(v ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
const token = $('meta[name="dl-token"]').content;
const statuses = { queued: 'En attente', running: 'En cours', paused: 'En pause', interrupted: 'Interrompu', completed: 'Terminé', failed: 'Erreur', authentication_required: 'Connexion requise', cancelled: 'Annulé', paused_quota: 'Limite atteinte' };
let data;
let page = 'downloads';
let search = '';
let modalJob = '';
let loading = null;
let toastTimer;
async function api(path, body) { const r = await fetch('/api/' + path, { method: body === undefined ? 'GET' : 'POST', headers: body === undefined ? {} : { 'Content-Type': 'application/json', 'X-DL-Token': token }, body: body === undefined ? undefined : JSON.stringify(body) }); const v = await r.json(); if (!r.ok)
    throw Error(v.error || `Erreur ${r.status}`); return v; }
function toast(s, error = false) { const t = $('#toast'); t.textContent = s; t.className = error ? 'error' : ''; t.style.display = 'block'; clearTimeout(toastTimer); toastTimer = setTimeout(() => t.style.display = 'none', error ? 9000 : 4500); }
function run(fn) { fn().catch(e => toast(e.message, true)); }
function bytes(n) { if (n < 1000)
    return `${n} o`; const units = ['Ko', 'Mo', 'Go', 'To']; let i = -1; do {
    n /= 1000;
    i++;
} while (n >= 1000 && i < 3); return `${n.toLocaleString('fr-FR', { maximumFractionDigits: i < 1 ? 0 : 2 })} ${units[i]}`; }
function date(s) { return s ? new Date(s).toLocaleString('fr-FR', { dateStyle: 'short', timeStyle: 'short' }) : '—'; }
function siteName(id) { return data.sites.find(s => s.id === id)?.name || id; }
function val(id) { return $('#' + id).value; }
function check(id) { return $('#' + id).checked; }
function opts(values, selected) { return values.map(([v, label]) => `<option value="${esc(v)}" ${v === selected ? 'selected' : ''}>${esc(label)}</option>`).join(''); }
const layouts = [['flat', 'Dossier de destination'], ['site', 'Site'], ['creator', 'Site / Créateur']];
const filenames = [['default', 'Nommage du site (identifiants inclus)'], ['id-title', 'Identifiant + titre']];
function extensions(prefix, f) { return `<div class="row wrap"><label class="check"><input id="${prefix}-images" type="checkbox" ${f.images ? 'checked' : ''}>Images</label><label class="check"><input id="${prefix}-videos" type="checkbox" ${f.videos ? 'checked' : ''}>Vidéos</label></div><div class="extensions">${['jpg', 'jpeg', 'png', 'webp', 'gif', 'mp4', 'webm', 'mov', 'avi', 'mkv'].map(x => `<label class="extension"><input class="${prefix}-extension" type="checkbox" value="${x}" ${f.extensions.includes(x) ? 'checked' : ''}>${x}</label>`).join('')}</div><label>Extensions supplémentaires (séparées par des virgules)<input id="${prefix}-custom" value="${esc(f.extensions.filter(x => !['jpg', 'jpeg', 'png', 'webp', 'gif', 'mp4', 'webm', 'mov', 'avi', 'mkv'].includes(x)).join(', '))}" placeholder="avif, bmp…"></label>`; }
function readFilters(prefix) { return { images: check(prefix + '-images'), videos: check(prefix + '-videos'), extensions: [...new Set([...Array.from(document.querySelectorAll('.' + prefix + '-extension:checked')).map(x => x.value), ...val(prefix + '-custom').split(',').map(x => x.trim().toLowerCase().replace(/^\./, '')).filter(Boolean)])] }; }
function pathInput(id, value, placeholder = 'Hériter du dossier global', kind = 'directory') { return `<div class="pathfield"><input id="${id}" value="${esc(value)}" placeholder="${esc(placeholder)}"><button type="button" data-pick="${id}" data-kind="${kind}" title="Parcourir">…</button></div>`; }
async function load(fresh = false) { if (loading) {
    await loading;
    if (fresh)
        return load(true);
    return;
} loading = (async () => { data = await api('state'); if (!$('#main'))
    shell(); updateChrome(); if (page === 'downloads' || page === 'history')
    updateJobs(); if (modalJob)
    await refreshLog(); })(); try {
    await loading;
}
finally {
    loading = null;
} }
function shell() { $('#app').innerHTML = `<div class="layout"><aside class="sidebar"><div class="brand"><span class="logo">↓</span><span class="brand-text">dl-manager</span></div><nav class="nav" aria-label="Navigation principale">${[['downloads', '↓', 'Téléchargements'], ['history', '◷', 'Historique'], ['accounts', '♙', 'Comptes & sites'], ['settings', '⚙', 'Paramètres']].map(([id, icon, label]) => `<button data-page="${id}" class="${page === id ? 'active' : ''}"><span class="nav-icon">${icon}</span><span class="nav-label">${label}</span></button>`).join('')}</nav><div class="side-bottom"><div>VOTRE FILE, À VOTRE RYTHME</div><div id="side-state">En pause</div><button id="hide-app">Réduire en arrière-plan</button><button class="ghost" id="quit-app">Quitter l'application</button><div class="section-gap">Version ${esc(data.version)} · ${data.platform === 'linux' ? 'Linux' : 'Windows'}</div></div></aside><main class="main" id="main"></main></div>`; renderPage(); }
function header(title, sub, actions = '') { return `<header class="page-head"><div><div class="eyebrow">DL-MANAGER / ${page === 'downloads' ? 'FILE D’ATTENTE' : page === 'history' ? 'BIBLIOTHÈQUE' : page === 'accounts' ? 'CONNEXIONS' : 'PRÉFÉRENCES'}</div><h1>${title}</h1><p>${sub}</p></div><div class="head-actions">${actions}</div></header>`; }
function renderPage() { document.querySelectorAll('[data-page]').forEach(x => x.classList.toggle('active', x.dataset.page === page)); if (page === 'downloads')
    renderDownloads();
else if (page === 'history')
    renderHistory();
else if (page === 'accounts')
    renderAccounts();
else
    renderSettings(); updateChrome(); }
function renderDownloads() { const s = data.state.settings; $('#main').innerHTML = header('Téléchargements', 'Ajoutez vos liens. La file s’occupe de la suite.', '<button id="pause-all">Ⅱ Tout mettre en pause</button><button class="primary" id="start-all">▶ Démarrer / reprendre</button>') + `<div id="queue-notice"></div><section class="panel"><div class="panel-title"><h2>Ajouter à la file</h2><span class="tag">Plusieurs liens acceptés</span></div><form id="add-form"><div class="add-row"><textarea id="urls" aria-label="Adresses à télécharger" placeholder="Collez une adresse, ou plusieurs — une par ligne" required></textarea><button class="primary" type="submit">＋ Ajouter</button></div><div class="media-row"><span id="detected" class="muted smalltext">Site détecté automatiquement</span><span class="divider"></span><span class="muted smalltext">Réglages du site appliqués par défaut</span></div><details><summary>Options pour ces téléchargements</summary><div class="form-grid"><label class="full">Destination${pathInput('job-dest', '', 'Dossier du site ou dossier global')}</label><label>Organisation<select id="job-layout">${opts([['', 'Réglage du site'], ...layouts], '')}</select></label><label>Compte<select id="job-profile"><option value="inherit">Compte par défaut du site</option><option value="">Sans compte</option>${data.state.profiles.map(p => `<option value="${p.id}">${esc(siteName(p.site) + ' · ' + p.name)}</option>`).join('')}</select></label><div class="full"><label class="check"><input id="override-media" type="checkbox">Personnaliser les médias pour ces liens</label><div id="job-filter-options" class="hidden section-gap">${extensions('job', s.filters)}</div></div></div></details></form></section><div id="stats" class="stats"></div><section class="panel table-panel"><div class="table-toolbar"><h2>File d'attente <span id="queue-count" class="muted"></span></h2><button class="small" id="refresh-destinations">Actualiser les destinations en attente</button></div><div id="job-table" class="table-wrap"></div></section><div class="bottom-status"><span id="queue-status"></span><span>Les fichiers partiels sont conservés.</span></div>`; updateJobs(); }
function renderHistory() { $('#main').innerHTML = header('Historique', 'Retrouvez les tâches terminées, les interruptions et les erreurs.', '<button id="clear-history">Retirer les tâches terminées</button>') + `<section class="panel table-panel"><div class="table-toolbar"><h2>Téléchargements précédents</h2><input id="history-search" placeholder="Rechercher une adresse ou un site…" value="${esc(search)}" aria-label="Rechercher dans l'historique"></div><div id="job-table" class="table-wrap"></div></section>`; updateJobs(); }
function updateChrome() { if (!data)
    return; const s = data.state; const el = $('#side-state'); if (el)
    el.textContent = s.running ? `${s.active} tâche(s) en cours` : s.active > 0 ? 'Arrêt en cours…' : 'En pause · reprise manuelle'; const start = $('#start-all'); if (start)
    start.disabled = s.active > 0 || !s.jobs.some(j => ['queued', 'paused', 'interrupted', 'paused_quota', 'authentication_required'].includes(j.status)); const pause = $('#pause-all'); if (pause)
    pause.disabled = !s.running && s.active === 0; }
function jobErrorMessage(j) {
    if (!/rate limit|too many requests|\b429\b/i.test(j.error))
        return j.error;
    const reset = j.error.match(/rate limit will reset at (\d{2}:\d{2}:\d{2})/i);
    const message = j.site === 'tumblr' ? 'Quota de l’API Tumblr atteint.' : 'Limite de requêtes du site atteinte.';
    return message + (reset ? ' Remise à zéro annoncée par le moteur : ' + reset[1] + '. Réessayez après cette heure.' : ' Réessayez plus tard.');
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
    $('#job-table').innerHTML = jobs.length ? `<table><thead><tr><th>Source</th><th>État</th><th>Fichiers</th><th>Volume ajouté</th><th>Actions</th></tr></thead><tbody>${jobs.map(j => `<tr><td class="url"><div class="url-main" title="${esc(j.url)}">${esc(j.url)}</div><div class="url-sub">${esc(siteName(j.site))} · ${esc(j.current || j.destination)}</div></td><td><span class="status ${esc(j.status)}">${esc(statuses[j.status] || j.status)}</span>${j.error ? `<div class="job-error error" title="${esc(j.error)}">${esc(jobErrorMessage(j))}</div>` : ''}</td><td>${j.files}<span class="muted"> / ${j.skipped} ignoré(s)</span></td><td>${bytes(j.bytes)}</td><td><div class="job-actions"><button class="small" data-detail="${j.id}">Détails</button>${j.status === 'running' ? `<button class="small" data-action="pause" data-id="${j.id}" title="Mettre en pause">Ⅱ</button>` : `<button class="small" data-action="retry" data-id="${j.id}" title="Remettre en attente">↻</button><button class="small" data-action="up" data-id="${j.id}" title="Monter">↑</button><button class="small" data-action="down" data-id="${j.id}" title="Descendre">↓</button><button class="small danger" data-action="delete" data-id="${j.id}" title="Retirer de la liste">×</button>`}</div></td></tr>`).join('')}</tbody></table>` : `<div class="empty"><div class="empty-icon">${page === 'downloads' ? '↓' : '◷'}</div><h3>${page === 'downloads' ? 'Votre file est prête' : 'Aucun téléchargement ici'}</h3><p class="smalltext">${page === 'downloads' ? 'Collez vos premières adresses ci-dessus, puis cliquez sur Démarrer.' : 'Les tâches terminées et leurs détails apparaîtront ici.'}</p></div>`;
    if (page === 'downloads') {
        const limit = s.settings.quotaBytes;
        $('#stats').innerHTML = `<div class="stat"><div class="label">EN ATTENTE</div><div class="value">${all.filter(j => j.status === 'queued').length}</div></div><div class="stat"><div class="label">EN COURS</div><div class="value">${s.active}<span class="muted" style="font-size:13px"> / ${s.settings.concurrency}</span></div></div><div class="stat"><div class="label">TERMINÉS</div><div class="value">${all.filter(j => j.status === 'completed').length}</div></div><div class="stat"><div class="row"><div class="label">VOLUME DEPUIS LA REMISE À ZÉRO</div><div class="spacer"></div><button class="small ghost" id="reset-quota">Réinitialiser</button></div><div class="value">${bytes(s.usedBytes)}<span class="muted" style="font-size:12px"> / ${limit ? bytes(limit) : 'sans limite'}</span></div><div class="meter ${limit && s.usedBytes >= limit ? 'limit' : ''}"><span style="width:${limit ? Math.min(100, s.usedBytes / limit * 100) : 0}%"></span></div></div>`;
        $('#queue-count').textContent = `(${jobs.length})`;
        $('#queue-status').textContent = s.running ? '● File active — vous pouvez ajouter d’autres liens' : '○ File en pause — aucun démarrage automatique';
        $('#queue-notice').innerHTML = s.lastError ? `<div class="notice">${esc(s.lastError)}</div>` : limit && s.usedBytes >= limit ? '<div class="notice">Limite de volume atteinte. Réinitialisez le compteur ou augmentez la limite, puis cliquez sur Reprendre.</div>' : !s.settings.destination ? '<div class="notice">Choisissez votre dossier global dans Paramètres, ou une destination dans les options de téléchargement.</div>' : '';
    }
}
function renderAccounts() { $('#main').innerHTML = header('Comptes & sites', 'Réglez une fois la destination et la connexion de chaque plateforme.', '<button id="backup-import">Importer une sauvegarde</button><button id="backup-export">Exporter les sites et comptes</button>') + `<div class="site-grid">${data.sites.map(s => { const n = data.state.profiles.filter(p => p.site === s.id).length; return `<button class="site-card" data-site="${s.id}"><span class="site-icon">${esc(s.name.slice(0, 2))}</span><span><h3>${esc(s.name)}</h3><p>${n ? n + ' compte(s) enregistré(s)' : 'Public ou à configurer'}</p></span></button>`; }).join('')}</div><section class="panel section-gap"><h2>Vos comptes</h2>${data.state.profiles.length ? data.state.profiles.map(p => `<div class="profile-row"><div><h3>${esc(p.name)}</h3><p class="muted smalltext">${esc(siteName(p.site))} · ${esc(p.method)} · ${p.state === 'authentication_required' ? 'Connexion à renouveler' : 'Configuré'}</p></div><div class="row"><button class="small" data-profile-check="${p.id}">Vérifier le stockage</button><button class="small" data-profile-edit="${p.id}">Modifier</button><button class="small danger" data-profile-delete="${p.id}">Supprimer</button></div></div>`).join('') : '<p class="muted section-gap">Aucun compte enregistré. Les sites publics restent accessibles sans compte.</p>'}<p class="footer-note">Les connexions se font dans votre navigateur. Les secrets importés sont protégés par le coffre de votre session. Pour changer de PC ou d’OS, utilisez une sauvegarde chiffrée avec une phrase secrète.</p></section>`; }
async function openSite(id, profile) {
    const site = data.sites.find(x => x.id === id);
    const s = data.state.settings.sites[id] || { destination: '', layout: '', filename: '', profileId: '' };
    const profiles = data.state.profiles.filter(p => p.site === id);
    const selected = profile?.id || s.profileId || profiles[0]?.id || '';
    showModal(`<div class="modal-head"><h2>${esc(site.name)}</h2><button data-close>×</button></div>
 <form id="site-form" data-site-id="${id}"><h3>Destination et classement</h3><div class="form-grid">
 <label class="full check"><input id="site-use-global" type="checkbox" ${!s.destination ? 'checked' : ''}>Utiliser le dossier global</label><label class="full">Dossier de téléchargement${pathInput('site-dest', s.destination || '', 'Choisir un dossier propre à ce site')}</label><p class="full footer-note">Dossier global actuel : <code>${esc(data.state.settings.destination)}</code>. Cochez pour suivre ses changements ; décochez pour choisir un dossier propre au site. Les tâches déjà commencées gardent leur dossier.</p>
 <label>Organisation<select id="site-layout">${opts([['', 'Hériter du réglage global'], ...layouts], s.layout || '')}</select></label>
 <label>Nommage des fichiers<select id="site-filename">${opts([['', 'Hériter du réglage global'], ...filenames], s.filename || '')}</select></label>
 <label class="full">Compte par défaut pour les tâches héritées<select id="site-default">${opts([['', 'Sans compte'], ...profiles.map(p => [p.id, p.name])], s.profileId || '')}</select></label>
 <div class="full"><label class="check"><input type="checkbox" id="site-media-override" ${s.filters ? 'checked' : ''}>Filtres propres à ce site</label><div class="section-gap" id="site-filter-options">${extensions('site', s.filters || data.state.settings.filters)}</div></div></div>
 <div class="settings-footer"><button class="primary">Enregistrer les réglages du site</button></div></form>
 <div class="guide">${esc(site.guide)}<br><button class="small ghost" type="button" data-help="${id}" style="margin-top:10px">Ouvrir le site / la documentation ↗</button></div>
 <label>Compte à consulter ou modifier<select id="profile-select" data-site-id="${id}">${opts([['', 'Ajouter un nouveau compte'], ...profiles.map(p => [p.id, p.name])], selected)}</select></label>
 <div id="account-editor" class="section-gap"></div>`);
    updateSiteDestination();
    $('#site-filter-options').classList.toggle('hidden', !s.filters);
    await showProfileEditor(id, selected);
}
async function showProfileEditor(siteId, id) {
    cancelOAuth();
    const site = data.sites.find(x => x.id === siteId);
    const p = data.state.profiles.find(p => p.id === id);
    $('#account-editor').textContent = 'Lecture du compte…';
    const saved = p ? await api('profile?id=' + encodeURIComponent(id)) : {};
    // Ignore responses for a modal/profile that has since been closed or replaced.
    const selector = $('#profile-select');
    if (!selector || selector.dataset.siteId !== siteId || selector.value !== id)
        return;
    const isDefault = data.state.settings.sites[siteId]?.profileId || '';
    $('#account-editor').innerHTML = `<form id="profile-form" data-site-id="${siteId}" data-profile-id="${p?.id || ''}">
 <h3>${p ? 'Modifier ' + esc(p.name) : 'Ajouter un compte'}</h3>
 ${p ? '<p class="notice">Compte enregistré. Les champs secrets restent masqués ; laissez-les vides pour conserver leur valeur.</p>' : ''}
 <div class="form-grid"><label>Nom du compte<input id="profile-name" required value="${esc(p?.name || 'Principal')}"></label>
 <label>Méthode<select id="profile-method">${opts(site.methods.map(m => [m, { api: 'User ID + API Key', cookies: 'Fichier cookies.txt', token: 'Refresh token', oauth1: 'OAuth Token + Secret' }[m] || m]), p?.method || site.methods[0])}</select></label>
 <label class="full" id="cookie-field">Fichier cookies.txt (Netscape)${pathInput('profile-cookie', '', saved.hasCookies ? 'Cookies enregistrés — laisser vide pour les conserver' : 'Sélectionner le fichier exporté', 'cookies')}</label>
 <label id="userid-field">User ID<input id="profile-user" value="${esc(saved.userId || '')}" inputmode="numeric" autocomplete="off"></label>
 <label id="token-field">Token / API Key<input id="profile-token" type="password" autocomplete="new-password" placeholder="${saved.hasToken ? 'Clé enregistrée — laisser vide pour la conserver' : 'Saisir la clé'}"></label>
 <label id="secret-field">OAuth Token Secret<input id="profile-secret" type="password" autocomplete="new-password" placeholder="${saved.hasTokenSecret ? 'Secret enregistré — laisser vide pour le conserver' : 'Saisir le secret'}"></label>
${['tumblr', 'deviantart'].includes(siteId) ? `<div id="oauth-controls" class="full"><p class="footer-note">Vous devrez vous connecter dans votre navigateur et autoriser gallery-dl. Les tokens rempliront les champs ci-dessus ; enregistrez ensuite le compte.</p><div class="row section-gap"><button type="button" id="oauth-start">Générer le token</button><button type="button" id="oauth-cancel" class="hidden">Annuler la connexion</button><button type="button" id="oauth-open" class="hidden">Ouvrir le navigateur</button></div><input id="oauth-url" class="hidden section-gap" aria-label="Lien de connexion" readonly><p id="oauth-status" class="footer-note" role="status"></p></div>` : ""}
 <label class="full check"><input type="checkbox" id="profile-default" ${(p ? isDefault === id : !isDefault) ? 'checked' : ''}>Utiliser ce compte par défaut pour ce site</label></div>
 <p class="footer-note">Le compte par défaut s'applique aussi aux tâches existantes réglées sur « Compte par défaut du site ». Un changement de méthode demande les nouveaux identifiants.</p>
 <div class="settings-footer"><button class="primary">Enregistrer le compte</button></div></form>`;
    toggleMethod();
}
let oauthAttempt = null;
function cancelOAuth() { const a = oauthAttempt; if (!a)
    return; oauthAttempt = null; a.cancelled = true; if (a.id)
    api('oauth/cancel', { id: a.id }).catch(() => { }); if (a.form.isConnected) {
    a.form.querySelector('#oauth-start').disabled = false;
    a.form.querySelector('button.primary').disabled = false;
    a.form.querySelector('#oauth-cancel').classList.add('hidden');
    a.form.querySelector('#oauth-open').classList.add('hidden');
    const link = a.form.querySelector('#oauth-url');
    link.value = '';
    link.classList.add('hidden');
} }
async function generateOAuth() {
    const form = $('#profile-form');
    if (!form || oauthAttempt)
        return;
    const a = { id: '', form, cancelled: false };
    oauthAttempt = a;
    const current = () => oauthAttempt === a && !a.cancelled && form.isConnected;
    const status = form.querySelector('#oauth-status');
    form.querySelector('#oauth-start').disabled = true;
    form.querySelector('button.primary').disabled = true;
    form.querySelector('#oauth-cancel').classList.remove('hidden');
    status.textContent = 'Préparation de la connexion…';
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
                throw Error(v.error || 'Connexion interrompue.');
            if (v.state === 'completed') {
                const c = await api('oauth/result', { id: a.id });
                if (!current())
                    return;
                $('#profile-token').value = c.token;
                $('#profile-secret').value = c.tokenSecret || '';
                status.textContent = 'Token reçu. Cliquez sur Enregistrer le compte pour le conserver.';
                return;
            }
            status.textContent = 'Connectez-vous dans le navigateur et autorisez gallery-dl. Attente du retour…';
            if (v.url && !opened) {
                opened = true;
                const link = form.querySelector('#oauth-url');
                link.value = v.url;
                link.classList.remove('hidden');
                form.querySelector('#oauth-open').classList.remove('hidden');
                try {
                    await api('oauth/open', { id: a.id });
                }
                catch {
                    if (current())
                        status.textContent = 'Ouvrez le lien affiché dans votre navigateur pour vous connecter.';
                }
            }
            await new Promise(r => setTimeout(r, 500));
        }
    }
    catch (e) {
        if (current()) {
            status.textContent = e.message;
            status.className = 'footer-note error';
        }
    }
    finally {
        if (oauthAttempt === a)
            cancelOAuth();
    }
}
function jobProfileMode(j) { return j.profileMode || (j.profileId ? 'explicit' : 'inherit'); }
function jobProfileLabel(j) { const mode = jobProfileMode(j); const id = mode === 'inherit' ? data.state.settings.sites[j.site]?.profileId : j.profileId; const p = data.state.profiles.find(x => x.id === id); return mode === 'none' ? 'Sans compte' : (mode === 'inherit' ? 'Par défaut : ' : '') + (p?.name || 'Sans compte'); }
function toggleMethod() { cancelOAuth(); const m = val('profile-method'); const controls = $('#oauth-controls'); if (controls)
    controls.classList.toggle('hidden', m === 'cookies'); $('#cookie-field').classList.toggle('hidden', m !== 'cookies'); $('#userid-field').classList.toggle('hidden', m !== 'api'); $('#token-field').classList.toggle('hidden', m === 'cookies'); $('#secret-field').classList.toggle('hidden', m !== 'oauth1'); }
let destinationSave = Promise.resolve();
async function saveGlobalDestination() {
    const input = $('#global-dest');
    if (!input)
        return destinationSave;
    const destination = input.value.trim();
    const status = $('#global-save-status');
    const pending = destinationSave.catch(() => { }).then(async () => {
        try {
            if (destination !== data.state.settings.destination) {
                if (status.isConnected) {
                    status.className = 'footer-note';
                    status.textContent = 'Enregistrement du dossier global…';
                }
                await api('settings/destination', { destination });
                await load(true);
            }
            if (input.isConnected && input.value.trim() === destination) {
                input.value = destination;
                status.className = 'footer-note';
                status.textContent = 'Dossier global enregistré. Les sites qui en héritent utilisent ce chemin.';
            }
        }
        catch (e) {
            if (status.isConnected) {
                status.className = 'footer-note error';
                status.textContent = 'Dossier non enregistré : ' + e.message;
            }
            throw e;
        }
    });
    destinationSave = pending;
    return pending;
}
function updateSiteDestination() {
    const input = $('#site-dest');
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
    $('[data-pick="site-dest"]').disabled = inherited;
}
function renderSettings() { const s = data.state.settings; $('#main').innerHTML = header('Paramètres', 'Le dossier global s’applique aux tâches héritées jamais démarrées. Les autres valeurs s’appliquent aux nouveaux ajouts.') + `<form id="settings-form"><section class="panel settings-panel"><h2>Destination & classement</h2><div class="form-grid"><label class="full">Dossier global de téléchargement${pathInput('global-dest', s.destination, 'Choisir un dossier')}</label><p id="global-save-status" class="full footer-note" role="status">Le dossier global est enregistré automatiquement après modification. Les autres paramètres se sauvegardent avec le bouton ci-dessous.</p><label>Organisation des dossiers<select id="global-layout">${opts(layouts, s.layout)}</select></label><label>Nommage des fichiers<select id="global-filename">${opts(filenames, s.filename)}</select></label><div class="full preview" id="path-preview"></div></div></section><section class="panel settings-panel"><h2>Rythme & limite de volume</h2><div class="form-grid"><label>Téléchargements simultanés<select id="concurrency">${opts([['1', '1 tâche'], ['2', '2 tâches'], ['3', '3 tâches']], String(s.concurrency))}</select></label><label>Débit par tâche (Kio/s, 0 = illimité)<input id="rate" type="number" min="0" value="${s.rateKiB}"></label><label>Pause après ce volume (Go, 0 = désactivé)<input id="quota" type="number" min="0" step="0.1" value="${s.quotaBytes / 1e9}"></label><div><label>Volume actuel</label><p class="section-gap">${bytes(data.state.usedBytes)} <button type="button" class="small" id="reset-quota">Réinitialiser</button></p></div></div><p class="footer-note">Le compteur est conservé au redémarrage. Le réinitialiser met les tâches en pause et attend votre clic pour reprendre. Un même site et dossier, ou un même compte, est traité par une seule tâche à la fois pour éviter les conflits.</p></section><section class="panel settings-panel"><h2>Médias par défaut</h2><div class="section-gap">${extensions('global', s.filters)}</div><p class="footer-note">Les extensions filtrent les fichiers disponibles à la source. Aucun transcodage automatique. Les extensions personnalisées inconnues sont classées comme images.</p></section><section class="panel settings-panel"><h2>Moteurs</h2><div id="engines" class="section-gap muted">Vérification des versions…</div><details><summary>Chemins personnalisés (facultatifs)</summary><div class="form-grid"><label class="full">gallery-dl${pathInput('gallery-path', s.galleryPath, 'Utiliser le moteur du paquet', 'engine')}</label><label class="full">yt-dlp${pathInput('ytdlp-path', s.ytdlpPath, 'Utiliser le moteur du paquet', 'engine')}</label></div></details><p class="footer-note">Données : <code>${esc(data.dataDir)}</code><br>Le paquet garde les outils dans tools/ et les données dans data/.</p><div class="settings-footer"><button class="primary">Enregistrer les paramètres</button></div></section></form>`; preview(); run(async () => { const es = await api('engines'); if (page === 'settings')
    $('#engines').innerHTML = es.map((e) => `<div class="engine"><span>${esc(e.name)}</span><span>${e.error ? `<span class="error">${esc(e.error)}</span>` : `<code>${esc(e.version)}</code>`}</span></div>`).join(''); }); }
function preview() { if (!$('#path-preview'))
    return; let p = val('global-dest') || 'Destination'; const l = val('global-layout'); if (l === 'site' || l === 'creator')
    p += (data.platform === 'linux' ? '/' : '\\') + 'artstation'; if (l === 'creator')
    p += (data.platform === 'linux' ? '/' : '\\') + 'nom_artiste'; p += (data.platform === 'linux' ? '/' : '\\') + 'identifiant_titre.jpg'; $('#path-preview').textContent = p; }
function showModal(content) { cancelOAuth(); modalJob = ''; $('#modal-content').innerHTML = content; const d = $('#modal'); if (!d.open)
    d.showModal(); }
function closeModal() { cancelOAuth(); modalJob = ''; $('#modal').close(); $('#modal-content').innerHTML = ''; }
async function openDetail(id) { const j = data.state.jobs.find(x => x.id === id); if (!j)
    return; showModal(`<div class="modal-head"><h2>Détails du téléchargement</h2><button data-close>×</button></div><div class="detail-grid"><span>Source</span><span>${esc(j.url)}</span><span>Destination</span><span>${esc(j.destination)}<br><small class="muted">${j.started ? 'Dossier conservé pour la reprise' : j.destinationMode === 'inherit' ? 'Dossier du site ou global, actualisé au premier départ' : j.destinationMode === 'explicit' ? 'Dossier personnalisé pour cette tâche' : 'Ancienne tâche : dossier fixé à l’ajout'}</small></span><span>Compte</span><span>${esc(jobProfileLabel(j))}</span><span>Ajout</span><span>${date(j.added)}</span><span>Dernier démarrage</span><span>${date(j.started)}</span><span>État</span><span>${esc(statuses[j.status])} · code ${j.exitCode}</span><span>Message</span><span class="error">${esc(jobErrorMessage(j) || '—')}</span></div>${j.status !== 'running' ? `<form id="job-profile-form" data-job-id="${j.id}" class="section-gap"><label>Compte pour cette tâche<select id="detail-profile">${opts([['inherit', 'Compte par défaut du site'], ['', 'Sans compte'], ...data.state.profiles.filter(p => p.site === j.site || p.site === 'generic').map(p => [p.id, p.name])], jobProfileMode(j) === 'inherit' ? 'inherit' : jobProfileMode(j) === 'none' ? '' : j.profileId)}</select></label><div class="settings-footer"><button class="primary">Enregistrer le compte de cette tâche</button></div></form>` : ''}<h3>Journal</h3><pre id="logs" class="logs section-gap">Chargement…</pre><div class="settings-footer"><button data-action="cancel" data-id="${j.id}" class="danger">Annuler cette tâche</button><button data-close>Fermer</button></div>`); modalJob = id; await refreshLog(); }
async function refreshLog() { const id = modalJob; const logs = await api('logs?id=' + encodeURIComponent(id)); if (modalJob === id && $('#logs'))
    $('#logs').textContent = logs.length ? logs.map((l) => `${date(l.at)}  ${l.line}`).join('\n') : 'Aucun message pour le moment.'; }
document.addEventListener('click', e => {
    const b = e.target.closest('button');
    if (!b)
        return;
    run(async () => {
        if (b.id === 'oauth-start') {
            await generateOAuth();
            return;
        }
        if (b.id === 'oauth-cancel') {
            cancelOAuth();
            $('#oauth-status').textContent = 'Connexion annulée.';
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
                $('#' + b.dataset.pick).value = r.path;
                if (b.dataset.pick === 'global-dest')
                    await saveGlobalDestination();
            }
            preview();
            return;
        }
        if (b.dataset.action) {
            if (b.dataset.action === 'delete' && !confirm('Retirer cette tâche de la liste ? Les fichiers téléchargés seront conservés.'))
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
            const p = data.state.profiles.find(x => x.id === b.dataset.profileEdit);
            await openSite(p.site, p);
            return;
        }
        if (b.dataset.profileDelete) {
            if (!confirm('Supprimer ce compte enregistré ?'))
                return;
            await api('profile/delete', { id: b.dataset.profileDelete });
            await load(true);
            renderAccounts();
            return;
        }
        if (b.dataset.profileCheck) {
            const r = await api('profile/validate', { id: b.dataset.profileCheck });
            toast(r.message);
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
                            status.textContent = data.state.active ? 'Arrêt des tâches en cours…' : 'File en pause : vous pouvez importer.';
                        if (!data.state.active)
                            break;
                        if (Date.now() > until)
                            throw Error('Arrêt toujours en cours. Attendez puis réessayez.');
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
                    toast('Aucune tâche éligible.');
                    return;
                }
                if (data.state.running || data.state.active)
                    throw Error('Mettez la file en pause et attendez l’arrêt des tâches.');
                if (!confirm(`Appliquer le dossier du site ou le dossier global aux ${ids.length} tâches jamais démarrées ? Les tâches commencées et les destinations personnalisées sont conservées.`))
                    return;
                for (let i = 0; i < ids.length; i += 500)
                    await api('jobs/destinations', { ids: ids.slice(i, i + 500) });
                toast('Destinations actualisées. Reprise manuelle.');
                break;
            }
            case 'start-all':
                await api('queue/start', {});
                break;
            case 'pause-all':
                await api('queue/pause', {});
                break;
            case 'reset-quota':
                if (!confirm('Remettre le compteur à zéro ? Les tâches actives seront mises en pause. L’historique et les fichiers restent conservés.'))
                    return;
                await api('quota/reset', {});
                toast('Compteur remis à zéro. Reprise uniquement sur votre clic.');
                break;
            case 'clear-history':
                if (!confirm('Retirer de l’historique les tâches terminées ? Les fichiers restent conservés.'))
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
document.addEventListener('change', e => { const el = e.target; if (el.id === 'global-dest')
    run(saveGlobalDestination); if (el.id === 'site-use-global')
    updateSiteDestination(); if (el.id === 'override-media')
    $('#job-filter-options').classList.toggle('hidden', !el.checked); if (el.id === 'site-media-override')
    $('#site-filter-options').classList.toggle('hidden', !el.checked); if (el.id === 'profile-method')
    toggleMethod(); if (el.id === 'profile-select')
    run(() => showProfileEditor(el.dataset.siteId, el.value)); if (el.id === 'site-default' && el.value) {
    const sel = $('#profile-select');
    sel.value = el.value;
    run(() => showProfileEditor(sel.dataset.siteId, el.value));
} if (el.id === 'global-layout')
    preview(); });
document.addEventListener('input', e => { const el = e.target; if (el.id === 'history-search') {
    search = el.value;
    updateJobs();
} if (el.id === 'global-dest') {
    preview();
    const status = $('#global-save-status');
    status.className = 'footer-note';
    status.textContent = 'Dossier modifié — enregistrement à la sortie du champ.';
} if (el.id === 'urls') {
    const lines = el.value.split(/\r?\n/).filter(x => x.trim());
    const hosts = new Set(lines.map(x => { try {
        return new URL(x).hostname.replace(/^www\./, '');
    }
    catch {
        return 'Adresse à vérifier';
    } }));
    $('#detected').textContent = lines.length ? `${lines.length} adresse(s) · ${[...hosts].slice(0, 3).join(', ')}` : 'Site détecté automatiquement';
} });
document.addEventListener('submit', e => {
    e.preventDefault();
    const form = e.target;
    run(async () => {
        const button = form.querySelector('button[type="submit"],.settings-footer .primary,.add-row .primary');
        if (button)
            button.disabled = true;
        try {
            if (form.id === 'backup-form') {
                await submitBackup(form);
                return;
            }
            if (form.id === 'add-form') {
                const p = val('job-profile');
                const body = { urls: val('urls').split(/\r?\n/).map(x => x.trim()).filter(Boolean), destination: val('job-dest'), layout: val('job-layout') };
                if (p !== 'inherit')
                    body.profileId = p;
                if (check('override-media'))
                    body.filters = readFilters('job');
                const r = await api('jobs', body);
                toast(`${r.added} tâche(s) ajoutée(s).${r.errors.length ? '\n' + r.errors.join('\n') : ''}`, !!r.errors.length);
                if (!r.errors.length)
                    $('#urls').value = '';
                await load(true);
            }
            if (form.id === 'settings-form') {
                await saveGlobalDestination();
                const s = { ...data.state.settings, destination: val('global-dest'), layout: val('global-layout'), filename: val('global-filename'), filters: readFilters('global'), concurrency: Number(val('concurrency')), quotaBytes: Math.round(Number(val('quota')) * 1e9), rateKiB: Number(val('rate')), galleryPath: val('gallery-path'), ytdlpPath: val('ytdlp-path') };
                await api('settings', s);
                await load(true);
                toast('Paramètres enregistrés.');
            }
            if (form.id === 'site-form') {
                const id = form.dataset.siteId;
                const sc = { destination: check('site-use-global') ? '' : val('site-dest').trim(), layout: val('site-layout'), filename: val('site-filename'), profileId: val('site-default') };
                if (check('site-media-override'))
                    sc.filters = readFilters('site');
                const s = { ...data.state.settings, sites: { ...data.state.settings.sites, [id]: sc } };
                await api('settings', s);
                await load(true);
                toast('Réglages du site enregistrés.');
                closeModal();
            }
            if (form.id === 'job-profile-form') {
                const p = val('detail-profile');
                await api('job/profile', { id: form.dataset.jobId, mode: p === 'inherit' ? 'inherit' : p ? 'explicit' : 'none', profileId: p === 'inherit' ? '' : p });
                await load(true);
                await openDetail(form.dataset.jobId);
                toast('Compte affecté. Cliquez sur Démarrer / reprendre.');
            }
            if (form.id === 'profile-form') {
                await api('profile', { id: form.dataset.profileId, name: val('profile-name'), site: form.dataset.siteId, method: val('profile-method'), userId: val('profile-user'), token: val('profile-token'), tokenSecret: val('profile-secret'), cookiePath: val('profile-cookie'), setDefault: check('profile-default') });
                await load(true);
                closeModal();
                renderAccounts();
                toast('Compte enregistré et relu avec succès. La reprise des téléchargements reste manuelle.');
            }
        }
        finally {
            if (button)
                button.disabled = false;
        }
    });
});
$('#modal').addEventListener('close', () => { cancelOAuth(); modalJob = ''; $('#modal-content').innerHTML = ''; });
run(load);
setInterval(() => run(load), 1200);
function canRefreshDestination(j) { return !j.started && !j.bytes && !j.files && !j.skipped && !['running', 'completed', 'cancelled'].includes(j.status) && j.destinationMode !== 'explicit'; }
function openBackup(mode) {
    const exporting = mode === 'export';
    showModal(`<div class="modal-head"><h2>${exporting ? 'Exporter les sites et comptes' : 'Importer les sites et comptes'}</h2><button data-close>×</button></div>
 <div class="notice">${exporting ? 'Cette sauvegarde contient vos clés, tokens et cookies de connexion. Toute personne possédant le fichier et la phrase secrète pourra accéder à ces comptes.' : 'Importez uniquement une sauvegarde de confiance. Les comptes seront ajoutés au coffre de cette session ; aucun téléchargement ne sera lancé.'}</div>
 <form id="backup-form" data-mode="${mode}" class="section-gap"><div class="form-grid">
 <label class="full">${exporting ? 'Nouveau fichier de sauvegarde' : 'Fichier de sauvegarde'}${pathInput('backup-path', '', 'Fichier .dlmbackup', exporting ? 'backup-export' : 'backup-import')}</label>
 <label class="full">Phrase secrète (16 caractères minimum)<input id="backup-phrase" type="password" autocomplete="${exporting ? 'new-password' : 'off'}" required></label>
 ${exporting ? `<label class="full">Confirmer la phrase secrète<input id="backup-confirm" type="password" autocomplete="new-password" required></label>
 <label class="full check"><input id="backup-ack" type="checkbox" required>Je comprends que ce fichier contient mes accès aux comptes.</label>` : `<label class="full check"><input id="backup-replace" type="checkbox">Remplacer les réglages des sites déjà configurés</label>`}
 </div><p class="footer-note">${exporting ? 'Le fichier est chiffré. Choisissez une phrase longue et unique : elle ne sera pas enregistrée et ne pourra pas être récupérée. Utilisez un nouveau nom de fichier à chaque export.' : 'Les comptes existants sont conservés. Réimporter le même fichier ajoute des doublons. Les chemins incompatibles avec cet OS héritent du dossier global ; vérifiez vos destinations après l’import.'}<br>${exporting ? 'L’export est possible pendant les téléchargements.' : 'L’import nécessite une file en pause et l’arrêt des tâches actives.'} Les téléchargements, l’historique et les réglages globaux ne font pas partie de cette sauvegarde.</p>
 ${!exporting ? '<div class="row wrap"><button type="button" id="backup-pause">Mettre la file en pause</button><span id="backup-queue-status" class="smalltext muted" role="status"></span></div>' : ''}
 <div class="settings-footer"><button type="button" data-close>Annuler</button><button class="primary" type="submit">${exporting ? 'Chiffrer et exporter' : 'Déchiffrer et importer'}</button></div></form>`);
}
async function submitBackup(form) {
    const exporting = form.dataset.mode === 'export';
    const phrase = val('backup-phrase');
    if ([...phrase].length < 16)
        throw Error('La phrase secrète doit contenir au moins 16 caractères.');
    if (exporting && phrase !== val('backup-confirm'))
        throw Error('Les deux phrases secrètes ne correspondent pas.');
    const body = { path: val('backup-path'), passphrase: phrase };
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
        toast(exporting ? `Sauvegarde chiffrée créée : ${r.accounts} compte(s), ${r.sites} site(s).` : `Import terminé : ${r.accounts} compte(s) ajouté(s), ${r.sites} site(s) appliqué(s).${r.resetPaths ? ' ' + r.resetPaths + ' chemin(s) remis au dossier global.' : ''}`);
    }
    finally {
        form.querySelectorAll('input[type=password]').forEach(input => input.value = '');
        body.passphrase = '';
        body.confirmation = '';
    }
}
