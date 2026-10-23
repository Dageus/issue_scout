# Issue Scout

A personal tool that finds newly trending GitHub repositories in my languages and surfaces open issues that fit my skills, fast enough to claim them before others do.

This is also a **learning project for agentic development**. I am building v1 myself. Your job is to help me understand and improve it, not to take it over.

## How to work with me

- I'm experienced in **Rust and Go**, newer to agentic/LLM development. Explain agentic concepts when they come up; don't explain basic Rust/Go.
- Prefer guiding over rewriting: review my code, point out problems, suggest the next step. Write larger pieces of code only when I ask.
- When there's a design choice, give me the tradeoff in a sentence or two and a recommendation, then let me decide.
- Keep the project small. Push back if I'm adding infrastructure the current version doesn't need.

## Architecture

The project grows in three stages. Know which stage we're in before suggesting changes.

1. **v1 – Workflow.** Code controls every step; the model only judges.
   cron → fetch trending repos → fetch new issues → cheap filters in code → LLM scores survivors → alert on high scores.
2. **v2 – Agent.** Give the model tools (`search_repos`, `list_issues`, `get_issue`, `read_profile`) and a goal ("find up to 5 unclaimed issues that fit me"). The model decides which calls to make; my code just runs the tool loop.
3. **v3 – Verified agent.** Add an independent check (like Claude Code's `/goal` evaluator): confirm each pick is still unassigned, has no linked PR, and matches the profile; if not, send the agent back.

## Division of responsibility

| Piece | Owner | Notes |
|---|---|---|
| Logic | Code | Scheduling, GitHub API, filtering, dedup, state, alerts. Deterministic and testable. |
| Context | Code assembles, model reads | Profile text + a batch of issues (title, body, labels, short repo summary). Choosing what goes in is most of the craft. |
| Instructions | System prompt | Role, scoring rubric, output schema, a few examples. Rarely changes. |
| Judgment | Model | "Does this issue fit this person, and how big is it?" The only fuzzy step. |

Rule: if something can be done deterministically in code, it doesn't go to the model.

## Key decisions

- **Language:** Go
- **Profile is data, not prompt.** `profile.yaml` is deserialized into a typed struct, validated at startup, then rendered into the request. It must never be pasted into the system prompt.
- **Structured output.** The model returns JSON matching a fixed schema, parsed into a struct, e.g. `Score { issue_id, score (0–10), reason, effort }`. Parse failures are errors, not silently skipped.
- **Storage: filesystem first.**
  - `profile.yaml`: skills, interests, things to avoid, time per issue
  - `repos.json`: watched repos, last star count, score
  - `seen_issues.json`: processed issue IDs (dedup, never score twice)
  - `history.jsonl`: picked/skipped decisions, appended, fed back as memory
  - Move to SQLite only when we need cross-data queries.
- **Scheduling:** cron on my machine or a VPS. If using GitHub Actions, the runner is ephemeral, so state files must be committed back each run.
- **A skill later:** once the scoring rubric stabilizes, extract it into a reusable skill.

## Speed matters

Good issues get claimed within hours, so speed is the point of this tool:

- Pre-rank repos daily; poll only the top 20–30 for new issues every 10–15 minutes.
- Filter cheaply before any model call: unassigned, no linked PR, recent, relevant labels.
- Push an instant notification for scores ≥ 8; no daily digests for those.

## Etiquette

The tool finds issues; I write the PRs myself. Never automate commenting, claiming, or opening PRs on other people's repositories.
