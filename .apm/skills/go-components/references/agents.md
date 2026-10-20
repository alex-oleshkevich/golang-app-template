# AI agents: ADK Go v2

`google.golang.org/adk/v2` runs LLM agents inside the app. The app owns the
model choice, conversation storage and the HTTP endpoint; ADK owns the agent
loop.

## Pieces

- `agent/llmagent`: an agent from instructions, a model and tools.
- `workflow`: v2's graph engine. Nodes are agents or plain Go steps, joined with
  `workflow.New(name, edges)`. Use it when the order of steps is known; let a
  single `llmagent` decide when it isn't.
- `tool/functiontool.New[Args, Results](cfg, fn)`: a typed tool. The JSON schema
  comes from the `Args` struct, so document each field with a `jsonschema` tag.
- `tool/toolconfirmation`: pauses a run until the user approves a tool call.
- `runner.New(runner.Config{AppName, Agent, SessionService})`, then
  `r.Run(ctx, userID, sessionID, msg, cfg)`: an `iter.Seq2[*session.Event, error]`
  of events to stream to the client.

Instructions live in `agents/<name>/instruction.md`, embedded with `go:embed`, so
prompt changes show up in diffs.

## Models

`APP_LLM_MODEL` picks the provider and model; each provider has its own key.

- `anthropic:claude-opus-5-5`: our own `model.LLM` adapter on
  `github.com/anthropics/anthropic-sdk-go`. ADK has none for Claude. It maps
  `genai.Content` to messages, tool declarations and results, the system
  instruction and stop reasons, streams, and marks the system prompt and tools
  with `cache_control` so repeated turns hit Anthropic's prompt cache. Key:
  `APP_ANTHROPIC_API_KEY`.
- `openai:<model>`: ADK's `model/openaimodel` with its default Responses API.
  Key: `APP_OPENAI_API_KEY`.
- `openrouter:<vendor>/<model>`: `openaimodel` with
  `BaseURL: "https://openrouter.ai/api/v1"` and `API: openaimodel.APIChatCompletions`,
  which is what OpenRouter serves. Key: `APP_OPENROUTER_API_KEY`.

`model.LLM` is two methods (`Name`, `GenerateContent`). `agents.OpenModel(ctx, spec)`
parses the spec and returns one; nothing else knows which provider is in use.

## Conversations: SQLite session service

ADK's `session/database` needs gorm, and the gorm SQLite driver that is pure Go
(`glebarez/sqlite`) registers the `sqlite` driver name that modernc already
holds, so the binary panics at start. Write `session.Service` ourselves instead:
`Create`, `Get`, `List`, `Delete`, `AppendEvent`, as sqlc queries on our
connection, with events stored as JSON. `sessiontestsuite.RunServiceTests` from
ADK checks it against ADK's own contract, including the rules for event ids.

Session ids are scoped to the caller: the runner's `userID` is
`"<team_id>:<user_id>"`, and `Get` and `List` only return that caller's sessions.

## Endpoint

`POST /api/v1/agents/{agent}/runs` in `api/openapi.yaml`, with `security` and
scopes like any operation. The body carries `session_id` (empty starts one) and
the message; the response is `text/event-stream` with one event per ADK event.

- The route sits outside `middleware.Timeout`: runs take longer than its 25 s. Use
  `http.NewResponseController(w).SetWriteDeadline` to extend the 30 s server
  write timeout per event, and cap the whole run with its own
  `context.WithTimeout` (5 minutes) and a maximum number of model calls.
- Flush after every event (`ResponseController.Flush`). Stop when the client's
  context is cancelled.
- The event format the UI renders is in `agent-ui.md`.

Don't use ADK's `server/adkrest` or launcher: they bring their own router and
auth, outside our spec, problem responses and teams.

## Tools

- **A tool checks permissions itself.** It runs inside the agent, not behind the
  spec's validator, so a tool that reads or changes data wraps its function with
  `agents.Tool(scope, fn)`, which requires the scope from the identity in `ctx`
  and fails the call otherwise. Every query still filters on `team_id`.
- A tool that destroys, sends or spends anything (delete, email, payment) uses
  `toolconfirmation`, so the user approves each call.
- Tool results are data for the model: return ids and short fields, never
  secrets or other users' personal data.
- Tool errors are returned to the model as results it can react to; only
  failures of the app itself end the run.

## Gotchas

- **`openaimodel` falls back to `OPENAI_API_KEY`** when `APIKey` is empty, even
  with `BaseURL` set to OpenRouter, and would send the OpenAI key there. Always
  pass the provider's key explicitly, and fail at start when it is empty.
- **`openaimodel` is marked experimental** in ADK; pin the ADK version and read
  its release notes on upgrade.
- **v2 moved the module path** to `google.golang.org/adk/v2`. Examples importing
  `google.golang.org/adk` are v1.
- **Binary size**: ADK pulls in genai, grpc and Google Cloud packages for the
  parts you import. Import subpackages, never the launchers.

## Tests

- Never call a real model in `just test`. `agentstest.Model` is a scripted
  `model.LLM` that returns fixed responses and tool calls in order, so agent and
  tool tests are deterministic and free.
- Each tool: called with an identity that has the scope it works; without it the
  call fails; another team's ids are not found.
- The session service passes `sessiontestsuite.RunServiceTests`.
- One smoke test per provider calls the real API, skipped unless `APP_TEST_LLM`
  names the model to use.
