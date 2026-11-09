// Run against a freshly built binary; all state is isolated in a temporary directory.
// Requires Playwright and Chrome (DLM_TEST_BROWSER overrides the browser executable).
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { spawn } from 'node:child_process';
import { mkdtemp, readFile } from 'node:fs/promises';
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
 const first=join(dir,'first-global'),second=join(dir,'second-global'),custom=join(dir,'site-custom');
 await page.locator('[data-page=settings]').click();await page.locator('#global-dest').fill(first);
 // Navigate immediately: no bottom-of-page save button and no artificial blur delay.
 await page.locator('[data-page=accounts]').click();
 await waitUntil(async()=>(await state()).settings.destination===first,'Global destination was lost on navigation');
 await page.locator('[data-site=instagram]').click();
 assert.equal(await page.locator('#site-use-global').isChecked(),true);
 assert.equal(await page.locator('#site-dest').inputValue(),first);
 assert.equal(await page.locator('#site-dest').isDisabled(),true);
 await page.locator('#site-form button.primary').click();
 assert.equal((await state()).settings.sites.instagram.destination,'');
 await page.locator('[data-site=rule34]').click();await page.locator('#site-use-global').uncheck();await page.locator('#site-dest').fill(custom);await page.locator('#site-form button.primary').click();
 await page.locator('[data-page=settings]').click();await page.locator('#global-dest').fill(second);await page.locator('[data-page=accounts]').click();
 await waitUntil(async()=>(await state()).settings.destination===second,'Second global change not saved');
 assert.equal((await state()).settings.sites.rule34.destination,custom);
 await page.locator('[data-site=instagram]').click();assert.equal(await page.locator('#site-dest').inputValue(),second);await page.locator('#modal [data-close]').first().click();
 const added=await post('jobs',{urls:['https://www.instagram.com/test-only/']});assert.equal(added.body.added,1);assert.equal((await state()).jobs[0].destination,second);
 // Remove the synthetic task before starting an empty scheduler: no remote download.
 await post('job',{id:(await state()).jobs[0].id,action:'delete'});await post('queue/start',{});
 await page.locator('#backup-export').click();const backup=join(dir,'sites.dlmbackup');await page.locator('#backup-path').fill(backup);await page.locator('#backup-phrase').fill('synthetic long phrase for testing');await page.locator('#backup-confirm').fill('synthetic long phrase for testing');await page.locator('#backup-ack').check();await page.locator('#backup-form button[type=submit]').click();await page.locator('#modal').waitFor({state:'hidden'});
 assert.equal((await readFile(backup)).subarray(0,8).toString(),'DLMBK001');assert.equal((await state()).running,true);
 await page.locator('#backup-import').click();await page.locator('#backup-pause').click();await waitUntil(async()=>!(await state()).running,'Import pause did not stop scheduler');await page.locator('#backup-path').fill(backup);await page.locator('#backup-phrase').fill('synthetic long phrase for testing');await page.locator('#backup-form button[type=submit]').click();await page.locator('#modal').waitFor({state:'hidden'});assert.equal((await state()).running,false);
 await page.locator('[data-page=settings]').click();await page.locator('#global-dest').fill('invalid-relative');await page.locator('[data-page=accounts]').click();await page.locator('#global-save-status.error').waitFor();assert.equal((await state()).settings.destination,second);assert.equal(await page.locator('#global-dest').inputValue(),'invalid-relative');
 // The native picker is unavailable in preview; stub only that OS boundary.
 const picked=join(dir,'picked-global');
 await page.route('**/api/pick',route=>route.fulfill({status:200,contentType:'application/json',body:JSON.stringify({path:picked})}));
 await page.locator('[data-pick="global-dest"]').click();
 await waitUntil(async()=>(await state()).settings.destination===picked,'Picked global destination was not saved');
 await page.locator('[data-page=accounts]').click();await page.locator('[data-site=instagram]').click();
 assert.equal(await page.locator('#site-dest').inputValue(),picked);
 assert.equal((await state()).settings.sites.rule34.destination,custom);
 await page.setViewportSize({width:840,height:620});
 await page.screenshot({path:join(dir,'site-inheritance.png'),fullPage:true});
 await page.locator('#modal [data-close]').first().click();
 // Exercise the public state contract without accessing Tumblr or real credentials.
 const fixture=await (await fetch(url+'/api/state')).json();
 const failed={id:'test-tumblr-quota',url:'https://www.tumblr.com/test-only',site:'tumblr',destination:picked,destinationMode:'inherit',layout:'site_creator',filename:'site',profileId:'',profileMode:'none',status:'failed',added:'2026-11-09T14:17:11Z',started:'2026-11-09T14:17:11Z',ended:'2026-11-09T14:17:13Z',error:'[tumblr][error] Aborting - Rate limit will reset at 19:06:46',exitCode:4,bytes:0,files:0,skipped:0,current:'',order:0};
 fixture.state.jobs=[failed,{...failed,id:'test-completed',url:'https://www.tumblr.com/completed-only',status:'completed',error:''},{...failed,id:'test-other-error',url:'https://example.org/test-only',site:'generic',error:'Network unavailable'}];
 await page.route('**/api/state',route=>route.fulfill({status:200,contentType:'application/json',body:JSON.stringify(fixture)}));
 await page.reload();await page.locator('[data-page=downloads]').click();
 await page.locator('[data-detail=test-tumblr-quota]').waitFor({timeout:3000});
 const row=page.locator('tr').filter({has:page.locator('[data-detail=test-tumblr-quota]')});
 assert.match(await row.innerText(),/Quota de l’API Tumblr atteint/);
 assert.match(await row.innerText(),/19:06:46/);
 assert.equal(await row.locator('[data-action=retry]').count(),1);
 assert.equal(await page.locator('[data-detail=test-completed]').count(),0);
 assert.match(await page.locator('tr').filter({has:page.locator('[data-detail=test-other-error]')}).innerText(),/Network unavailable/);
 assert.equal(await row.locator('.job-error').evaluate(el=>getComputedStyle(el).whiteSpace),'normal');
 await page.setViewportSize({width:1100,height:800});await page.screenshot({path:join(dir,'visible-errors.png'),fullPage:true});
 await row.locator('[data-detail]').click();assert.match(await page.locator('#modal .detail-grid').innerText(),/Quota de l’API Tumblr atteint.*19:06:46/s);
 await page.locator('#modal [data-close]').first().click();
 await page.locator('[data-page=history]').click();assert.equal(await page.locator('[data-detail=test-tumblr-quota]').count(),1);assert.equal(await page.locator('[data-detail=test-completed]').count(),1);
 assert.deepEqual(jsErrors,[]);console.log('PASS: global autosave, inheritance, override, encrypted export with running scheduler, import pause, invalid path, visible errors and quota explanation; '+dir);
}finally{if(browser)await browser.close();child.kill();}
