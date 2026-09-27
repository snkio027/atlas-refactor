"""Read-only API probe predicates for the disposable OT-1 platform."""


def monitoring_targets_ready(targets):
    """Wait for required discovery as well as health of discovered targets."""
    if not targets or any(target.get("health") != "up" for target in targets):
        return False
    storage = [
        target for target in targets
        if target.get("labels", {}).get("service") == "seaweedfs"
        and target.get("labels", {}).get("namespace") == "atlas-storage"
    ]
    return len(storage) == 1


if __name__ == "__main__":
    import json
    import sys

    try:
        targets = json.load(sys.stdin)["data"]["activeTargets"]
        ready = monitoring_targets_ready(targets)
    except (KeyError, TypeError, AttributeError, ValueError):
        print("invalid Prometheus targets response", file=sys.stderr)
        sys.exit(2)
    print("ready" if ready else "waiting for required targets")
    sys.exit(0 if ready else 1)
