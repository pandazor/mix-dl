"""Build clean portable/source archives from an explicit allowlist. Python 3.11+."""
import hashlib
import json
import pathlib
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
OUT = ROOT.parents[1] / 'outputs'
VERSION = '0.1.0'

def sha(path):
    with path.open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()

def build():
    OUT.mkdir(parents=True, exist_ok=True)
    manifest = json.loads((ROOT / 'tools/manifest.json').read_text(encoding='utf-8'))
    for engine in manifest:
        for name, expected in engine['files'].items():
            path = ROOT / 'tools' / name
            if sha(path) != expected:
                raise RuntimeError('Engine checksum mismatch: ' + name)
    required = ['gallery-dl.exe', 'yt-dlp.exe', 'ffmpeg.exe', 'ffprobe.exe', 'deno.exe']
    for name in required:
        if not (ROOT / 'tools' / name).is_file():
            raise RuntimeError('Missing tool: ' + name)

    portable = [(ROOT / 'bin/dl-manager.exe', 'dl-manager.exe')]
    portable += [(ROOT / name, name) for name in ['README.md', 'VALIDATION.md', 'THIRD_PARTY.md']]
    for folder in ['tools', 'licenses']:
        portable += [(p, p.relative_to(ROOT).as_posix()) for p in (ROOT / folder).rglob('*') if p.is_file()]
    source_names = ['main.go', 'go.mod', 'go.sum', 'package.json', 'build.ps1', 'test.ps1', 'README.md', 'VALIDATION.md', 'THIRD_PARTY.md', '.gitignore',
                    'docs/design.md', 'docs/progress.md', 'docs/superpowers/plans/2026-11-08-dl-manager.md']
    sources = [(ROOT / name, name) for name in source_names]
    for folder in ['internal', 'frontend', 'scripts', 'licenses']:
        sources += [(p, p.relative_to(ROOT).as_posix()) for p in (ROOT / folder).rglob('*')
                    if p.is_file() and '__pycache__' not in p.parts and 'node_modules' not in p.parts]
    sources += [(ROOT / 'tools/manifest.json', 'tools/manifest.json')]

    results = []
    for label, entries in [('windows-x64', portable), ('sources', sources)]:
        target = OUT / f'dl-manager-{VERSION}-{label}.zip'
        checksums = []
        with zipfile.ZipFile(target, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=6) as z:
            for path, name in sorted(entries, key=lambda item:item[1]):
                if not path.is_file() or path.is_symlink() or not path.resolve().is_relative_to(ROOT):
                    raise RuntimeError('Unexpected package entry: ' + str(path))
                if any(part in {'data', 'downloads', 'temporary'} for part in pathlib.PurePosixPath(name).parts):
                    raise RuntimeError('Private/test data cannot enter archive: ' + name)
                z.write(path, 'dl-manager/' + name)
                checksums.append(f'{sha(path)}  {name}')
            z.writestr('dl-manager/SHA256SUMS.txt', '\n'.join(checksums) + '\n')
        with zipfile.ZipFile(target) as z:
            bad = z.testzip()
            if bad:
                raise RuntimeError('ZIP integrity failed: ' + bad)
        result = {'file':str(target), 'size':target.stat().st_size, 'sha256':sha(target), 'entries':len(entries)+1}
        results.append(result)
        print(json.dumps(result), flush=True)
    (OUT / f'dl-manager-{VERSION}-SHA256SUMS.txt').write_text(
        '\n'.join(f"{item['sha256']}  {pathlib.Path(item['file']).name}" for item in results)+'\n', encoding='utf-8')
    return results

if __name__ == '__main__':
    build()
