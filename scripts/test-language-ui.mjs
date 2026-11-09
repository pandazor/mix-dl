// Run against a freshly built binary; all state is isolated in a temporary directory.
// Requires Playwright and Chrome (DLM_TEST_BROWSER overrides the browser executable).
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { resolve, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import net from 'node:net';
const require=createRequire(import.meta.url);
const {chromium}=require('playwright');
const root=resolve(fileURLToPath(new URL('..',import.meta.url)));
const dir=await mkdtemp(join(tmpdir(),'dlm-ui-031-'));
const port=await new Promise((resolve,reject)=>{const s=net.createServer();s.on('error',reject);s.listen(0,'127.0.0.1',()=>{const p=s.address().port;s.close(()=>resolve(p));});});
const url='http://127.0.0.1:'+port;
const child=spawn(process.env.DLM_TEST_BINARY||join(root,'bin',process.platform==='win32'?'dl-manager.exe':'dl-manager'),['--preview','127.0.0.1:'+port,'--data',dir],{cwd:root,windowsHide:true,stdio:'pipe'});
let browser;let serverError='';child.stderr.on('data',b=>serverError+=b);child.on('error',e=>serverError+=e.message);
async function waitUntil(fn,description){const until=Date.now()+10000;while(Date.now()<until){if(await fn())return;await new Promise(r=>setTimeout(r,50));}throw Error(description+'; '+serverError);}
try{
 await waitUntil(async()=>{try{return (await fetch(url)).ok}catch{return false}},'Preview not ready');
 browser=await chromium.launch({headless:true,...((process.env.DLM_TEST_BROWSER||process.platform==='win32')?{executablePath:process.env.DLM_TEST_BROWSER||'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'}:{})});
 const page=await browser.newPage({viewport:{width:1100,height:800}});const jsErrors=[];page.on('pageerror',e=>jsErrors.push(e.message));
 await page.goto(url);await page.locator('[data-page=settings]').waitFor();
 const state=async()=>(await (await fetch(url+'/api/state')).json()).state;
 async function post(route,body){return page.evaluate(async({route,body})=>{const r=await fetch('/api/'+route,{method:'POST',headers:{'Content-Type':'application/json','X-DL-Token':document.querySelector('meta[name=dl-token]').content},body:JSON.stringify(body)});return {status:r.status,body:await r.json()}},{route,body});}

 const labels={fr:['Paramètres','Comptes & sites','Téléchargements','Historique'],en:['Settings','Accounts & sites','Downloads','History'],es:['Configuración','Cuentas y sitios','Descargas','Historial'],'zh-CN':['设置','账户与站点','下载','历史记录'],ja:['設定','アカウントとサイト','ダウンロード','履歴']};
 const catalog=await (await fetch(url+'/locales.json')).json();
 const before=await state();
 await page.locator('[data-page=settings]').click();
 // Draft values must survive a language switch, while only language is persisted.
 await page.locator('#rate').fill('177');await page.locator('#quota').fill('8.2');await page.locator('#global-layout').selectOption('flat');
 for(const lang of ['en','es','zh-CN','ja','fr']){
  await page.locator('#language').selectOption(lang);
  await waitUntil(async()=>await page.locator('html').getAttribute('lang')===lang&&await page.locator('#language').isEnabled(),'language not applied: '+lang);
  assert.equal(await page.locator('#main h1').innerText(),labels[lang][0]);
  assert.equal(await page.locator('#rate').inputValue(),'177');assert.equal(await page.locator('#quota').inputValue(),'8.2');assert.equal(await page.locator('#global-layout').inputValue(),'flat');
  assert.equal((await state()).settings.rateKiB,before.settings.rateKiB);assert.equal((await state()).settings.language,lang);
 }
 // Changes typed while the language request is pending must also survive.
 let releaseLanguage;const languageGate=new Promise(resolve=>releaseLanguage=resolve);let signalRequest;const requestedLanguage=new Promise(resolve=>signalRequest=resolve);
 await page.route('**/api/settings/language',async route=>{signalRequest();await languageGate;await route.continue()});
 await page.locator('#language').selectOption('en');await requestedLanguage;assert.equal(await page.locator('[data-page=accounts]').isDisabled(),true,'navigation must wait for language save');await page.locator('#rate').fill('188');releaseLanguage();
 await waitUntil(async()=>await page.locator('#main h1').innerText()==='Settings','delayed language');assert.equal(await page.locator('#rate').inputValue(),'188');
 await page.unroute('**/api/settings/language');
 // A stale poll must finish before the fresh load used by a language change.
 let releasePoll,signalPoll;const pollGate=new Promise(resolve=>releasePoll=resolve);const pollStarted=new Promise(resolve=>signalPoll=resolve);const stale=await (await fetch(url+'/api/state')).json();let held=false;
 await page.route('**/api/state',async route=>{if(!held){held=true;signalPoll();await pollGate;await route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(stale)})}else await route.continue()});
 await pollStarted;await page.locator('#language').selectOption('es');await waitUntil(async()=>(await state()).settings.language==='es','saved Spanish while poll held');releasePoll();
 await waitUntil(async()=>await page.locator('#main h1').innerText()===labels.es[0]&&await page.locator('#language').isEnabled(),'fresh state after stale poll');await page.unroute('**/api/state');
 assert.equal(await page.locator('html').getAttribute('lang'),'es');assert.equal(await page.locator('[data-page=accounts]').isEnabled(),true);
 // Exercise all screens and account/backup dialogues in each language, with real persisted settings.
 for(const lang of ['en','es','zh-CN','ja','fr']){
  await page.locator('#language').selectOption(lang);await waitUntil(async()=>await page.locator('#main h1').innerText()===labels[lang][0],'heading '+lang);
  await page.reload();await page.locator('[data-page=settings]').click();
  assert.equal(await page.locator('#language').inputValue(),lang);assert.equal(await page.locator('html').getAttribute('lang'),lang);
  await page.setViewportSize({width:840,height:720});
  await page.screenshot({path:join(dir,lang+'-settings.png'),fullPage:true});
  for(const [index,name] of [[1,'accounts'],[2,'downloads'],[3,'history']]){
   await page.locator('[data-page='+name+']').click();assert.equal(await page.locator('#main h1').innerText(),labels[lang][index]);
   assert.ok(await page.evaluate(()=>document.documentElement.scrollWidth<=window.innerWidth+1),'page overflows: '+lang+' '+name);
  }
  await page.locator('[data-page=accounts]').click();await page.locator('[data-site=deviantart]').click();
  assert.equal(await page.locator('#oauth-start').innerText(),catalog[lang]['Générer le token']);
  assert.equal(await page.locator('#site-use-global').isChecked(),true);assert.equal(await page.locator('#site-dest').inputValue(),before.settings.destination);
  assert.ok(!(await page.locator('.guide').innerText()).includes('Les contenus publics')||lang==='fr','untranslated guide');
  await page.screenshot({path:join(dir,lang+'-account.png'),fullPage:true});
  await page.locator('#modal [data-close]').first().click();await page.locator('#backup-export').click();
  assert.equal(await page.locator('#modal h2').innerText(),catalog[lang]['Exporter les sites et comptes']);await page.locator('#modal [data-close]').first().click();
  await page.locator('[data-page=settings]').click();
 }
 await page.locator('#language').selectOption('ja');await waitUntil(async()=>await page.locator('#main h1').innerText()===labels.ja[0],'Japanese heading');
 const result=await post('custom-site',{name:'<img src=x onerror=alert(1)> Paramètres',domain:'art.example.org'});assert.equal(result.status,200);await page.reload();
 await page.locator('[data-page=accounts]').click();assert.match(await page.locator('[data-site='+result.body.id+']').innerText(),/<img src=x onerror=alert\(1\)> Paramètres/);assert.equal(await page.locator('.site-card img').count(),0);
 await page.locator('[data-site='+result.body.id+']').click();assert.equal(await page.locator('#custom-name').inputValue(),'<img src=x onerror=alert(1)> Paramètres');
 await page.locator('#modal [data-close]').first().click();
 await page.locator('#custom-site-add').click();await page.locator('#custom-name').fill('duplicate');await page.locator('#custom-domain').fill('art.example.org');await page.locator('#custom-site-form button.primary').click();
 await waitUntil(async()=>(await page.locator('#toast').innerText())===catalog.ja['Un site utilise déjà ce domaine'],'backend validation translation');
 assert.deepEqual(jsErrors,[]);console.log('PASS: five languages, persisted reload, drafts preserved, translated dialogs/errors, escaped names, minimum width; '+dir);
}finally{if(browser)await browser.close();child.kill();}
