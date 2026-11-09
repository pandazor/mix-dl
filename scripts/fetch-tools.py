"""Fetch official, checksummed engine binaries. No system installation."""
import concurrent.futures, hashlib, json, pathlib, shutil, tarfile, urllib.request, zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
TOOLS = ROOT / 'tools'
CACHE = ROOT.parent / 'toolchain' / 'downloads'
TOOLS.mkdir(exist_ok=True); CACHE.mkdir(exist_ok=True)
HEADERS = {'User-Agent':'dl-manager-build/0.1'}
def get(url):
    with urllib.request.urlopen(urllib.request.Request(url,headers=HEADERS), timeout=60) as r:
        return r.read()
def api(url): return json.loads(get(url))
def fetch(url, name, expected=None):
    target=CACHE/name
    if not target.exists():
        print('Downloading '+name,flush=True)
        with urllib.request.urlopen(urllib.request.Request(url,headers=HEADERS), timeout=90) as r, target.open('wb') as f:
            shutil.copyfileobj(r,f)
    digest=hashlib.file_digest(target.open('rb'),'sha256').hexdigest()
    if expected and digest!=expected:
        raise RuntimeError('Checksum mismatch: '+name)
    return target,digest
def gh(repo): return api('https://api.github.com/repos/'+repo+'/releases/latest')
def asset(info,name):return next(a for a in info['assets'] if a['name']==name)
def run(kind):
    if kind=='gallery-dl':
        info=api('https://codeberg.org/api/v1/repos/mikf/gallery-dl/releases/latest')
        a=asset(info,'gallery-dl.exe');checks=get(asset(info,'SHA256SUMS')['browser_download_url']).decode()
        expected=next(line.split()[0] for line in checks.splitlines() if line.split()[-1].lstrip('*')=='gallery-dl.exe')
        path,digest=fetch(a['browser_download_url'],'gallery-dl-'+info['tag_name']+'.exe',expected)
        shutil.copy2(path,TOOLS/'gallery-dl.exe')
    elif kind=='yt-dlp':
        info=gh('yt-dlp/yt-dlp');a=asset(info,'yt-dlp.exe');checks=get(asset(info,'SHA2-256SUMS')['browser_download_url']).decode()
        expected=next(line.split()[0] for line in checks.splitlines() if line.split()[-1].lstrip('*')=='yt-dlp.exe')
        path,digest=fetch(a['browser_download_url'],'yt-dlp-'+info['tag_name']+'.exe',expected)
        shutil.copy2(path,TOOLS/'yt-dlp.exe')
    elif kind=='deno':
        info=gh('denoland/deno');a=asset(info,'deno-x86_64-pc-windows-msvc.zip')
        expected=a.get('digest','').removeprefix('sha256:') or None
        path,digest=fetch(a['browser_download_url'],'deno-'+info['tag_name']+'.zip',expected)
        with zipfile.ZipFile(path) as z: (TOOLS/'deno.exe').write_bytes(z.read('deno.exe'))
    elif kind=='ffmpeg':
        info=gh('yt-dlp/FFmpeg-Builds');a=asset(info,'ffmpeg-master-latest-win64-gpl.zip')
        expected=a.get('digest','').removeprefix('sha256:') or None
        path,digest=fetch(a['browser_download_url'],'ffmpeg-'+str(a['id'])+'.zip',expected)
        with zipfile.ZipFile(path) as z:
            for name in ('ffmpeg.exe','ffprobe.exe'):
                source=next(n for n in z.namelist() if n.endswith('/bin/'+name))
                with z.open(source) as src,(TOOLS/name).open('wb') as dst:shutil.copyfileobj(src,dst)
            for n in z.namelist():
                if '/LICENSE' in n or '/COPYING' in n:
                    dst=TOOLS/'licenses'/'ffmpeg'/pathlib.PurePosixPath(n).name;dst.parent.mkdir(parents=True,exist_ok=True);dst.write_bytes(z.read(n))
    else: raise ValueError(kind)
    print(kind+' ready '+info['tag_name'],flush=True)
    return {'name':kind,'version':info['tag_name'],'source':a['browser_download_url'],'archiveSHA256':digest,'verifiedAgainstPublishedChecksum':bool(expected)}

if __name__=='__main__':
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool: result=list(pool.map(run,['gallery-dl','yt-dlp','deno','ffmpeg']))
    for f in TOOLS.glob('*.exe'):
        for item in result:
            if f.stem==item['name'] or (item['name']=='ffmpeg' and f.stem=='ffprobe'):
                item.setdefault('files',{})[f.name]=hashlib.file_digest(f.open('rb'),'sha256').hexdigest()
    (TOOLS/'manifest.json').write_text(json.dumps(result,indent=2),encoding='utf-8')
    for name,url in {
      'gallery-dl-COPYING.txt':'https://codeberg.org/mikf/gallery-dl/raw/branch/master/LICENSE',
      'yt-dlp-UNLICENSE.txt':'https://raw.githubusercontent.com/yt-dlp/yt-dlp/master/LICENSE',
      'yt-dlp-third-party.txt':'https://raw.githubusercontent.com/yt-dlp/yt-dlp/master/THIRD_PARTY_LICENSES.txt',
      'deno-LICENSE.txt':'https://raw.githubusercontent.com/denoland/deno/main/LICENSE.md',
    }.items():
        dst=TOOLS/'licenses'/name;dst.parent.mkdir(exist_ok=True);dst.write_bytes(get(url))
    print('Engine package complete',flush=True)
