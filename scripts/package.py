"""Build clean portable/source archives from an explicit allowlist. Python 3.11+."""
import argparse
import hashlib
import io
import json
import pathlib
import tarfile
import zipfile

ROOT = pathlib.Path(__file__).resolve().parents[1]
OUT = ROOT.parents[1] / 'outputs'
VERSION = '0.3.2'
ENGINES = ('gallery-dl', 'yt-dlp', 'ffmpeg', 'ffprobe', 'deno')
DOCS = ('README.md', 'VALIDATION.md', 'THIRD_PARTY.md', 'LICENSE')


def sha(path):
    with path.open('rb') as file:
        return hashlib.file_digest(file, 'sha256').hexdigest()


def portable_entries(platform):
    linux = platform == 'linux'
    suffix = '' if linux else '.exe'
    tools = ROOT / ('tools/linux-x64' if linux else 'tools')
    required = {name + suffix for name in ENGINES}
    manifest_path = tools / 'manifest.json'
    manifest = json.loads(manifest_path.read_text(encoding='utf-8'))
    verified = set()
    for engine in manifest:
        if not engine['verifiedAgainstPublishedChecksum']:
            raise RuntimeError('Engine missing published checksum: ' + engine['name'])
        for name, expected in engine['files'].items():
            if name not in required or name in verified:
                raise RuntimeError('Unexpected engine entry: ' + name)
            if sha(tools / name) != expected:
                raise RuntimeError('Engine checksum mismatch: ' + name)
            verified.add(name)
    if verified != required:
        raise RuntimeError('Missing tool: ' + ', '.join(sorted(required - verified)))

    executable = 'dl-manager' + suffix
    binary = ROOT / ('bin/linux-x64' if linux else 'bin') / executable
    entries = [(binary, executable)]
    entries += [(ROOT / name, name) for name in DOCS]
    entries += [(tools / name, 'tools/' + name) for name in required | {'manifest.json'}]
    entries += [(p, 'tools/' + p.relative_to(tools).as_posix())
                for p in (tools / 'licenses').rglob('*') if p.is_file()]
    entries += [(p, p.relative_to(ROOT).as_posix()) for p in (ROOT / 'licenses').rglob('*') if p.is_file()]
    return entries


def source_entries(platforms):
    names = ['main.go', 'main_windows.go', 'main_linux.go', 'go.mod', 'go.sum', 'package.json',
             'build.ps1', 'build-linux.ps1', 'build-linux.sh', 'test.ps1', '.gitignore', *DOCS,
             'docs/design.md', 'docs/progress.md', 'docs/linux-implementation-report.md',
             'docs/superpowers/plans/2026-11-08-dl-manager.md',
             'docs/superpowers/plans/2026-11-09-profiles-linux.md',
              'docs/superpowers/plans/2026-11-09-destinations-backup.md',
              'docs/superpowers/specs/2026-11-09-destinations-backup-design.md',
             'docs/superpowers/plans/2026-11-09-0.3.1.md',
             'docs/superpowers/specs/2026-11-09-0.3.1-design.md',
             'docs/releases/0.3.1.md',
             'docs/releases/0.3.2.md',
             'docs/superpowers/plans/2026-11-09-visible-errors.md']
    if (ROOT / '.gitattributes').is_file():
        names.append('.gitattributes')
    entries = [(ROOT / name, name) for name in names]
    for folder in ['internal', 'frontend', 'scripts', 'licenses']:
        entries += [(p, p.relative_to(ROOT).as_posix()) for p in (ROOT / folder).rglob('*')
                    if p.is_file() and '__pycache__' not in p.parts and 'node_modules' not in p.parts]
    workflows = ROOT / '.github/workflows'
    if workflows.is_dir():
        entries += [(p, p.relative_to(ROOT).as_posix()) for p in workflows.iterdir()
                    if p.is_file() and p.suffix in {'.yaml', '.yml'}]
    for platform in platforms:
        name = 'tools/linux-x64/manifest.json' if platform == 'linux' else 'tools/manifest.json'
        entries.append((ROOT / name, name))
    return entries


