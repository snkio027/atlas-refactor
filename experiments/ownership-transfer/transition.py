#!/usr/bin/env python3
"""Render a guarded Application-only mode transition; no Kubernetes calls."""
import argparse
import json
from pathlib import Path

from prepare import STAGES, application, options
from observe import PREVIOUS, require


def transition_patch(current, target, stage, revision):
    require(stage in ('A4', 'A5', 'A6-transfer', 'A6-restored'), 'not a mode-transition stage')
    owner, strict, _ = STAGES[stage]
    require(target == application(owner, strict, revision), 'target differs from reviewed stage')
    require(current['kind'] == 'Application' and current['apiVersion'] == 'argoproj.io/v1alpha1', 'not an Application')
    m = current['metadata']
    require(m['name'] == owner and m['namespace'] == 'argocd' and m.get('uid') and m.get('resourceVersion'), 'wrong Application identity')
    require(not m.get('finalizers') and not m.get('deletionTimestamp'), 'Application deletion/finalizer boundary')
    require(not current.get('operation'), 'operation active')
    op = current['status']['operationState']
    require(op['phase'] in ('Succeeded', 'Failed') and op.get('finishedAt'), 'operation not terminal')
    require(op['operation'].get('info') == [{'name': 'ot0-stage', 'value': PREVIOUS[stage]}], 'wrong predecessor operation')
    require(current['spec'] == application(owner, not strict, revision)['spec'], 'unexpected source Application spec')
    # The resourceVersion test covers concurrent spec/status/operation changes.
    # This changes only the reviewed Application option. Probe objects and their
    # tracking metadata remain exclusively under Argo SSA.
    return [
        {'op': 'test', 'path': '/metadata/uid', 'value': m['uid']},
        {'op': 'test', 'path': '/metadata/resourceVersion', 'value': m['resourceVersion']},
        {'op': 'test', 'path': '/spec', 'value': current['spec']},
        {'op': 'replace', 'path': '/spec/syncPolicy/syncOptions', 'value': options(strict)},
    ]


if __name__ == '__main__':
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--stage', required=True)
    p.add_argument('--current', type=Path, required=True)
    p.add_argument('--plan-dir', type=Path, required=True)
    args = p.parse_args()
    plan = json.loads((args.plan_dir / 'plan.json').read_text())
    target = json.loads((args.plan_dir / (args.stage + '-app.json')).read_text())
    current = json.loads(args.current.read_text())
    print(json.dumps(transition_patch(current, target, args.stage, plan['revision']), indent=2))
