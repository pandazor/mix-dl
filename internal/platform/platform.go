package platform

type Site struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Methods []string `json:"methods"`
	Guide   string   `json:"guide"`
	URL     string   `json:"url"`
	Creator string   `json:"-"`
}

var Sites = []Site{
	{"artstation", "ArtStation", []string{"cookies"}, "Les portfolios publics peuvent être téléchargés sans compte. Pour un contenu nécessitant une connexion, connectez-vous vous-même puis importez les cookies de ce site au format Netscape.", "https://www.artstation.com/", "{userinfo[username]|username|artist|unknown}"},
	{"rule34", "Rule34.xxx", []string{"api", "cookies"}, "Connectez-vous sur le site, ouvrez les options de votre compte puis la rubrique API. Copiez User ID et API Key dans les champs ci-dessous. Ne collez pas la clé dans une URL.", "https://rule34.xxx/index.php?page=account&s=options", "{search_tags|tags|unknown}"},
	{"instagram", "Instagram", []string{"cookies"}, "Connectez-vous dans votre navigateur. Utilisez une extension d'export cookies.txt au format Netscape (par exemple cookies.txt sur Firefox). Exportez seulement les cookies d'instagram.com puis sélectionnez le fichier ici. Renouvelez cet export si la session expire. Le programme ne pilote pas le navigateur.", "https://www.instagram.com/", "{username|user[username]|unknown}"},
	{"tiktok", "TikTok", []string{"cookies"}, "Connectez-vous dans votre navigateur, exportez les cookies de tiktok.com au format Netscape, puis importez cookies.txt ici.", "https://www.tiktok.com/", "{user|author[unique_id]|username|unknown}"},
	{"tumblr", "Tumblr", []string{"oauth1", "cookies"}, "Pour OAuth, suivez la documentation gallery-dl (oauth:tumblr), puis renseignez OAuth Token et OAuth Token Secret. Un export cookies.txt peut être utilisé pour les contenus compatibles. Les actions de connexion restent manuelles.", "https://gdl-org.github.io/docs/configuration.html#extractor-tumblr-access-token", "{name|blog_name|username|unknown}"},
	{"patreon", "Patreon", []string{"cookies"}, "Connectez-vous sur Patreon avec le compte ayant accès au contenu. Exportez les cookies de patreon.com en format Netscape puis importez le fichier. L'application n'accorde aucun accès supplémentaire.", "https://www.patreon.com/", "{creator[vanity]|creator[full_name]|campaign[name]|unknown}"},
	{"deviantart", "DeviantArt", []string{"token", "cookies"}, "Les contenus publics peuvent fonctionner sans compte. Pour les contenus du compte, suivez la connexion OAuth de gallery-dl (oauth:deviantart), puis copiez le refresh-token obtenu. Vous pouvez aussi importer des cookies pour les contenus compatibles.", "https://gdl-org.github.io/docs/configuration.html#extractor-deviantart-refresh-token", "{username|author[username]|unknown}"},
	{"pixiv", "Pixiv", []string{"token"}, "Pixiv attend un refresh-token. Suivez la procédure oauth:pixiv de la documentation gallery-dl dans votre navigateur, puis collez le refresh-token ici. Un mot de passe ou un export de cookies seul ne remplace pas ce token.", "https://gdl-org.github.io/docs/configuration.html#extractor-pixiv-refresh-token", "{user[id]}_{user[name]}"},
	{"youtube", "YouTube", []string{"cookies"}, "Les vidéos publiques ne nécessitent généralement pas de compte. Si YouTube demande une connexion, exportez vos cookies YouTube au format Netscape depuis le navigateur, puis importez le fichier ici.", "https://github.com/yt-dlp/yt-dlp/wiki/Extractors#youtube", ""},
	{"generic", "Autres sites", []string{"cookies"}, "Toute URL HTTP(S) acceptée par gallery-dl peut être ajoutée. Importez au besoin les cookies du site au format Netscape. Le nom de créateur utilise les métadonnées disponibles, sinon le domaine.", "https://gdl-org.github.io/docs/supportedsites.html", "{username|user[name]|artist|author|unknown}"},
}

func Get(id string) Site {
	for _, s := range Sites {
		if s.ID == id {
			return s
		}
	}
	return Sites[len(Sites)-1]
}
