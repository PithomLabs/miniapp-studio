// Package render turns admitted MiniappSpec documents into server-rendered
// HTML. It accepts only [runner.ValidatedSpec] values bound to the calling
// tenant, renders a closed component catalog with html/template
// autoescaping, and resolves bindings exclusively through a caller-supplied
// [DataSource]. The renderer fetches nothing, authorizes nothing, and
// submits nothing: data arrives scoped, actions post to caller-provided
// endpoints, and every interpolated value is escaped.
package render