def verify_entries(entries):
    for path, name in entries:
        if not path.is_file() or path.is_symlink() or not path.resolve().is_relative_to(ROOT):
            raise RuntimeError('Unexpected package entry: ' + str(path))
        if any(part in {'data', 'downloads', 'temporary', 'credentials'}
               for part in pathlib.PurePosixPath(name).parts):
            raise RuntimeError('Private/test data cannot enter archive: ' + name)


def package(target, entries, linux):
    entries = sorted(entries, key=lambda item: item[1])
    verify_entries(entries)
    digests = {name: sha(path) for path, name in entries}
    checksums = ''.join(f'{digest}  {name}\n' for name, digest in digests.items())
    temporary = target.with_suffix(target.suffix + '.partial')
    try:
        if linux:
            executables = {'dl-manager', *('tools/' + name for name in ENGINES)}
            with tarfile.open(temporary, 'w:gz', compresslevel=6) as archive:
                for path, name in entries:
                    info = archive.gettarinfo(str(path), 'dl-manager/' + name)
                    info.mode = 0o755 if name in executables else 0o644
                    info.uid = info.gid = 0
                    info.uname = info.gname = ''
                    with path.open('rb') as file:
                        archive.addfile(info, file)
                payload = checksums.encode('utf-8')
                info = tarfile.TarInfo('dl-manager/SHA256SUMS.txt')
                info.mode = 0o644
                info.size = len(payload)
                archive.addfile(info, io.BytesIO(payload))
            with tarfile.open(temporary, 'r:gz') as archive:
                remaining = digests.copy()
                remaining['SHA256SUMS.txt'] = hashlib.sha256(payload).hexdigest()
                for member in archive:
                    name = member.name.removeprefix('dl-manager/')
                    expected_mode = 0o755 if name in executables else 0o644
                    with archive.extractfile(member) as file:
                        actual = hashlib.file_digest(file, 'sha256').hexdigest()
                    if actual != remaining.pop(name, None) or member.mode != expected_mode:
                        raise RuntimeError('Tar integrity failed: ' + member.name)
                if remaining:
                    raise RuntimeError('Tar entries missing: ' + ', '.join(remaining))
        else:
            with zipfile.ZipFile(temporary, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=6) as archive:
                for path, name in entries:
                    archive.write(path, 'dl-manager/' + name)
                archive.writestr('dl-manager/SHA256SUMS.txt', checksums)
            with zipfile.ZipFile(temporary) as archive:
                if bad := archive.testzip():
                    raise RuntimeError('ZIP integrity failed: ' + bad)
        temporary.replace(target)
    finally:
        temporary.unlink(missing_ok=True)
    result = {'file': str(target), 'size': target.stat().st_size, 'sha256': sha(target), 'entries': len(entries) + 1}
    print(json.dumps(result), flush=True)
    return result


def build(platform='windows', output=OUT):
    output.mkdir(parents=True, exist_ok=True)
    platforms = ('windows', 'linux') if platform == 'all' else (platform,)
    bundles = [(f'{item}-x64', portable_entries(item), item == 'linux') for item in platforms]
    bundles.append(('sources', source_entries(platforms), False))
    results = []
    for label, entries, linux in bundles:
        suffix = '.tar.gz' if linux else '.zip'
        target = output / f'dl-manager-{VERSION}-{label}{suffix}'
        results.append(package(target, entries, linux))
    (output / f'dl-manager-{VERSION}-SHA256SUMS.txt').write_text(
        ''.join(f"{item['sha256']}  {pathlib.Path(item['file']).name}\n" for item in results), encoding='utf-8')
    return results


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--platform', choices=('windows', 'linux', 'all'), default='windows')
    parser.add_argument('--output', type=pathlib.Path, default=OUT)
    args = parser.parse_args()
    build(args.platform, args.output)
