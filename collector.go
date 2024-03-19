package main

import (
	"context"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v60/github"
	"github.com/gravitational/trace"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	workflowRunRunnerTimeVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_workflow_run_runner_seconds",
		},
		[]string{"repo", "ref", "event_type", "workflow"},
	)
	workflowRunElapsedTimeVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_workflow_run_elapsed_seconds",
		},
		[]string{"repo", "ref", "event_type", "workflow"},
	)
	jobRunTimeVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_job_run_time_seconds",
		},
		[]string{"repo", "ref", "event_type", "workflow", "job"},
	)
	stepRunTimeVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_step_run_time_seconds",
		},
		[]string{"repo", "ref", "event_type", "workflow", "job", "step"},
	)
	workflowRunCountVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_workflow_run_count",
		},
		[]string{"repo", "ref", "event_type", "workflow", "conclusion"},
	)
	jobRunCountVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_job_run_count",
		},
		[]string{"repo", "ref", "event_type", "workflow", "job", "conclusion"},
	)
	stepRunCountVec = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gha_step_run_count",
		},
		[]string{"repo", "ref", "event_type", "workflow", "job", "step", "conclusion"},
	)
)

type Collector struct {
	cfg    *CLI
	client *github.Client
	// seenRuns tracks which completed workflows runs we've seen so we don't count
	// them again.
	seenRuns map[int64]bool
	// oldestIncomplete tracks per-repo the earliest start time of a workflow run that we've
	// seen that was not completed. This becomes the cutoff for querying github
	// each collect loop.
	oldestIncomplete map[string]time.Time
}

func NewCollector(cfg *CLI) *Collector {
	prometheus.MustRegister(workflowRunRunnerTimeVec)
	prometheus.MustRegister(workflowRunElapsedTimeVec)
	prometheus.MustRegister(jobRunTimeVec)
	prometheus.MustRegister(stepRunTimeVec)
	prometheus.MustRegister(workflowRunCountVec)
	prometheus.MustRegister(jobRunCountVec)
	prometheus.MustRegister(stepRunCountVec)

	return &Collector{
		cfg:              cfg,
		seenRuns:         make(map[int64]bool),
		oldestIncomplete: make(map[string]time.Time),
	}
}

func (c *Collector) Run(ctx context.Context) {
	log.Print("Collector started")
	defer func() { log.Print("Collector finished: ", ctx.Err()) }()

	for ; ctx.Err() == nil; time.Sleep(c.cfg.Sleep) {
		for _, repo := range c.cfg.Repos {
			if err := c.collectRepo(ctx, repo); err != nil {
				log.Print(err)
			}
		}
	}
}

