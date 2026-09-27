#!/usr/bin/env python3
"""Offline OT-0 plan materialization. Never calls a cluster or changes Git."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[2]
BASE = Path(__file__).resolve().parent
CLUSTER = 'atlas-refactor-test-ot0'
CONTEXT = 'kind-' + CLUSTER
REPO = 'https://github.com/snkio027/atlas-refactor.git'
FIXTURE = 'experiments/ownership-transfer/fixture'
STAGES = {
    'A1': ('owner-a', True, 'success'),
    'A2': (None, True, 'released'),
    'A3': ('owner-b', True, 'blocked'),
    'A4': ('owner-b', False, 'success'),
    'A5': ('owner-b', True, 'success'),
    'A6-release': (None, True, 'released'),
    'A6-strict': ('owner-a', True, 'blocked'),
    'A6-transfer': ('owner-a', False, 'success'),
    'A6-restored': ('owner-a', True, 'success'),
}


def encoded(value):
    return (json.dumps(value, indent=2, sort_keys=True) + '\n').encode()


def write(path, value):
    data = value if isinstance(value, bytes) else encoded(value)
    with path.open('xb') as f:
        f.write(data)
    path.chmod(0o600)


def command(args):
    return subprocess.check_output([str(x) for x in args], cwd=ROOT, timeout=120)


def documents(data):
    decoder = json.JSONDecoder()
    result = []
    rest = data.decode().strip()
    while rest:
        value, end = decoder.raw_decode(rest)
        if value is not None:
            result.extend(value['items'] if value.get('kind') == 'List' else [value])
        rest = rest[end:].strip()
    return result


def options(strict):
    return ['ServerSideApply=true', 'FailOnSharedResource=' + str(strict).lower()]


def application(owner, strict, revision):
    return {
        'apiVersion': 'argoproj.io/v1alpha1', 'kind': 'Application',
        'metadata': {'name': owner, 'namespace': 'argocd'},
        'spec': {
            'project': 'ot0-probe',
            'source': {'repoURL': REPO, 'targetRevision': revision,
                       'path': FIXTURE},
            'destination': {'server': 'https://kubernetes.default.svc', 'namespace': 'ot0-probe'},
            'syncPolicy': {'syncOptions': options(strict)},
        },
    }


def images(value):
    if isinstance(value, dict):
        for k, v in value.items():
            if k == 'image' and isinstance(v, str):
                yield v
            else:
                yield from images(v)
    elif isinstance(value, list):
        for v in value:
            yield from images(v)


def argocd_config():
    # Argo v3.5.1's SettingsManager filters ConfigMaps by this part-of label.
    return {'apiVersion': 'v1', 'kind': 'ConfigMap',
            'metadata': {'name': 'argocd-cm', 'namespace': 'argocd', 'labels': {
                'app.kubernetes.io/name': 'argocd-cm',
                'app.kubernetes.io/part-of': 'argocd'}},
            'data': {'application.resourceTrackingMethod': 'annotation',
                     'timeout.reconciliation': '15s', 'timeout.reconciliation.jitter': '0s'}}


def prepare(revision, output, tools):
    if not re.fullmatch(r'[0-9a-f]{40}', revision):
        raise ValueError('a full reviewed commit SHA is required')
    if command(['git', 'rev-parse', 'HEAD']).decode().strip() != revision:
        raise ValueError('HEAD must equal the reviewed revision')
    if command(['git', 'status', '--porcelain']):
        raise ValueError('a clean reviewed checkout is required')
    lock = json.loads((ROOT / 'versions.lock.json').read_text())
    for path, digest in {**lock['assets'], lock['chart']: lock['chartSHA256']}.items():
        if hashlib.sha256((ROOT / path).read_bytes()).hexdigest() != digest:
            raise ValueError('locked artifact mismatch: ' + path)
    versions = {
        'helm': command([tools / 'helm', 'version', '--template', '{{.Version}}']).decode().strip(),
        'kind': command([tools / 'kind', 'version']).decode().strip(),
        'kubectl': json.loads(command([tools / 'kubectl', 'version', '--client', '-o', 'json']))['clientVersion']['gitVersion'],
        'yq': command([tools / 'yq', '--version']).decode().strip(),
    }
    for name, actual in versions.items():
        expected = json.loads((ROOT / 'platform/development/versions.lock.json').read_text()).get(name, lock.get(name))
        if name in ('helm', 'kubectl'):
            valid = actual == 'v' + expected
        elif name == 'kind':
            valid = actual.split()[1] == 'v' + expected
        else:
            valid = actual.endswith('version v' + expected)
        if not valid:
            raise ValueError('wrong locked tool version: ' + name)
    os.umask(0o077)
    output.mkdir(parents=True, exist_ok=False, mode=0o700)
    # Helm is a renderer only. Server/ApplicationSet replicas are zero; no UI,
    # ingress, root Application, Cilium or Atlas handoff state is installed.
    yaml = command([tools / 'helm', 'template', 'argocd', lock['chart'],
                    '--namespace', 'argocd', '--include-crds', '--skip-tests',
                    '--kube-version', lock['kubernetes'], '-f', 'assets/argocd-values.yaml',
                    '--set', 'server.replicas=0', '--set', 'applicationSet.replicas=0'])
    write(output / 'seed.yaml', yaml)
    raw = command([tools / 'yq', 'eval-all', '-o=json', '-I=0', '.', output / 'seed.yaml'])
    seed = documents(raw)
    if set(images(seed)) != {lock['argoImage'], lock['redisImage']}:
        raise ValueError('unexpected seed image set')
    write(output / 'seed.json', {'apiVersion': 'v1', 'kind': 'List', 'items': seed})
    write(output / 'argocd-namespace.json', {'apiVersion': 'v1', 'kind': 'Namespace', 'metadata': {'name': 'argocd'}})
    write(output / 'argocd-cm.json', argocd_config())
    project = {'apiVersion': 'argoproj.io/v1alpha1', 'kind': 'AppProject', 'metadata': {'name': 'ot0-probe', 'namespace': 'argocd'}, 'spec': {
        'sourceRepos': [REPO], 'destinations': [{'server': 'https://kubernetes.default.svc', 'namespace': 'ot0-probe'}],
        'clusterResourceWhitelist': [{'group': '', 'kind': 'Namespace'}],
        'namespaceResourceWhitelist': [{'group': '', 'kind': 'ConfigMap'}],
    }}
    write(output / 'project.json', project)
    write(output / 'kind.json', {'apiVersion': 'kind.x-k8s.io/v1alpha4', 'kind': 'Cluster',
        'networking': {'apiServerAddress': '127.0.0.1'}, 'nodes': [{'role': 'control-plane', 'image': lock['nodeImage']}]})
    for stage, (owner, strict, _) in STAGES.items():
        if owner:
            write(output / (stage + '-app.json'), application(owner, strict, revision))
            write(output / (stage + '-sync.json'), {'operation': {'initiatedBy': {'username': 'ot0-reviewed-ceremony'},
                'info': [{'name': 'ot0-stage', 'value': stage}],
                'sync': {'revision': revision, 'prune': False, 'syncOptions': options(strict)}}})
    hashes = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(output.iterdir())}
    write(output / 'plan.json', {'schema': 1, 'cluster': CLUSTER, 'context': CONTEXT, 'dockerContext': 'orbstack',
        'revision': revision, 'repository': REPO, 'fixturePath': FIXTURE,
        'fixtureSHA256': hashlib.sha256((BASE / 'fixture/resources.json').read_bytes()).hexdigest(),
        'images': {k: lock[k] for k in ['nodeImage', 'argoImage', 'redisImage']}, 'tools': versions,
        'stages': STAGES, 'filesSHA256': hashes,
        'authority': 'isolated OT-0 only; execution needs exact-target approval; no dev02 or OT-1 authorization'})
    print('OT-0 plan rendered locally: ' + str(output))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--revision', required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--tool-dir', type=Path, required=True)
    args = parser.parse_args()
    prepare(args.revision, args.output.resolve(), args.tool_dir.resolve())
