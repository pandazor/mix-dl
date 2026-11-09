export const languages: [string,string][] = [['fr','Français'],['en','English'],['es','Español'],['zh-CN','简体中文'],['ja','日本語']];
let selected='fr';
let catalog: Record<string,Record<string,string>> = {};
let patterns: {source:string; regex:RegExp; positions:number[]}[]=[];
export function language(){return selected;}
export function locale(){return ({fr:'fr-FR',en:'en-US',es:'es-ES','zh-CN':'zh-CN',ja:'ja-JP'} as Record<string,string>)[selected];}
export function setLanguage(value:string){selected=languages.some(([id])=>id===value)?value:'fr';document.documentElement.lang=selected;}
export async function loadCatalog(){
 const response=await fetch('/locales.json');if(!response.ok)throw Error('Impossible de charger les traductions. Relancez l’application.');
 catalog=await response.json();
 patterns=Object.keys(catalog.fr).filter(s=>/\{\d+\}/.test(s)).map(source=>{
  const positions:number[]=[];let p=0;let regex='^';
  for(const m of source.matchAll(/\{(\d+)\}/g)){regex+=escapeRE(source.slice(p,m.index))+'([\\s\\S]*?)';positions.push(Number(m[1]));p=m.index!+m[0].length}
  return {source,regex:new RegExp(regex+escapeRE(source.slice(p))+'$'),positions};
 }).sort((a,b)=>b.source.length-a.source.length);
}
function escapeRE(s:string){return s.replace(/[.*+?^$()|\[\]{}\\]/g,'\\$&');}
export function t(source:string,...values:unknown[]):string{
 const text=catalog[selected]?.[source]??source;
 return text.replace(/\{(\d+)\}/g,(match,n)=>Number(n)<values.length?String(values[Number(n)]??''):match);
}
// Only application messages pass through here. Names, URLs, paths, secrets and raw logs do not.
export function message(value:unknown):string{
 const source=String(value??'');if(!source||selected==='fr')return source;
 if(Object.hasOwn(catalog[selected]??{},source))return t(source);
 for(const {regex,source:pattern,positions} of patterns){const match=source.match(regex);if(match){const args:string[]=[];positions.forEach((n,i)=>args[n]=match[i+1]);return t(pattern,...args)}}
 for(const key of Object.keys(catalog[selected]??{}).filter(k=>k.endsWith(": " )).sort((a,b)=>b.length-a.length)){if(source.startsWith(key))return t(key)+source.slice(key.length)}
 return source;
}
export function formatBytes(n:number){
 const units=selected==='fr'?['o','Ko','Mo','Go','To']:['B','KB','MB','GB','TB'];let i=0;
 while(n>=1000&&i<units.length-1){n/=1000;i++}
 return new Intl.NumberFormat(locale(),{maximumFractionDigits:i<2?0:2}).format(n)+' '+units[i];
}