func (c *Collector) collectRepo(ctx context.Context, repo string) error {
	log.Printf("Started collecting repo: %s", repo)
	defer func() { log.Printf("Finished collecting repo: %s", repo) }()

	// Recreate client on demand after errors and on startup
	if c.client == nil {
		client, err := newGHClient(c.cfg.Owner, c.cfg.AppID, []byte(c.cfg.AppKey))
		if err != nil {
			return trace.Wrap(err, "collection failed for repo: %q", repo)
		}
		c.client = client
	}

	ignoreCompleted := false
	cutoff, ok := c.oldestIncomplete[repo]
	if !ok {
		cutoff = time.Now().UTC().Add(-c.cfg.InitialWindow)
		// This is the first time collecting from this repo. We normally ignore
		// completed runs and only collect run time from runs that complete
		// after we have started, but if the `--backfill` flag is given, then
		// we collect run times from all completed runs in the initial window.
		ignoreCompleted = !c.cfg.Backfill
	}

	created := cutoff.Format("created:>=2006-01-02T15:04:05+00:00")
	opts := &github.ListWorkflowRunsOptions{
		Created:     created,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	actions := c.client.Actions
	newCutoff := time.Now().UTC()
	for {
		runs, response, err := actions.ListRepositoryWorkflowRuns(ctx, c.cfg.Owner, repo, opts)
		if err != nil {
			return trace.Wrap(err, "failed to list workflow runs")
		}

		for _, run := range runs.WorkflowRuns {
			newCutoff, err = c.collectRun(ctx, repo, run, newCutoff, ignoreCompleted)
			if err != nil {
				return trace.Wrap(err, "failed to collect workflow runs")
			}
		}

		opts.Page = response.NextPage
		if opts.Page == 0 {
			break
		}
	}
	c.oldestIncomplete[repo] = newCutoff
	return nil
}

func (c *Collector) collectRun(ctx context.Context, repo string, run *github.WorkflowRun, cutoff time.Time, ignoreCompleted bool) (time.Time, error) {
	var err error
	switch {
	case run.GetStatus() != "completed":
		switch {
		case run.CreatedAt == nil:
			// Ignore runs without a created timestamp
		case run.CreatedAt.Before(cutoff):
			cutoff = run.CreatedAt.Time
			log.Printf("open: %s/%d: %s (new cutoff: %v)",
				repo, run.GetID(), run.GetName(), cutoff)
		default:
			log.Printf("open: %s/%d: %s", repo, run.GetID(), run.GetName())
		}
	case c.seenRuns[run.GetID()]:
		log.Printf("seen: %s/%d: %s", repo, run.GetID(), run.GetName())
	default:
		if !ignoreCompleted {
			err = c.collectJobs(ctx, repo, run)
		}
		if err == nil {
			c.seenRuns[run.GetID()] = true
			log.Printf("done: %s/%d: %s\n", repo, run.GetID(), run.GetName())
		}
	}

	return cutoff, err
}

func (c *Collector) collectJobs(ctx context.Context, repo string, run *github.WorkflowRun) error {
	opts := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	actions := c.client.Actions
	for {
		jobs, response, err := actions.ListWorkflowJobs(ctx, c.cfg.Owner, repo, run.GetID(), opts)
		if err != nil {
			return trace.Wrap(err, "failed to list workflow run jobs")
		}

		countJobs(run, jobs.Jobs)

		opts.Page = response.NextPage
		if opts.Page == 0 {
			break
		}
	}
	return nil
}

func countJobs(run *github.WorkflowRun, jobs []*github.WorkflowJob) {
	// The tool doesn't currently account for more than one run attempt.
	// This brings a multitude of issues that are near-impossible to
	// account for due to GH's API design.
	if run.GetRunAttempt() > 1 {
		return
	}

	workflowName := path.Base(run.GetPath())
	workflowName = strings.TrimSuffix(workflowName, path.Ext(workflowName))
	repo := run.GetRepository().GetName()
	ref := makeRef(run)
	eventType := run.GetEvent()

	var workflowRunTime time.Duration
	for _, job := range jobs {
		var jobRunTime time.Duration
		for _, step := range job.Steps {
			stepRunCountVec.WithLabelValues(
				repo, ref, eventType, workflowName, job.GetName(), step.GetName(), step.GetConclusion(),
			).Add(1)

			if step.GetConclusion() != "success" {
				continue
			}
			if step.StartedAt == nil || step.CompletedAt == nil {
				continue
			}
			stepRunTime := step.CompletedAt.Time.Sub(step.StartedAt.Time)
			stepRunTimeVec.WithLabelValues(
				repo, ref, eventType, workflowName, job.GetName(), step.GetName(),
			).Add(stepRunTime.Seconds())
			jobRunTime += stepRunTime
		}
		jobRunCountVec.WithLabelValues(
			repo, ref, eventType, workflowName, job.GetName(), job.GetConclusion(),
		).Add(1)
		if job.GetConclusion() != "success" {
			continue
		}
		jobRunTimeVec.WithLabelValues(
			repo, ref, eventType, workflowName, job.GetName(),
		).Add(jobRunTime.Seconds())

		workflowRunTime += jobRunTime
	}

	workflowRunCountVec.WithLabelValues(repo, ref, eventType, workflowName, run.GetConclusion()).Add(1)

	if run.GetConclusion() != "success" {
		return
	}

	workflowRunRunnerTimeVec.WithLabelValues(repo, ref, eventType, workflowName).Add(workflowRunTime.Seconds())
	workflowRunElapsedTimeVec.WithLabelValues(repo, ref, eventType, workflowName).Add(run.GetUpdatedAt().Sub(run.GetCreatedAt().Time).Seconds())
}

func makeRef(run *github.WorkflowRun) string {
	eventName := run.GetEvent()
	if strings.HasPrefix(eventName, "pull_request") {
		// Attempt to tie the workflow to a PR
		headSha := run.GetHeadSHA()
		headBranch := run.GetHeadBranch()

		if headSha == "" && headBranch == "" {
			return "error-missing-head-ref"
		}

		for _, pr := range run.PullRequests {
			prHeadBranch := pr.GetHead()

			if prHeadBranch.GetSHA() == headSha ||
				prHeadBranch.GetRef() == headBranch {
				return pr.GetBase().GetRef()
			}
		}

		return headBranch
	}

	if strings.HasPrefix(eventName, "merge_group") {
		headBranch := run.GetHeadBranch()
		if headBranch == "" {
			return "error-head-branch-is-nil"
		}

		mergeBranch := strings.TrimPrefix(headBranch, "gh-readonly-queue/")
		mergeBranch = strings.SplitN(mergeBranch, "/pr-", 2)[0]
		return mergeBranch
	}

	headBranch := run.GetHeadBranch()
	if headBranch == "" {
		return "error-head-branch-is-nil"
	}

	return headBranch
}

func newGHClient(owner string, appID int64, appKey []byte) (*github.Client, error) {
	// Create a retryable http client / roundtripper, but turn off logging
	// for now as it is a bit noisy. We should use a logger that implements
	// retryablehttp.LevelledLogger so we can turn on/off debug logging as needed.
	rtclient := retryablehttp.NewClient()
	rtclient.Logger = nil // too noisy right now
	rt := &retryablehttp.RoundTripper{Client: rtclient}

	// Create a temporary github client that can list the app installations
	// so we can get the installation ID for the proper github client.
	tmpTransport, err := ghinstallation.NewAppsTransport(rt, appID, appKey)
	if err != nil {
		return nil, err
	}
	gh := github.NewClient(&http.Client{Transport: tmpTransport})

	var instID int64
	listOptions := github.ListOptions{PerPage: 100}
findInst:
	for {
		installations, response, err := gh.Apps.ListInstallations(context.Background(), &listOptions)
		if err != nil {
			return nil, trace.Wrap(err, "Failed to list installations")
		}

		for _, inst := range installations {
			if inst.GetAccount().GetLogin() == owner {
				instID = inst.GetID()
				break findInst
			}
		}

		if response.NextPage == 0 {
			return nil, trace.NotFound("No such installation found")
		}

		listOptions.Page = response.NextPage
	}

	transport, err := ghinstallation.New(rt, appID, instID, appKey)
	if err != nil {
		return nil, trace.Wrap(err, "Failed creating authenticated transport")
	}

	gh = github.NewClient(&http.Client{Transport: transport})

	return gh, nil
}
