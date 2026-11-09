import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
const root=new URL('../',import.meta.url);
const catalog=JSON.parse(await readFile(new URL('internal/i18n/catalog.json',root),'utf8'));
const keys=Object.keys(catalog.fr).sort();
const params=s=>[...s.matchAll(/\{\d+\}/g)].map(m=>m[0]).sort();
for(const lang of ['fr','en','es','zh-CN','ja']){
 assert.ok(catalog[lang],'Missing '+lang);
 assert.deepEqual(Object.keys(catalog[lang]).sort(),keys,lang+' keys differ');
 for(const key of keys){const value=catalog[lang][key];assert.ok(typeof value==='string'&&value.trim(),lang+' empty '+key);assert.deepEqual(params(value),params(key),lang+' placeholders '+key);assert.ok(!/<\/?[a-z][^>]*>/i.test(value),'HTML in '+lang+' '+key)}
}
const app=await readFile(new URL('frontend/app.ts',root),'utf8');
for(const match of app.matchAll(/\bt\(("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/g)){
 const key=Function('return '+match[1])();assert.ok(Object.hasOwn(catalog.fr,key),'Uncatalogued UI message: '+key);
}
const main=await readFile(new URL('main.go',root),'utf8');
for(const m of main.matchAll(/tr\("([^"]+)"\)/g))assert.ok(Object.hasOwn(catalog.fr,m[1]),'Uncatalogued native message '+m[1]);
globalThis.document={documentElement:{lang:'fr'}};
globalThis.fetch=async()=>({ok:true,json:async()=>catalog});
const runtime=await import(new URL('frontend/dist/i18n.js',root));await runtime.loadCatalog();
for(const lang of ['fr','en','es','zh-CN','ja']){runtime.setLanguage(lang);assert.equal(document.documentElement.lang,lang);assert.equal(runtime.t('Paramètres'),catalog[lang]['Paramètres']);assert.equal(runtime.message('Network: unrecognized issue'),'Network: unrecognized issue');const detail='<path> $& \$1 日本語';assert.ok(runtime.message('Destination inaccessible : '+detail).endsWith(detail));}
runtime.setLanguage('invalid');assert.equal(runtime.language(),'fr');
console.log('All 5 catalogs complete; '+keys.length+' messages, placeholders and UI coverage verified.');
