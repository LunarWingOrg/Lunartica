package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDashboardEndpoints covers the workspace-dashboard rollups:
//   - daily token usage with and without project filter
//   - per-agent token usage with and without project filter
//   - per-agent run time
//
// Asserts that (1) tasks belonging to a project show up under the workspace
// view, (2) the project filter excludes tasks tied to issues without a
// matching project_id, and (3) run-time aggregation accumulates the
// completed_at − started_at delta correctly.
func TestDashboardEndpoints(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()

	var runtimeID, agentID string
	if err := testPool.QueryRow(ctx, `
		SELECT id FROM agent_runtime WHERE workspace_id = $1 LIMIT 1
	`, testWorkspaceID).Scan(&runtimeID); err != nil {
		t.Fatalf("fetch runtime: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		SELECT id FROM agent WHERE workspace_id = $1 LIMIT 1
	`, testWorkspaceID).Scan(&agentID); err != nil {
		t.Fatalf("fetch agent: %v", err)
	}

	// Two issues: one bound to a project, one not.
	var projectID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO project (workspace_id, title)
		VALUES ($1, 'dashboard test project')
		RETURNING id
	`, testWorkspaceID).Scan(&projectID); err != nil {
		t.Fatalf("create project: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM project WHERE id = $1`, projectID) })

	// issue.number is `UNIQUE (workspace_id, number)` (migration 020) and
	// defaults to 0. Two inserts into the same workspace would collide on the
	// default; allocate `MAX(number) + 1` per row to stay sequential and
	// avoid stepping on rows other tests have left behind in the shared
	// fixture workspace.
	mkIssue := func(withProject bool) string {
		var id string
		var pid any
		if withProject {
			pid = projectID
		}
		if err := testPool.QueryRow(ctx, `
			INSERT INTO issue (workspace_id, title, creator_id, creator_type, project_id, number)
			VALUES (
				$1, 'dashboard test', $2, 'member', $3,
				(SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1)
			)
			RETURNING id
		`, testWorkspaceID, testUserID, pid).Scan(&id); err != nil {
			t.Fatalf("insert issue: %v", err)
		}
		t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM issue WHERE id = $1`, id) })
		return id
	}
	projectIssueID := mkIssue(true)
	otherIssueID := mkIssue(false)

	now := time.Now().UTC()
	started := now.Add(-30 * time.Minute)
	completed := started.Add(10 * time.Minute) // 600s run

	mkTaskWithUsage := func(issueID string, status string, tokens int64) {
		var taskID string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO agent_task_queue (agent_id, issue_id, runtime_id, status, started_at, completed_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, now())
			RETURNING id
		`, agentID, issueID, runtimeID, status, started, completed).Scan(&taskID); err != nil {
			t.Fatalf("insert task: %v", err)
		}
		if _, err := testPool.Exec(ctx, `
			INSERT INTO task_usage (task_id, provider, model, input_tokens, output_tokens, created_at)
			VALUES ($1, 'claude', 'claude-3-5-sonnet', $2, 0, now())
		`, taskID, tokens); err != nil {
			t.Fatalf("insert task_usage: %v", err)
		}
		t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })
	}

	mkTaskWithUsage(projectIssueID, "completed", 1000)
	mkTaskWithUsage(otherIssueID, "completed", 500)

	type dailyRow struct {
		Date        string `json:"date"`
		Model       string `json:"model"`
		InputTokens int64  `json:"input_tokens"`
	}
	type byAgentRow struct {
		AgentID     string `json:"agent_id"`
		Model       string `json:"model"`
		InputTokens int64  `json:"input_tokens"`
	}
	type runtimeRow struct {
		AgentID      string `json:"agent_id"`
		TotalSeconds int64  `json:"total_seconds"`
		TaskCount    int32  `json:"task_count"`
	}

	// daily — workspace-wide
	{
		w := httptest.NewRecorder()
		testHandler.GetDashboardUsageDaily(w, newRequest("GET", "/api/dashboard/usage/daily?days=1", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("daily ws: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var rows []dailyRow
		_ = json.NewDecoder(w.Body).Decode(&rows)
		var total int64
		for _, r := range rows {
			if r.Model == "claude-3-5-sonnet" {
				total += r.InputTokens
			}
		}
		if total < 1500 {
			t.Errorf("daily ws: expected >=1500 tokens (1000+500), got %d", total)
		}
	}

	// daily — project-scoped
	{
		w := httptest.NewRecorder()
		testHandler.GetDashboardUsageDaily(w, newRequest("GET", "/api/dashboard/usage/daily?days=1&project_id="+projectID, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("daily project: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var rows []dailyRow
		_ = json.NewDecoder(w.Body).Decode(&rows)
		var total int64
		for _, r := range rows {
			if r.Model == "claude-3-5-sonnet" {
				total += r.InputTokens
			}
		}
		// Project filter must exclude the 500-token "other" issue. Token total
		// for this project must be >= 1000 (our task) and < 1500 (would only
		// reach 1500 if filter leaked).
		if total < 1000 {
			t.Errorf("daily project: expected >=1000 tokens, got %d", total)
		}
		if total >= 1500 {
			t.Errorf("daily project: filter leaked — expected <1500 tokens, got %d", total)
		}
	}

	// by-agent — project-scoped
	{
		w := httptest.NewRecorder()
		testHandler.GetDashboardUsageByAgent(w, newRequest("GET", "/api/dashboard/usage/by-agent?days=1&project_id="+projectID, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("by-agent project: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var rows []byAgentRow
		_ = json.NewDecoder(w.Body).Decode(&rows)
		found := false
		for _, r := range rows {
			if r.AgentID == agentID && r.InputTokens >= 1000 {
				found = true
			}
		}
		if !found {
			t.Errorf("by-agent project: expected agent %s with >=1000 tokens; got %v", agentID, rows)
		}
	}

	// agent-runtime — project-scoped
	{
		w := httptest.NewRecorder()
		testHandler.GetDashboardAgentRunTime(w, newRequest("GET", "/api/dashboard/agent-runtime?days=1&project_id="+projectID, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("agent-runtime: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var rows []runtimeRow
		_ = json.NewDecoder(w.Body).Decode(&rows)
		var seconds int64
		var tasks int32
		for _, r := range rows {
			if r.AgentID == agentID {
				seconds += r.TotalSeconds
				tasks += r.TaskCount
			}
		}
		if tasks < 1 {
			t.Errorf("agent-runtime: expected >=1 task for agent, got %d", tasks)
		}
		if seconds < 600 {
			t.Errorf("agent-runtime: expected >=600s (one 10-minute run), got %d", seconds)
		}
	}

	// agent-runtime — invalid project_id rejected
	{
		w := httptest.NewRecorder()
		testHandler.GetDashboardAgentRunTime(w, newRequest("GET", "/api/dashboard/agent-runtime?project_id=not-a-uuid", nil))
		if w.Code != http.StatusBadRequest {
			t.Errorf("agent-runtime: expected 400 for invalid uuid, got %d", w.Code)
		}
	}

	// Rollup path — run the dashboard window function, flip the feature
	// flag, and verify daily + by-agent reads come back with the same
	// project-filtered totals. The raw path above already passed, so this
	// validates that the rollup table mirrors the raw aggregation
	// (modulo project_id snapshot semantics, which match here since
	// nothing has changed since the rows were created).
	{
		// rollup the full window in one shot; same idempotent primitive
		// the cron path uses.
		if _, err := testPool.Exec(ctx, `
			SELECT rollup_task_usage_dashboard_daily_window('1970-01-01'::timestamptz, now() + interval '1 hour')
		`); err != nil {
			t.Fatalf("rollup window: %v", err)
		}
		origRollup := testHandler.cfg.UseDailyRollupForDashboard
		testHandler.cfg.UseDailyRollupForDashboard = true
		t.Cleanup(func() { testHandler.cfg.UseDailyRollupForDashboard = origRollup })

		// daily — project-scoped through rollup
		w := httptest.NewRecorder()
		testHandler.GetDashboardUsageDaily(w, newRequest("GET", "/api/dashboard/usage/daily?days=1&project_id="+projectID, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("rollup daily: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var dRows []dailyRow
		_ = json.NewDecoder(w.Body).Decode(&dRows)
		var dTotal int64
		for _, r := range dRows {
			if r.Model == "claude-3-5-sonnet" {
				dTotal += r.InputTokens
			}
		}
		if dTotal < 1000 {
			t.Errorf("rollup daily project: expected >=1000 tokens, got %d", dTotal)
		}
		if dTotal >= 1500 {
			t.Errorf("rollup daily project: filter leaked — expected <1500, got %d", dTotal)
		}

		// by-agent — workspace-wide through rollup
		w = httptest.NewRecorder()
		testHandler.GetDashboardUsageByAgent(w, newRequest("GET", "/api/dashboard/usage/by-agent?days=1", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("rollup by-agent: expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var aRows []byAgentRow
		_ = json.NewDecoder(w.Body).Decode(&aRows)
		var aTotal int64
		for _, r := range aRows {
			if r.AgentID == agentID && r.Model == "claude-3-5-sonnet" {
				aTotal += r.InputTokens
			}
		}
		if aTotal < 1500 {
			t.Errorf("rollup by-agent: expected >=1500 tokens across workspace, got %d", aTotal)
		}
	}
}

// TestDashboardRollupReattributesOnProjectChange verifies the trigger that
// fires on `UPDATE issue SET project_id` enqueues both old + new project
// buckets so the next rollup tick re-attributes the affected tokens.
// Uses the rollup window function directly to drain the dirty queue,
// then asserts the rollup table reflects the new project_id.
func TestDashboardRollupReattributesOnProjectChange(t *testing.T) {
	if testHandler == nil {
		t.Skip("database not available")
	}
	ctx := context.Background()

	var runtimeID, agentID string
	if err := testPool.QueryRow(ctx, `SELECT id FROM agent_runtime WHERE workspace_id = $1 LIMIT 1`, testWorkspaceID).Scan(&runtimeID); err != nil {
		t.Fatalf("fetch runtime: %v", err)
	}
	if err := testPool.QueryRow(ctx, `SELECT id FROM agent WHERE workspace_id = $1 LIMIT 1`, testWorkspaceID).Scan(&agentID); err != nil {
		t.Fatalf("fetch agent: %v", err)
	}

	mkProject := func(name string) string {
		var id string
		if err := testPool.QueryRow(ctx, `
			INSERT INTO project (workspace_id, title) VALUES ($1, $2) RETURNING id
		`, testWorkspaceID, name).Scan(&id); err != nil {
			t.Fatalf("create project: %v", err)
		}
		t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM project WHERE id = $1`, id) })
		return id
	}
	projectA := mkProject("dashboard reattr A")
	projectB := mkProject("dashboard reattr B")

	var issueID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO issue (workspace_id, title, creator_id, creator_type, project_id, number)
		VALUES ($1, 'reattr issue', $2, 'member', $3,
		        (SELECT COALESCE(MAX(number), 0) + 1 FROM issue WHERE workspace_id = $1))
		RETURNING id
	`, testWorkspaceID, testUserID, projectA).Scan(&issueID); err != nil {
		t.Fatalf("create issue: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM issue WHERE id = $1`, issueID) })

	var taskID string
	if err := testPool.QueryRow(ctx, `
		INSERT INTO agent_task_queue (agent_id, issue_id, runtime_id, status, created_at)
		VALUES ($1, $2, $3, 'completed', now()) RETURNING id
	`, agentID, issueID, runtimeID).Scan(&taskID); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	t.Cleanup(func() { testPool.Exec(ctx, `DELETE FROM agent_task_queue WHERE id = $1`, taskID) })

	if _, err := testPool.Exec(ctx, `
		INSERT INTO task_usage (task_id, provider, model, input_tokens, output_tokens, created_at)
		VALUES ($1, 'claude', 'claude-3-5-sonnet', 7777, 0, now())
	`, taskID); err != nil {
		t.Fatalf("insert task_usage: %v", err)
	}

	// First rollup pass: tokens attributed to project A.
	if _, err := testPool.Exec(ctx, `
		SELECT rollup_task_usage_dashboard_daily_window('1970-01-01'::timestamptz, now() + interval '1 hour')
	`); err != nil {
		t.Fatalf("rollup A: %v", err)
	}
	var aTokens int64
	if err := testPool.QueryRow(ctx, `
		SELECT COALESCE(SUM(input_tokens), 0) FROM task_usage_dashboard_daily
		WHERE workspace_id = $1 AND project_id = $2 AND agent_id = $3
	`, testWorkspaceID, projectA, agentID).Scan(&aTokens); err != nil {
		t.Fatalf("read A rollup: %v", err)
	}
	if aTokens < 7777 {
		t.Fatalf("project A: expected >=7777 tokens after first rollup, got %d", aTokens)
	}

	// Move the issue to project B. Trigger enqueues both A and B buckets.
	if _, err := testPool.Exec(ctx, `UPDATE issue SET project_id = $1 WHERE id = $2`, projectB, issueID); err != nil {
		t.Fatalf("reassign project: %v", err)
	}
	// Second rollup pass: A bucket drops to zero (deleted_empty), B
	// bucket gets the tokens.
	if _, err := testPool.Exec(ctx, `
		SELECT rollup_task_usage_dashboard_daily_window('1970-01-01'::timestamptz, now() + interval '1 hour')
	`); err != nil {
		t.Fatalf("rollup B: %v", err)
	}

	var bTokens, aTokensAfter int64
	if err := testPool.QueryRow(ctx, `
		SELECT COALESCE(SUM(input_tokens), 0) FROM task_usage_dashboard_daily
		WHERE workspace_id = $1 AND project_id = $2 AND agent_id = $3
	`, testWorkspaceID, projectB, agentID).Scan(&bTokens); err != nil {
		t.Fatalf("read B rollup: %v", err)
	}
	if err := testPool.QueryRow(ctx, `
		SELECT COALESCE(SUM(input_tokens), 0) FROM task_usage_dashboard_daily
		WHERE workspace_id = $1 AND project_id = $2 AND agent_id = $3
	`, testWorkspaceID, projectA, agentID).Scan(&aTokensAfter); err != nil {
		t.Fatalf("read A rollup after move: %v", err)
	}
	if bTokens < 7777 {
		t.Errorf("project B: expected >=7777 tokens after reassign + rollup, got %d", bTokens)
	}
	if aTokensAfter != 0 {
		t.Errorf("project A: expected 0 tokens after reassign + rollup, got %d", aTokensAfter)
	}
}
