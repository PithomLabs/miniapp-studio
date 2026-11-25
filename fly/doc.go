// Package fly is Miniapp Studio's owned Fly.io control-plane boundary. All
// Sprites and Connectors operations flow through the narrow interfaces here;
// the SDK adapter and typed REST client live behind them, and fakes back
// every test. Credentials arrive only as constructor arguments from
// server-side secret state — never from specs, prompts, logs, or responses.
package fly
