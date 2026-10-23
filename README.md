# Issue Scout

> **Status: in development.** Nothing here works end to end yet.

Issue Scout watches newly trending GitHub repositories in the languages I work in and alerts me about open issues that fit my skills, quickly enough to pick them up before someone else does.

Code handles fetching, filtering, deduplication and alerts. An LLM is used for one step only: judging whether an issue fits my profile and how much work it is. It never comments on, claims, or opens PRs against anyone's repository. I write those myself.

Written in Go, and also a learning project in agentic development: v1 is a fixed workflow, v2 lets the model drive the search with tools, and v3 adds an independent check on its picks.
