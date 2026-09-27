"""Negative tests for proof acceptance, not simulated runtime proof."""
import copy
import unittest

from observe import check, semantic, TRACKING, STAGES, OWNERS, PREVIOUS
from prepare import application, options, CLUSTER, argocd_config

SHA = 'a' * 40
PLAN = {'revision': SHA}


def snapshot(stage):
    owner, strict, outcome = STAGES[stage]
    tracking_owner = OWNERS[stage]
    objects = []
    for kind, name, ns in [('Namespace', 'ot0-probe', ''), ('ConfigMap', 'probe', 'ot0-probe')]:
        obj = {'apiVersion': 'v1', 'kind': kind, 'metadata': {
            'name': name, 'uid': kind + '-uid', 'resourceVersion': '1',
            'annotations': {TRACKING: tracking_owner + ':/' + kind + ':ot0-probe/' + name},
            'managedFields': [{'manager': 'argocd-controller', 'operation': 'Apply'}],
        }}
        if ns:
            obj['metadata']['namespace'] = ns
            obj['data'] = {'contract': 'unchanged'}
        objects.append(obj)
    apps = []
    if owner:
        app = application(owner, strict, SHA)
        app['metadata']['uid'] = owner + '-uid'
        app['status'] = {'sync': {'revision': SHA, 'status': 'Synced'}, 'health': {'status': 'Healthy'},
            'resources': [{'kind': 'Namespace', 'name': 'ot0-probe'}, {'kind': 'ConfigMap', 'name': 'probe', 'namespace': 'ot0-probe'}],
            'operationState': {'finishedAt': '2026-09-27T00:00:00Z', 'phase': 'Failed' if outcome == 'blocked' else 'Succeeded',
                'message': 'Shared resource found: stale owner' if outcome == 'blocked' else 'success',
                'syncResult': {'revision': SHA},
                'operation': {'info': [{'name': 'ot0-stage', 'value': stage}],
                              'sync': {'revision': SHA, 'prune': False, 'syncOptions': options(strict)}}}}
        if outcome == 'blocked':
            app['status']['conditions'] = [{'type': 'SharedResourceWarning'}]
        apps.append(app)
    return {'stage': stage, 'cluster': CLUSTER, 'clusterUID': 'cluster-uid', 'revision': SHA,
        'trackingMethod': 'annotation', 'objects': objects, 'applications': apps,
        'project': {'kind': 'AppProject', 'metadata': {'name': 'ot0-probe', 'uid': 'project-uid'}, 'spec': {'unchanged': True}}}


class EvidenceTests(unittest.TestCase):
    def test_settings_configmap_is_visible_to_argocd_informer(self):
        # Upstream v3.5.1 SettingsManager.initialize applies this selector;
        # the first live attempt failed before A1 when the label was omitted.
        cm = argocd_config()
        selected = [o for o in [cm] if o['metadata'].get('labels', {}).get(
            'app.kubernetes.io/part-of') == 'argocd']
        self.assertEqual([o['metadata']['name'] for o in selected], ['argocd-cm'])
        self.assertEqual(selected[0]['data']['application.resourceTrackingMethod'], 'annotation')

    def test_accepts_explicit_forward_and_reverse_sequence(self):
        baseline = snapshot('A1')
        previous = None
        for stage in STAGES:
            current = snapshot(stage)
            check(current, PLAN, baseline if stage != 'A1' else None, previous)
            previous = current

    def test_rejects_false_positive_proof(self):
        def altered(mutate, stage='A4'):
            value = snapshot(stage)
            mutate(value)
            with self.assertRaises((ValueError, KeyError)):
                check(value, PLAN, snapshot('A1'), snapshot(PREVIOUS[stage]))
        altered(lambda s: s['objects'][0]['metadata'].update(uid='replacement'))
        altered(lambda s: s['objects'][1]['data'].update(contract='changed'))
        altered(lambda s: s['objects'][0]['metadata']['annotations'].update({TRACKING: 'wrong'}))
        altered(lambda s: s['objects'][1]['metadata'].update(annotations={TRACKING: s['objects'][1]['metadata']['annotations'][TRACKING], 'unrelated': 'changed'}))
        altered(lambda s: s['objects'][0]['metadata'].update(managedFields=[]))
        altered(lambda s: s['applications'].append(application('owner-a', True, SHA)))
        altered(lambda s: s['applications'][0]['metadata'].update(finalizers=['resources-finalizer.argocd.argoproj.io']))
        altered(lambda s: s.update(clusterUID='other-cluster'))
        altered(lambda s: s.update(trackingMethod='label'))
        altered(lambda s: s['project']['spec'].update(unchanged=False))
        altered(lambda s: s['applications'][0]['status']['operationState']['operation'].update(info=[{'name': 'ot0-stage', 'value': 'A3'}]))
        altered(lambda s: s['applications'][0]['status']['operationState']['syncResult'].update(revision='b'*40))
        altered(lambda s: s['applications'][0]['spec']['syncPolicy'].update(syncOptions=options(False)), 'A5')
        altered(lambda s: s['applications'][0]['status']['operationState'].update(phase='Succeeded'), 'A3')
        altered(lambda s: s['applications'][0]['status']['operationState'].update(message='unrelated network failure'), 'A3')
        altered(lambda s: s['objects'][1]['metadata'].update(resourceVersion='2'), 'A3')
        altered(lambda s: s['objects'][1]['metadata'].update(resourceVersion='2'), 'A2')
        altered(lambda s: s['applications'][0]['status']['operationState'].update(phase='Succeeded'), 'A6-strict')

    def test_only_tracking_and_api_lifecycle_fields_are_normalized(self):
        original = snapshot('A1')['objects'][1]
        changed = copy.deepcopy(original)
        changed['metadata']['annotations'][TRACKING] = 'other-owner'
        changed['metadata']['resourceVersion'] = '2'
        changed['metadata']['managedFields'] = [{'different': True}]
        self.assertEqual(semantic(original), semantic(changed))
        changed['metadata']['labels'] = {'unexpected': 'mutation'}
        self.assertNotEqual(semantic(original), semantic(changed))


if __name__ == '__main__':
    unittest.main()
