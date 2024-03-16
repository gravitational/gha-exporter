# Metrics for GHA exporter

## `gha_step_run_time_seconds`

A `counter` of the run time of steps in a job in a workflow.

### Labels
* repo: github `owner/repo` containing the workflow
* ref: depending on event type:
  * If a currently open `pull_request` or related events, pull request target branch
  * If a `merge_group` event, the merge group target branch
  * All others (i.e. push event), the head branch
* event: the name of the triggering event (such as `push` or `pull_request`)
* workflow: workflow path without leading `.github/workflows` and trailing `.ya?ml`
* job: job name
* step: step name
TODO:
* `runner_type`

Note: it is currently hard to get the workflow name of a reused workflow - i.e.
one invoked via `workflow_call` with a `uses:` clause - we just see the job name
for each of the jobs in the called workflow as "Caller name / Job name", with
more levels when there are more workflows in the chain. It will suffice
initially to use this long string as the job name and we'll come back later to
see if we can find a way to correlate this job name "path" with the workflow
file(s).

Note: The ref field is not intended to be exhaustive, but to collect the
"class" of ref the workflows ran against. We'd like to see metrics for master
and the individual release branches split out, and the rest is either `pr` for
workflows that run on PRs, in the merge_queue and anything else (other). We may
need to do some munging for tag builds to extract the major version for the
branch.


## `gha_job_run_time_seconds`

A `counter` of the run time of jobs in a workflow. This is a sum of the
`gha_step_run_time_seconds` for a particular job instance (job ID in GHA).

### Labels

Same as `gha_step_run_time_seconds` but without the `step` label.


## `gha_workflow_run_time_seconds`

A `counter` of the run time of a workflow. This is a sum of the
`gha_job_run_time_seconds` for a particular job instance (job ID in GHA).

### Labels

Same as `gha_job_run_time_seconds` but without the `job` label.


## `gha_step_run_count`

A `counter` of the number of runs of a step by conclusion.

### Labels

Same as `gha_step_run_time_seconds` with the addition of:
* conclusion: The github conclusion of the step: `success`, `skipped`,
  `cancelled`, `failure`.


## `gha_job_run_count`

A `counter` of the number of runs of a job by conclusion.

### Labels

Same as `gha_step_run_count` but without the `step` label.


## `gha_workflow_run_count`

A `counter` of the number of runs of a workflow by conclusion.

### Labels

Same as `gha_job_run_count` but without the `job` label.

