#!/usr/bin/env python3
"""Read-only OT-0 capture and assertion. Kubernetes access is GET/config-view only."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
from urllib.parse import urlparse

from prepare import application, encoded, options, STAGES, CLUSTER, CONTEXT, write

TRACKING = 'argocd.argoproj.io/tracking-id'
RESOURCE_KEYS = {'Namespace//ot0-probe', 'ConfigMap/ot0-probe/probe'}
OWNERS = {'A1': 'owner-a', 'A2': 'owner-a', 'A3': 'owner-a',
          'A4': 'owner-b', 'A5': 'owner-b', 'A6-release': 'owner-b',
          'A6-strict': 'owner-b', 'A6-transfer': 'owner-a', 'A6-restored': 'owner-a'}
PREVIOUS = {'A2': 'A1', 'A3': 'A2', 'A4': 'A3', 'A5': 'A4',
            'A6-release': 'A5', 'A6-strict': 'A6-release',
            'A6-transfer': 'A6-strict', 'A6-restored': 'A6-transfer'}


def require(value, message):
    if not value:
        raise ValueError(message)


def key(obj):
    m = obj['metadata']
    return obj['kind'] + '/' + m.get('namespace', '') + '/' + m['name']


def semantic(obj):
    """Remove only API lifecycle fields/status and the single tracking annotation.

    Preserve all other labels, annotations, spec, data, finalizers and ownerRefs.
    resourceVersion/managedFields are captured in full, but naturally change on SSA.
    """
    result = copy.deepcopy(obj)
    result.pop('status', None)
    metadata = result['metadata']
    for field in ['uid', 'resourceVersion', 'creationTimestamp', 'managedFields', 'generation']:
        metadata.pop(field, None)
    annotations = metadata.get('annotations', {})
    annotations.pop(TRACKING, None)
    if not annotations:
        metadata.pop('annotations', None)
    return result


def check(snapshot, plan, baseline=None, previous=None):
    stage = snapshot['stage']
    require(stage in STAGES, 'unknown stage')
    require(snapshot['cluster'] == CLUSTER and snapshot['revision'] == plan['revision'], 'wrong target/revision')
    require(snapshot['trackingMethod'] == 'annotation', 'annotation tracking required')
    resources = {key(o): o for o in snapshot['objects']}
    require(len(snapshot['objects']) == 2 and set(resources) == RESOURCE_KEYS, 'wrong resource set')
    owner, strict, outcome = STAGES[stage]
    tracking_owner = OWNERS[stage]
    for k, obj in resources.items():
        m = obj['metadata']
        require(m.get('uid') and m.get('resourceVersion') and m.get('managedFields'), 'missing identity/evidence')
        require(not m.get('deletionTimestamp'), 'probe resource is terminating')
        expected_tracking = tracking_owner + ':/' + obj['kind'] + ':ot0-probe/' + m['name']
        require(m.get('annotations', {}).get(TRACKING) == expected_tracking, 'wrong tracking: ' + k)
        require(any(f.get('manager') == 'argocd-controller' and f.get('operation') == 'Apply'
                    for f in m['managedFields']), 'missing Argo SSA ownership: ' + k)
    apps = snapshot['applications']
    if owner is None:
        require(apps == [], 'source Application still exists or another owner exists')
    else:
        require(len(apps) == 1 and apps[0]['metadata']['name'] == owner, 'source/target ownership overlap')
        app = apps[0]
        require(not app['metadata'].get('finalizers'), 'Application has a finalizer')
        require(not app['metadata'].get('deletionTimestamp'), 'Application is terminating')
        expected = application(owner, strict, plan['revision'])['spec']
        for field in ['project', 'source', 'destination']:
            require(app['spec'][field] == expected[field], 'unexpected Application ' + field)
        require(set(app['spec']['syncPolicy']['syncOptions']) == set(options(strict)), 'wrong strict/SSA mode')
        require(not app['spec']['syncPolicy'].get('automated'), 'probe must use explicit sync requests')
        require(not app.get('operation'), 'Application operation still active')
        status = app['status']
        op = status['operationState']
        require(op.get('finishedAt'), 'operation has not finished')
        require(op['operation'].get('info') == [{'name': 'ot0-stage', 'value': stage}], 'stale operation evidence')
        sync_request = op['operation']['sync']
        require(sync_request['revision'] == plan['revision'] and not sync_request.get('prune'), 'wrong requested sync')
        require(set(sync_request['syncOptions']) == set(options(strict)), 'operation options differ from stage')
        require(op['syncResult']['revision'] == plan['revision'], 'wrong operation revision')
        require(status['sync']['revision'] == plan['revision'], 'wrong observed revision')
        conditions = status.get('conditions', [])
        if outcome == 'blocked':
            require(op['phase'] == 'Failed' and 'Shared resource found:' in op.get('message', ''), 'strict sync did not fail on sharing')
            require(any(c['type'] == 'SharedResourceWarning' for c in conditions), 'shared-resource warning missing')
            require(not [c for c in conditions if c['type'] != 'SharedResourceWarning'], 'unrelated failure masks shared-resource test')
        else:
            require(op['phase'] == 'Succeeded', 'sync did not succeed')
            require(status['sync']['status'] == 'Synced' and status['health']['status'] == 'Healthy', 'not Synced/Healthy')
            require(not conditions, 'Application condition remains')
            synced = {(o.get('kind'), o.get('namespace', ''), o.get('name')) for o in status['resources']}
            require(synced == {('Namespace', '', 'ot0-probe'), ('ConfigMap', 'ot0-probe', 'probe')}, 'wrong managed resource inventory')
    if baseline is not None:
        require(baseline['stage'] == 'A1', 'baseline must be A1')
        require(snapshot['clusterUID'] == baseline['clusterUID'], 'cluster identity changed')
        require(semantic(snapshot['project']) == semantic(baseline['project']), 'AppProject permissions/content changed')
        require(snapshot['project']['metadata']['uid'] == baseline['project']['metadata']['uid'], 'AppProject identity changed')
        originals = {key(o): o for o in baseline['objects']}
        for k, obj in resources.items():
            require(obj['metadata']['uid'] == originals[k]['metadata']['uid'], 'resource recreated: ' + k)
            require(semantic(obj) == semantic(originals[k]), 'resource content changed: ' + k)
    if previous is not None:
        require(previous['stage'] == PREVIOUS[stage], 'wrong predecessor evidence')
        if outcome in ('blocked', 'released'):
            require(snapshot['objects'] == previous['objects'], 'release/blocked sync mutated probe resources')
        if stage in ('A4', 'A5', 'A6-transfer', 'A6-restored'):
            require(apps[0]['metadata']['uid'] == previous['applications'][0]['metadata']['uid'], 'target Application replaced within window')
    return {'stage': stage, 'result': 'PASS', 'trackingOwner': tracking_owner,
            'resourceUIDs': {k: o['metadata']['uid'] for k, o in resources.items()},
            'semanticSHA256': {k: hashlib.sha256(encoded(semantic(o))).hexdigest() for k, o in resources.items()},
            'expectedOutcome': outcome}


def capture(stage, plan_dir, tools):
    plan = json.loads((plan_dir / 'plan.json').read_text())
    binding = json.loads((plan_dir / 'binding.json').read_text())
    kubeconfig = plan_dir / 'kubeconfig'
    require(binding['cluster'] == CLUSTER and plan['context'] == CONTEXT, 'wrong experiment binding')
    require(hashlib.sha256(kubeconfig.read_bytes()).hexdigest() == binding['kubeconfigSHA256'], 'kubeconfig changed')
    def get(*args):
        command = [str(tools / 'kubectl'), '--kubeconfig', str(kubeconfig), '--context', CONTEXT,
                   '--request-timeout=20s', *args]
        result = subprocess.run(command, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
        require(result.returncode == 0, 'read failed: ' + ' '.join(args))
        return json.loads(result.stdout)
    config = get('config', 'view', '--minify', '-o', 'json')
    require(urlparse(config['clusters'][0]['cluster']['server']).hostname == '127.0.0.1', 'API must be loopback')
    cluster_uid = get('get', 'namespace', 'kube-system', '-o', 'json')['metadata']['uid']
    require(cluster_uid == binding['clusterUID'], 'cluster UID differs from setup binding')
    cm = get('get', 'configmap', 'argocd-cm', '-n', 'argocd', '-o', 'json')
    return {'stage': stage, 'cluster': CLUSTER, 'clusterUID': cluster_uid, 'revision': plan['revision'],
        'trackingMethod': cm['data'].get('application.resourceTrackingMethod'),
        'objects': [get('get', 'namespace', 'ot0-probe', '-o', 'json', '--show-managed-fields'),
                    get('get', 'configmap', 'probe', '-n', 'ot0-probe', '-o', 'json', '--show-managed-fields')],
        'applications': get('get', 'applications.argoproj.io', '-n', 'argocd', '-o', 'json', '--show-managed-fields')['items'],
        'project': get('get', 'appproject', 'ot0-probe', '-n', 'argocd', '-o', 'json', '--show-managed-fields')}


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--stage', choices=STAGES, required=True)
    parser.add_argument('--plan-dir', type=Path, required=True)
    parser.add_argument('--tool-dir', type=Path, required=True)
    args = parser.parse_args()
    os.umask(0o077)
    directory = args.plan_dir.resolve()
    evidence = directory / 'evidence'
    evidence.mkdir(mode=0o700, exist_ok=True)
    snapshot = capture(args.stage, directory, args.tool_dir.resolve())
    # Preserve the actual observation even when an assertion fails. No overwrite,
    # retry, mutation, refresh, ownership patch, rollback or cleanup is performed.
    write(evidence / (args.stage + '.json'), snapshot)
    baseline = json.loads((evidence / 'A1.json').read_text()) if args.stage != 'A1' else None
    previous = json.loads((evidence / (PREVIOUS[args.stage] + '.json')).read_text()) if args.stage in PREVIOUS else None
    report = check(snapshot, json.loads((directory / 'plan.json').read_text()), baseline, previous)
    write(evidence / (args.stage + '-result.json'), report)
    print(json.dumps(report, sort_keys=True))
