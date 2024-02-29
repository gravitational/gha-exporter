# gha-exporter
GitHub Actions metrics exporter for Prometheus

## Usage

```
Usage: gha-exporter --app-id=INT-64 --app-key=STRING --owner=STRING --repos=REPOS,...

Flags:
  -h, --help                 Show context-sensitive help.
      --app-id=INT-64        GitHub App ID of application to authenticate as ($GHA_APP_ID)
      --app-key=STRING       Private key of GitHub App for the App ID ($GHA_APP_KEY)
      --owner=STRING         GitHub owner of repositories to monitor ($GHA_OWNER)
      --repos=REPOS,...      Repositories to monitor (must be owned by --owner)
      --sleep=1m             Sleep time (duration string)between polls of GitHub Actions
      --initial-window=2h    Initial time to look back for runs
      --backfill             Backfill completed runs from initial window
```

gha-exporter requires a GitHub app be installed in the organization that owns
the repositories and that app has the `actions: read` scope.

The `--initial-window` flag tells `gha-exporter` how far back to look for
incomplete workflow runs. Any incomplete runs older than that will not be
collected for metrics. When those runs complete, `gha-exporter` will include the
run times of the workflow, jobs and steps in the metrics it publishes. The
initial window should be set to large enough to capture the longest running
workflow that you have.

If the `--backfill` flag is provided, any completed runs within the initial
window are collected for metrics. By default `gha-exporter` only collects
metrics for workflows that completes after it starts. If `gha-exporter` is down
for a period, then any workflow runs that complete during that period will not
get collected. `--backfill` allows you to catch up if the period might be longer
than you are prepared to accept for gaps in your data.

