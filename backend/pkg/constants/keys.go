package constants

// // Context keys (typed to avoid collisions).
// type contextKey string

// const (
//   CtxUserID   contextKey = "user_id"
//   CtxEmail    contextKey = "email"
//   CtxPlanID   contextKey = "plan_id"
//   CtxKeyID    contextKey = "api_key_id"
// )

// // Redis key patterns.
// const (
//   RateLimitKeyPrefix = "ratelimit:"  // ratelimit:<user_id>:<YYYY-MM-DD>
//   JobLockKeyPrefix   = "jlock:"      // jlock:<job_id>
// )

// // Plan names — match seeds in migrations/000004.
// const (
//   PlanFree = "free"
//   PlanPro  = "pro"
//   PlanMax  = "max"
// )

// // Job statuses — match jobs table CHECK constraint.
// const (
//   JobStatusPending    = "pending"
//   JobStatusProcessing = "processing"
//   JobStatusDone       = "done"
//   JobStatusFailed     = "failed"
// )