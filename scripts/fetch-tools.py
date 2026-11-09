"""Fetch official, checksummed engine binaries. No system installation."""
import argparse, concurrent.futures, hashlib, json, pathlib, shutil, tarfile, urllib.request, zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
TOOLS = ROOT / 'tools'
CACHE = ROOT.parent / 'toolchain' / 'downloads'
HEADERS = {'User-Agent':'dl-manager-build/0.1'}
def get(url):
    with urllib.request.urlopen(urllib.request.Request(url,headers=HEADERS), timeout=60) as r:
        return r.read()
def api(url): return json.loads(get(url))
def fetch(url, name, expected=None):
    target=CACHE/name
    if not target.exists():
        print('Downloading '+name,flush=True)
        temporary = target.with_suffix(target.suffix + '.partial')
        try:
            with urllib.request.urlopen(urllib.request.Request(url, headers=HEADERS), timeout=90) as response, temporary.open('wb') as output:
                shutil.copyfileobj(response, output)
            if expected and sha256(temporary) != expected:
                raise RuntimeError('Checksum mismatch: ' + name)
            temporary.replace(target)
        finally:
            temporary.unlink(missing_ok=True)
    digest = sha256(target)
    if expected and digest != expected:
        raise RuntimeError('Checksum mismatch in cache: ' + str(target))
    return target,digest
def sha256(path):
    with path.open('rb') as file:
        return hashlib.file_digest(file, 'sha256').hexdigest()

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

LINUX_RELEASES = {
    'gallery-dl': ('https://codeberg.org/api/v1/repos/mikf/gallery-dl', 'v1.32.19', 'gallery-dl.bin', 'SHA256SUMS'),
    'yt-dlp': ('https://api.github.com/repos/yt-dlp/yt-dlp', '2026.10.26', 'yt-dlp_linux', 'SHA2-256SUMS'),
    'deno': ('https://api.github.com/repos/denoland/deno', 'v2.9.8', 'deno-x86_64-unknown-linux-gnu.zip', None),
    'ffmpeg': ('https://api.github.com/repos/yt-dlp/FFmpeg-Builds', 'autobuild-2026-11-08-18-52',
               'ffmpeg-N-128713-g0f89d936b3-linux64-gpl.tar.xz', 'checksums.sha256'),
}


def install_linux(kind):
    base, version, filename, checksums = LINUX_RELEASES[kind]
    info = api(f'{base}/releases/tags/{version}')
    binary = asset(info, filename)
    if checksums:
        text = get(asset(info, checksums)['browser_download_url']).decode()
        expected = next(fields[0] for line in text.splitlines() if len(fields := line.split()) >= 2
                        and fields[-1].lstrip('*') == filename)
    else:
        expected = (binary.get('digest') or '').removeprefix('sha256:')
    if len(expected) != 64 or any(char not in '0123456789abcdef' for char in expected):
        raise RuntimeError('Missing valid published SHA-256: ' + filename)
    source = binary['browser_download_url']
    path, digest = fetch(source, f'{kind}-{version}-{filename}', expected)
    if kind == 'deno':
        with zipfile.ZipFile(path) as archive, archive.open('deno') as entry, (TOOLS / 'deno').open('wb') as output:
            shutil.copyfileobj(entry, output)
    elif kind == 'ffmpeg':
        names = {'ffmpeg', 'ffprobe'}
        with tarfile.open(path, 'r|xz') as archive:
            for member in archive:
                if not member.isfile():
                    continue
                filename = pathlib.PurePosixPath(member.name).name
                if '/bin/' in member.name and filename in names:
                    with archive.extractfile(member) as entry, (TOOLS / filename).open('wb') as output:
                        shutil.copyfileobj(entry, output)
                    names.remove(filename)
                elif filename.startswith(('LICENSE', 'COPYING')):
                    dest = TOOLS / 'licenses' / 'ffmpeg' / filename
                    dest.parent.mkdir(parents=True, exist_ok=True)
                    with archive.extractfile(member) as entry, dest.open('wb') as output:
                        shutil.copyfileobj(entry, output)
        if names:
            raise RuntimeError('Missing FFmpeg archive entries: ' + ', '.join(sorted(names)))
    else:
        shutil.copyfile(path, TOOLS / kind)
    print(f'{kind} ready {version}', flush=True)
    return {'name': kind, 'version': version, 'source': source, 'archiveSHA256': digest,
            'verifiedAgainstPublishedChecksum': True}


def main():
    global TOOLS
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--platform', choices=('windows', 'linux'), default='windows')
    platform = parser.parse_args().platform
    if platform == 'linux':
        TOOLS = TOOLS / 'linux-x64'
    TOOLS.mkdir(parents=True, exist_ok=True)
    CACHE.mkdir(parents=True, exist_ok=True)
    engines = ('gallery-dl', 'yt-dlp', 'deno', 'ffmpeg')
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        result = list(pool.map(install_linux if platform == 'linux' else run, engines))
    suffix = '.exe' if platform == 'windows' else ''
    for item in result:
        names = ('ffmpeg', 'ffprobe') if item['name'] == 'ffmpeg' else (item['name'],)
        item['files'] = {}
        for name in names:
            path = TOOLS / (name + suffix)
            path.chmod(0o755)
            item['files'][path.name] = sha256(path)
    gallery_version = LINUX_RELEASES['gallery-dl'][1] if platform == 'linux' else 'master'
    yt_version = LINUX_RELEASES['yt-dlp'][1] if platform == 'linux' else 'master'
    deno_version = LINUX_RELEASES['deno'][1] if platform == 'linux' else 'main'
    gallery_ref = f'tag/{gallery_version}' if platform == 'linux' else f'branch/{gallery_version}'
    for name, url in {
        'gallery-dl-COPYING.txt': f'https://codeberg.org/mikf/gallery-dl/raw/{gallery_ref}/LICENSE',
        'yt-dlp-UNLICENSE.txt': f'https://raw.githubusercontent.com/yt-dlp/yt-dlp/{yt_version}/LICENSE',
        'yt-dlp-third-party.txt': f'https://raw.githubusercontent.com/yt-dlp/yt-dlp/{yt_version}/THIRD_PARTY_LICENSES.txt',
        'deno-LICENSE.txt': f'https://raw.githubusercontent.com/denoland/deno/{deno_version}/LICENSE.md',
    }.items():
        dst = TOOLS / 'licenses' / name
        dst.parent.mkdir(exist_ok=True)
        dst.write_bytes(get(url))
    (TOOLS / 'manifest.json').write_text(json.dumps(result, indent=2), encoding='utf-8')
    print('Engine package complete', flush=True)


if __name__ == '__main__':
    main()
