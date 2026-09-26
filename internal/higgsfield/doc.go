// Package higgsfield is the HTTP client for the Higgsfield generation API.
//
// Typical flow:
//  1. SubmitImage posts the model path and returns request_id plus status_url.
//  2. WaitForCompletion polls that status_url (never a hand-built URL) until a terminal status.
//  3. Download fetches the CDN URL from a completed response. CDN fetches are unauthenticated.
package higgsfield
