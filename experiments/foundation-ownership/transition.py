#!/usr/bin/env python3
"""Pure OT-1 Application mode patch builder; never calls Kubernetes.

The future reviewed executor must GET a new object immediately before patching,
verify the phase binding and append the response/exit code to the attempt log.
This helper is not an executor or an authorization mechanism.
"""
import copy
from contract import require

REPOSITORY = 'https://github.com/snkio027/atlas-refactor.git'
REVISION = 'codex/ot1-desired-state'
OWNERS = {'capability-foundation', 'secrets-foundation', 'observability-foundation', 'storage-foundation'}


def options(strict):
    return ['ServerSideApply=true', 'FailOnSharedResource=' + str(strict).lower()]


def mode_patch(current, expected, desired_sha, previous_stage, strict):
    require(isinstance(strict, bool), 'strict must be boolean')
    require(len(desired_sha) == 40 and all(c in '0123456789abcdef' for c in desired_sha), 'full SHA required')
    require(expected['kind'] == 'Application' and expected['apiVersion'] == 'argoproj.io/v1alpha1', 'wrong expected kind')
    require(current['kind'] == expected['kind'] and current['apiVersion'] == expected['apiVersion'], 'wrong live kind')
    meta = current['metadata']
    require(meta['name'] in OWNERS and meta['name'] == expected['metadata']['name'] and
            meta['namespace'] == expected['metadata']['namespace'] == 'argocd', 'wrong Application target')
    require(meta.get('uid') and meta.get('resourceVersion'), 'missing concurrency guards')
    require(not meta.get('finalizers') and not meta.get('deletionTimestamp'), 'Application deletion boundary')
    require(not current.get('operation'), 'Application operation active')
    require(current['spec'] == expected['spec'], 'full expected spec mismatch')
    spec = current['spec']
    require(spec['project'] == 'platform-project', 'wrong canonical project')
    require(spec['source']['repoURL'] == REPOSITORY and spec['source']['targetRevision'] == REVISION, 'not isolated OT-1 source')
    require(not spec['syncPolicy'].get('automated'), 'ceremony Application must be detached and manually synced')
    require(spec['syncPolicy']['syncOptions'] == options(not strict), 'unexpected predecessor mode')
    require(current['status']['sync']['revision'] == desired_sha, 'wrong observed revision')
    op = current['status']['operationState']
    require(op.get('finishedAt') and op['phase'] == ('Succeeded' if strict else 'Failed'), 'wrong predecessor result')
    require(op['syncResult']['revision'] == desired_sha, 'wrong operation result revision')
    request = op['operation']
    require(request.get('info') == [{'name': 'ot1-stage', 'value': previous_stage}], 'stale operation')
    require(request['sync']['revision'] == desired_sha and not request['sync'].get('prune'), 'unexpected sync request')
    require(request['sync']['syncOptions'] == options(not strict), 'wrong operation options')
    conditions = current['status'].get('conditions', [])
    if strict:
        require(not conditions and current['status']['sync']['status'] == 'Synced' and
                current['status']['health']['status'] == 'Healthy', 'window did not converge')
    else:
        require('Shared resource found:' in op.get('message', ''), 'not an expected shared-resource refusal')
        require(conditions and all(c['type'] == 'SharedResourceWarning' for c in conditions), 'unexpected failure condition')
    return [
        {'op': 'test', 'path': '/metadata/uid', 'value': meta['uid']},
        {'op': 'test', 'path': '/metadata/resourceVersion', 'value': meta['resourceVersion']},
        {'op': 'test', 'path': '/spec', 'value': copy.deepcopy(spec)},
        {'op': 'replace', 'path': '/spec/syncPolicy/syncOptions', 'value': options(strict)},
    ]
